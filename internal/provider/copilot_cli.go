package provider

// The standalone Copilot CLI keeps its sign-in apart from the editors: who
// is signed in in ~/.copilot/config.json, the token in the system keychain
// (service copilot-cli), or in that file when the user chose plaintext. Its
// token is not traded for a session token the way an editor's is (that
// exchange answers 403): the CLI sends it as is, under its own integration.

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/tidwall/jsonc"

	"github.com/yetone/magpie/internal/appdir"
	"github.com/yetone/magpie/internal/proc"
	"github.com/yetone/magpie/internal/settings"
)

// copilotCLIHeaders are what the CLI sends with its token.
var copilotCLIHeaders = map[string]string{
	"Editor-Version":         "copilot/1.0.88",
	"Copilot-Integration-Id": "copilot-developer-cli",
	"X-GitHub-Api-Version":   "2026-07-01",
	"User-Agent":             "copilot/1.0.88",
}

// CopilotUserURL says where an account's Copilot API is; a var so tests can
// point it elsewhere.
var CopilotUserURL = "https://api.github.com/copilot_internal/user"

type copilotCLIUser struct {
	Host  string `json:"host"`
	Login string `json:"login"`
}

// copilotCLIHome is where the CLI keeps its settings.
func copilotCLIHome() string {
	if h := appdir.Getenv("COPILOT_HOME"); h != "" {
		return h
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".copilot")
}

// CopilotWSLHomes, set by internal/agent, are the homes, as magpie opens
// them, of the WSL distros that run now: magpie on Windows finds a Copilot
// CLI signed in inside one (#723, jia2). nil, or none, off Windows. A
// stopped distro isn't in it: reading its files would start it.
var CopilotWSLHomes func() []string

// copilotWSLAge is how long a WSL distro's Copilot CLI sign-in is kept as
// read: its files are read over WSL's file share, slower than a disk's.
var copilotWSLAge = 20 * time.Second

var copilotWSL struct {
	sync.Mutex
	at  time.Time
	app copilotApp
	ok  bool
}

// copilotCLILogin is the account the Copilot CLI is signed in to, with its
// token: on this machine, or, for magpie on Windows, in a WSL distro that
// runs now. A distro's token is read only from its config.json, where the
// CLI keeps it when the distro has no keyring to put it in (WSL has none
// unless one is set up): magpie can't reach a distro's keyring.
func copilotCLILogin() (copilotApp, bool) {
	if settings.AccountDiscoveryOff() {
		return copilotApp{}, false
	}
	if app, ok := copilotCLILoginIn(copilotCLIHome(), copilotCLISecret); ok {
		return app, true
	}
	if CopilotWSLHomes == nil {
		return copilotApp{}, false
	}
	c := &copilotWSL
	c.Lock()
	defer c.Unlock()
	if !c.at.IsZero() && time.Since(c.at) < copilotWSLAge {
		return c.app, c.ok
	}
	c.app, c.ok = copilotApp{}, false
	for _, home := range CopilotWSLHomes() {
		if app, ok := copilotCLILoginIn(filepath.Join(home, ".copilot"), nil); ok {
			c.app, c.ok = app, true
			break
		}
	}
	c.at = time.Now()
	return c.app, c.ok
}

// copilotCLILoginIn is the account the Copilot CLI whose settings are in
// dir is signed in to, with its token: from its config.json, else from
// secret (the keychain), when given.
func copilotCLILoginIn(dir string, secret func(string) string) (copilotApp, bool) {
	raw, err := readAccountFile(filepath.Join(dir, "config.json"))
	if err != nil {
		return copilotApp{}, false
	}
	var cfg struct {
		Last   *copilotCLIUser   `json:"lastLoggedInUser"`
		Users  []copilotCLIUser  `json:"loggedInUsers"`
		Tokens map[string]string `json:"copilotTokens"`
	}
	if json.Unmarshal(jsonc.ToJSON(raw), &cfg) != nil {
		return copilotApp{}, false
	}
	users := cfg.Users
	if cfg.Last != nil {
		users = append([]copilotCLIUser{*cfg.Last}, users...)
	}
	// a github.com account first, as before; then one an enterprise's
	// `copilot login --host <name>.ghe.com` signed in (#723). A GitHub
	// Enterprise Server's own host is left out: its API isn't GHE.com's.
	for _, ghe := range []bool{false, true} {
		for _, u := range users {
			host, ok := copilotHostOf(u.Host)
			if u.Login == "" || !ok || (host != "") != ghe {
				continue
			}
			key := "https://github.com:" + u.Login
			if ghe {
				key = strings.TrimSuffix(u.Host, "/") + ":" + u.Login
			}
			tok := cfg.Tokens[key]
			if tok == "" && secret != nil {
				tok = secret(key)
			}
			if tok != "" {
				return copilotApp{User: u.Login, Token: tok, Host: host, cli: true}, true
			}
		}
	}
	return copilotApp{}, false
}

