package agent

import (
	"path/filepath"
	"testing"

	"github.com/tidwall/gjson"
	"github.com/yetone/magpie/internal/edit"
	"github.com/yetone/magpie/internal/gateway"
)

// li1how, #2768: a Remote - WSL window's Copilot Chat runs in the extension
// host inside the distro, so under NAT its requests leave from there, at an
// address that isn't Windows' 127.0.0.1, and with a key the gateway takes
// from there (the WSL key). The group magpie writes has that pair in the
// default profile and each other one, and a sync follows a distro that
// changes how it reaches Windows; a distro in mirrored networking, whose
// 127.0.0.1 is Windows', keeps the loopback pair. A distro whose only sign
// of VS Code is its server folder is looked at for this.
func TestVSCodeChatInWSLAsksWindowsAsItsDistroReachesIt(t *testing.T) {
	home := syncHome(t)
	fakeWSLNet(t, "172.20.0.1/20")
	roots := map[string]string{"Ubuntu": filepath.Join(home, "wsl")}
	restore := FakeWSL(map[string]string{
		"Ubuntu": "home:/home/me\ndir:.vscode-server\nroute:default via 172.20.0.1 dev eth0\nnet:nat\n",
	}, roots)
	t.Cleanup(restore)

	dir := filepath.Join(home, "vscode", "User")
	a := vscodeAt(dir)
	writeFile(t, a.Path, "{\n\t\"chat.defaultModel\": \"gpt-5\"\n}\n")
	lm := filepath.Join(dir, "chatLanguageModels.json")
	writeFile(t, lm, "[]\n")
	work := filepath.Join(dir, "profiles", "-5f1c2a")
	wlm := filepath.Join(work, "chatLanguageModels.json")
	writeFile(t, filepath.Join(dir, "globalStorage", "storage.json"), `{"userDataProfiles": [{"location": "-5f1c2a", "name": "Work"}]}`)
	writeFile(t, filepath.Join(work, "settings.json"), "{\n\t\"chat.defaultModel\": \"gpt-5\"\n}\n")
	writeFile(t, wlm, "[]\n")

	if err := a.Connect(); err != nil {
		t.Fatal(err)
	}
	nat := "http://172.20.0.1:" + gateway.Port() + "/v1/chat/completions"
	key := gateway.WSLKey()
	if key == gateway.Token {
		t.Fatalf("no WSL key: %q", key)
	}
	for _, f := range []string{lm, wlm} {
		g, ok := edit.GetJSONItem(f, vscodeGroup)
		if !ok {
			t.Fatalf("no group at %s:\n%s", f, readFile(f))
		}
		if got := gjson.Get(g, "models.0.url").String(); got != nat {
			t.Errorf("%s: url %q, want %q", f, got, nat)
		}
		if got := gjson.Get(g, "models.0.requestHeaders.Authorization").String(); got != "Bearer "+key {
			t.Errorf("%s: key %q, want the WSL key", f, got)
		}
	}
	if c := a.Check(); c != "" {
		t.Fatalf("check: %q", c)
	}

	// the same distro in mirrored networking: 127.0.0.1 is Windows' there,
	// so the sync puts the loopback pair back
	restore()
	restore = FakeWSL(map[string]string{
		"Ubuntu": "home:/home/me\ndir:.vscode-server\nnet:mirrored\n",
	}, roots)
	t.Cleanup(restore)
	if err := a.Sync(); err != nil {
		t.Fatal(err)
	}
	g, _ := edit.GetJSONItem(lm, vscodeGroup)
	if got, want := gjson.Get(g, "models.0.url").String(), vscodeURL(); got != want {
		t.Fatalf("mirrored: url %q, want %q", got, want)
	}
	if got, want := gjson.Get(g, "models.0.requestHeaders.Authorization").String(), "Bearer "+gateway.Token; got != want {
		t.Fatalf("mirrored: key %q, want %q", got, want)
	}
	if c := a.Check(); c != "" {
		t.Fatalf("mirrored check: %q", c)
	}
}
