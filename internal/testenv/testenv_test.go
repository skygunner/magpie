package testenv

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

// A stand-in is the program a lookup finds, before an installed one of the
// same name further down PATH, on Windows too, where a file with no
// extension isn't a program: the gateway's tests make theirs with
// StandIns, and read the user's own Codex's version through it (#1827).
func TestStandInsComeBeforeTheInstalled(t *testing.T) {
	stand, installed := t.TempDir(), t.TempDir()
	if err := StandIns(stand, []string{"codex"}); err != nil {
		t.Fatal(err)
	}
	real := filepath.Join(installed, "codex")
	body := "#!/bin/sh\necho codex-cli 0.162.0\n"
	if runtime.GOOS == "windows" {
		real, body = real+".cmd", "@echo codex-cli 0.162.0\r\n"
	}
	if err := os.WriteFile(real, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", stand+string(os.PathListSeparator)+installed)
	got, err := exec.LookPath("codex")
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Dir(got) != stand {
		t.Fatalf("codex is %s, not the stand-in in %s", got, stand)
	}
	if out, err := exec.Command(got).Output(); err == nil || len(out) > 0 {
		t.Fatalf("the stand-in ran: %q, %v", out, err)
	}
}

func TestIsolate(t *testing.T) {
	for _, k := range append(folderVars, "CODEX_HOME", "PATH", Marker) {
		t.Setenv(k, os.Getenv(k)) // restored after the test
	}
	os.Setenv("CODEX_HOME", filepath.Join(t.TempDir(), "codex"))
	t.Setenv("HTTPS_PROXY", "http://127.0.0.1:9") // the machine's own, which net/http would freeze into its first read
	home, err := Isolate()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { remove(home) })
	for _, k := range proxyVars {
		if _, ok := os.LookupEnv(k); ok {
			t.Errorf("%s is still set", k)
		}
	}
	for _, f := range []func() (string, error){os.UserHomeDir, os.UserConfigDir, os.UserCacheDir} {
		if p, err := f(); err != nil || !within(p, home) {
			t.Errorf("%q (%v) is outside %q", p, err, home)
		}
	}
	if _, ok := os.LookupEnv("CODEX_HOME"); ok {
		t.Error("CODEX_HOME is still set")
	}
	remove(home)
	if _, err := os.Stat(home); !os.IsNotExist(err) {
		t.Errorf("the test home is still there: %v", err)
	}
}

// A binary started again by a test keeps that test's homes, all in the
// temporary folder; one pointing anywhere else is refused.
func TestCheckChild(t *testing.T) {
	sandbox, fixture := t.TempDir(), t.TempDir()
	for _, k := range append(folderVars, "CODEX_HOME") {
		t.Setenv(k, "")
	}
	SetHome(t, fixture)
	t.Setenv("CODEX_HOME", filepath.Join(fixture, ".codex"))
	if err := checkChild(sandbox); err != nil {
		t.Errorf("a fixture home in the temporary folder: %v", err)
	}
	outside, _ := filepath.Abs(filepath.Join(filepath.VolumeName(sandbox)+string(filepath.Separator), "not-temp", "home"))
	t.Setenv("CODEX_HOME", outside)
	if err := checkChild(sandbox); err == nil {
		t.Error("CODEX_HOME outside the temporary folder passed")
	}
	t.Setenv("CODEX_HOME", "")
	if err := checkChild(outside); err == nil {
		t.Error("a sandbox outside the temporary folder passed")
	}
}

// A test that switches home takes every per-user base dir with it: the XDG
// folders Go reads on Linux, and Windows' APPDATA and LOCALAPPDATA. Left at
// the package's own home, one test reads the files another wrote there
// (#1841).
func TestSetHomePointsEveryUserBaseDirIntoTheTestHome(t *testing.T) {
	pkg, dir := t.TempDir(), t.TempDir()
	// the package's home, standing as testenv.Run leaves it
	for k, v := range map[string]string{
		"HOME": pkg, "USERPROFILE": pkg,
		"XDG_CONFIG_HOME": filepath.Join(pkg, ".config"),
		"XDG_CACHE_HOME":  filepath.Join(pkg, ".cache"),
		"XDG_DATA_HOME":   filepath.Join(pkg, ".local", "share"),
		"XDG_STATE_HOME":  filepath.Join(pkg, ".local", "state"),
		"APPDATA":         filepath.Join(pkg, "AppData", "Roaming"),
		"LOCALAPPDATA":    filepath.Join(pkg, "AppData", "Local"),
	} {
		t.Setenv(k, v)
	}
	SetHome(t, dir)
	for _, f := range []struct {
		name   string
		folder func() (string, error)
	}{
		{"home", os.UserHomeDir}, {"config", os.UserConfigDir}, {"cache", os.UserCacheDir},
	} {
		if p, err := f.folder(); err != nil || !within(p, dir) {
			t.Errorf("the %s folder %q (%v) is outside the test home %s", f.name, p, err, dir)
		}
	}
	for _, k := range []string{"APPDATA", "LOCALAPPDATA", "XDG_CONFIG_HOME", "XDG_CACHE_HOME", "XDG_DATA_HOME", "XDG_STATE_HOME"} {
		if v := os.Getenv(k); !within(v, dir) {
			t.Errorf("%s is %q, outside the test home %s", k, v, dir)
		}
	}
}

func TestRemoveTouchesOnlyItsOwn(t *testing.T) {
	dir := t.TempDir()
	other := filepath.Join(dir, prefix+"x")
	os.MkdirAll(other, 0o755)
	remove(other) // not in os.TempDir() itself
	remove("")
	remove(".")
	if _, err := os.Stat(other); err != nil {
		t.Errorf("remove deleted a folder it didn't make: %v", err)
	}
}