var copilotSecrets struct {
	sync.Mutex
	at  map[string]time.Time
	tok map[string]string
}

// copilotAsk is held while `security` may put a prompt up: one at a time,
// and a reader that can't have it doesn't wait out a dialog (#3058).
var copilotAsk sync.Mutex

// copilotCLISecret reads the CLI's token from the keychain, once in a
// while: each read may ask the user to allow it. Tests stand in for it, so
// no test reaches the runner's own keychain.
var copilotCLISecret = copilotCLISecretFrom

// copilotCLISecretFrom is the read behind copilotCLISecret: a token kept
// an hour, a refusal kept until the CLI's sign-in changes, and no lock
// every reader takes held over the process (#3058).
func copilotCLISecretFrom(account string) string {
	if settings.AccountDiscoveryOff() {
		return ""
	}
	c := &copilotSecrets
	c.Lock()
	if c.at == nil {
		c.at, c.tok = map[string]time.Time{}, map[string]string{}
	}
	tok, at := c.tok[account], c.at[account]
	c.Unlock()
	ttl := time.Hour
	if tok == "" {
		ttl = keychainDeniedTTL // denied or absent; don't ask again at once
	}
	if !at.IsZero() && time.Since(at) < ttl {
		return tok
	}
	// the user denied the prompt for the sign-in the CLI has now: that
	// stands until the CLI's sign-in changes, not for ten minutes (#3058)
	mark := copilotCLISignInMark()
	if copilotKeychainDenied(account, mark) {
		c.Lock()
		c.at[account], c.tok[account] = time.Now(), ""
		c.Unlock()
		return ""
	}
	if !copilotAsk.TryLock() {
		// a prompt is up: the kept read stands, nobody waits on the dialog
		return tok
	}
	defer copilotAsk.Unlock()
	c.Lock()
	// the ask this one was behind landed: its answer stands
	if moved := c.at[account]; !moved.Equal(at) {
		tok = c.tok[account]
		c.Unlock()
		return tok
	}
	c.Unlock()
	out, err := copilotKeychainRead(account)
	tok = ""
	if err == nil {
		tok = strings.TrimSpace(string(out))
	}
	c.Lock()
	c.at[account], c.tok[account] = time.Now(), tok
	c.Unlock()
	switch {
	case err == nil:
		copilotKeychainForgetDenied(account)
	case errors.Is(err, errKeychainRefused):
		copilotKeychainDeny(account, mark)
	}
	return tok
}

// copilotCLIKeychainService is the keychain item the CLI keeps its token
// in; its account names the sign-in it is for.
const copilotCLIKeychainService = "copilot-cli"

// copilotKeychainRead is the CLI's token as the platform keeps it: macOS'
// `security`, or the Secret Service. A prompt the user answers after a
// while still reads, and one nobody answers is bounded (#3058). Tests put
// a fake in it, so no test reaches the runner's own keychain.
var copilotKeychainRead = func(account string) ([]byte, error) {
	switch runtime.GOOS {
	case "darwin":
		return securityKeychain{}.Find(copilotCLIKeychainService, account)
	case "linux":
		if p, err := exec.LookPath("secret-tool"); err == nil {
			return []byte(copilotSecretTool(p, account)), nil
		}
	}
	return nil, fs.ErrNotExist
}

// copilotKeychainPath is where the user's refusal to read the item is
// kept: a dialog answered Deny outlives the process, rather than asking
// again every ten minutes and after every restart (#3058).
func copilotKeychainPath() string {
	return filepath.Join(filepath.Dir(Path()), "copilot-keychain.json")
}

// copilotCLISignInMark is the CLI's sign-in as the item was read for it.
// `copilot login` and `copilot logout` write config.json, so a refusal for
// the sign-in before that one no longer stands.
func copilotCLISignInMark() string {
	st, err := os.Stat(filepath.Join(copilotCLIHome(), "config.json"))
	if err != nil {
		return ""
	}
	return strconv.FormatInt(st.ModTime().UnixNano(), 10) + "-" + strconv.FormatInt(st.Size(), 10)
}

// copilotKeychainDenied says the user refused the item for account while
// the CLI's sign-in was mark. A refusal for another sign-in is dropped:
// signing in again is the user asking again.
func copilotKeychainDenied(account, mark string) bool {
	denied := copilotKeychainDenials()
	m, ok := denied[account]
	if !ok {
		return false
	}
	if m == mark {
		return true
	}
	delete(denied, account)
	copilotKeychainWriteDenials(denied)
	return false
}

