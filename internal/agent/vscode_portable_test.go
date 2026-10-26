package agent

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/yetone/magpie/internal/proc"
)

// A VS Code installed portable (Scoop's vscode: bin on PATH, data persisted
// beside it) keeps its User folder in data/user-data, and that is the folder
// magpie must write: the %APPDATA%\Code\User it computed before does not
// exist, or is the one Scoop copied and the running editor never reads
// (#3260).
func TestVSCodePortableUserDir(t *testing.T) {
	for _, tt := range []struct {
		name            string
		portable, plain bool
		want            string
	}{
		{name: "portable only", portable: true, want: "portable"},
		{name: "portable beside a copied plain folder", portable: true, plain: true, want: "portable"},
		{name: "plain only", plain: true, want: "plain"},
		{name: "neither", want: "plain"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			app := filepath.Join(root, "scoop", "apps", "vscode", "current")
			binDir := filepath.Join(app, "bin")
			if err := os.MkdirAll(binDir, 0o755); err != nil {
				t.Fatal(err)
			}
			name := "code"
			if runtime.GOOS == "windows" {
				name += ".cmd"
				t.Setenv("PATHEXT", ".CMD")
			}
			bin := filepath.Join(binDir, name)
			writeFile(t, bin, "not a runnable editor\n")
			if err := os.Chmod(bin, 0o755); err != nil {
				t.Fatal(err)
			}
			t.Setenv("PATH", binDir)
			proc.Forget()

			home := t.TempDir()
			cfg := filepath.Join(root, "config")
			plain := filepath.Join(cfg, "Code", "User")
			switch runtime.GOOS {
			case "darwin":
				plain = filepath.Join(home, "Library", "Application Support", "Code", "User")
			case "windows":
				appdata := filepath.Join(root, "appdata")
				t.Setenv("APPDATA", appdata)
				plain = filepath.Join(appdata, "Code", "User")
			}
			portable := filepath.Join(app, "data", "user-data", "User")
			for _, d := range []struct {
				on   bool
				path string
			}{{tt.portable, portable}, {tt.plain, plain}} {
				if !d.on {
					continue
				}
				if err := os.MkdirAll(d.path, 0o755); err != nil {
					t.Fatal(err)
				}
			}

			a := vscode(home, cfg)
			want := plain
			if tt.want == "portable" {
				want = portable
			}
			if a.Dir != want {
				t.Errorf("Dir = %q, want %q", a.Dir, want)
			}
			if a.Path != filepath.Join(want, "settings.json") {
				t.Errorf("Path = %q, want settings.json in %q", a.Path, want)
			}
			if tt.portable && !a.Detected() {
				t.Errorf("Detected() = false, want true for a portable VS Code")
			}
		})
	}
}