// copilotKeychainDeny keeps the user's answer for the sign-in it was given
// for.
func copilotKeychainDeny(account, mark string) {
	denied := copilotKeychainDenials()
	denied[account] = mark
	copilotKeychainWriteDenials(denied)
}

// copilotKeychainForgetDenied drops a refusal kept for account: the item
// was read, so there is nothing left to keep.
func copilotKeychainForgetDenied(account string) {
	denied := copilotKeychainDenials()
	if _, ok := denied[account]; !ok {
		return
	}
	delete(denied, account)
	copilotKeychainWriteDenials(denied)
}

// forgetCopilotKeychain has the item asked again, refusal or not: a
// sign-in or out in magpie is the user asking.
func forgetCopilotKeychain() {
	copilotSecrets.Lock()
	copilotSecrets.at, copilotSecrets.tok = map[string]time.Time{}, map[string]string{}
	copilotSecrets.Unlock()
	os.Remove(copilotKeychainPath())
}

// copilotKeychainDenials is the refusals kept, by account. A file that
// can't be read or parsed is no refusal known, so the item is asked once
// and the answer kept again: never a refusal the user didn't give.
func copilotKeychainDenials() map[string]string {
	var f struct {
		Denied map[string]string `json:"denied"`
	}
	if b, err := readAccountFile(copilotKeychainPath()); err == nil {
		json.Unmarshal(b, &f)
	}
	if f.Denied == nil {
		f.Denied = map[string]string{}
	}
	return f.Denied
}

func copilotKeychainWriteDenials(denied map[string]string) {
	if len(denied) == 0 {
		os.Remove(copilotKeychainPath())
		return
	}
	b, _ := json.Marshal(struct {
		Denied map[string]string `json:"denied"`
	}{denied})
	// a refusal that can't be kept on disk still holds this run through
	_ = writePrivate(copilotKeychainPath(), b)
}

// copilotSecretWait bounds one secret-tool lookup: a keyring that waits on
// a prompt nobody answers would hold the sign-in that asked (#723).
var copilotSecretWait = 5 * time.Second

// copilotSecretTool is the CLI's token in the Secret Service, as
// secret-tool (at path) finds it. The CLI before 1.0 (keytar, libsecret)
// keyed it by service and account; 1.0's own keyring store
// (zbus-secret-service-keyring-store) by service and username, so both
// are asked (#723).
func copilotSecretTool(path, account string) string {
	for _, attr := range []string{"account", "username"} {
		ctx, cancel := context.WithTimeout(context.Background(), copilotSecretWait)
		cmd := proc.CommandContext(ctx, path, "lookup", "service", "copilot-cli", attr, account)
		cmd.WaitDelay = time.Second
		out, _ := cmd.Output()
		cancel()
		if tok := strings.TrimSpace(string(out)); tok != "" {
			return tok
		}
	}
	return ""
}

// copilotDirect is the session a CLI token makes: the token itself, at the
// API its account is served from.
func copilotDirect(ctx context.Context, app copilotApp) (copilotSession, error) {
	github := app.Token
	copilotMu.Lock()
	defer copilotMu.Unlock()
	if s, ok := copilotSessions[github]; ok && time.Until(time.Unix(s.ExpiresAt, 0)) > 2*time.Minute {
		return s, nil
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, copilotUserURL(app.Host), nil)
	if err != nil {
		return copilotSession{}, err
	}
	req.Header.Set("Authorization", "token "+github)
	req.Header.Set("Accept", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return copilotSession{}, errors.New("Copilot sign-in: " + err.Error())
	}
	defer res.Body.Close()
	b, _ := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	var u struct {
		Endpoints struct {
			API string `json:"api"`
		} `json:"endpoints"`
	}
	if res.StatusCode != 200 || json.Unmarshal(b, &u) != nil {
		// 401 is GitHub refusing the token itself; a 403 here can be an
		// account without Copilot, and a 5xx is GitHub's own trouble
		if res.StatusCode == http.StatusUnauthorized {
			copilotRefuse(github, res.StatusCode)
		}
		return copilotSession{}, errors.New("the Copilot CLI's sign-in was refused (" + APIError(b, res.Status) + "); run `copilot login` again")
	}
	copilotRefusals.Lock()
	delete(copilotRefusals.at, github)
	delete(copilotRefusals.status, github)
	copilotRefusals.Unlock()
	s := copilotSession{Token: github, ExpiresAt: time.Now().Add(30 * time.Minute).Unix(), direct: true}
	s.Endpoints.API = u.Endpoints.API
	copilotSessions[github] = s
	return s, nil
}
