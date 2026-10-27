package testenv

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Every package's tests install Offline from their TestMain, through
// testenv's Run (Main and RunIn call it too) or by calling it themselves, as
// internal/gateway does. One that doesn't lets a test ask a vendor or the
// machine's cached catalog, and then pass or fail differently per machine
// (#2661).
func TestEveryPackageStaysOffTheNetwork(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	dirs := map[string][]string{}
	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if path != root && (strings.HasPrefix(d.Name(), ".") || d.Name() == "node_modules") {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(path, "_test.go") {
			dirs[filepath.Dir(path)] = append(dirs[filepath.Dir(path)], path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	for dir, files := range dirs {
		rel, _ := filepath.Rel(root, dir)
		main := ""
		for _, f := range files {
			b, err := os.ReadFile(f)
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(string(b), "func TestMain(m *testing.M)") {
				main = string(b)
				break
			}
		}
		switch {
		case main == "":
			t.Errorf("%s: the package's tests have no TestMain; add func TestMain(m *testing.M) { testenv.Main(m) }", rel)
		case !installsOffline(main):
			t.Errorf("%s: TestMain doesn't install testenv.Offline, so its tests reach the network", rel)
		}
	}
}

// installsOffline says whether a test source installs Offline: a TestMain
// through testenv's Run, Main or RunIn, or a call of Offline itself.
func installsOffline(src string) bool {
	for _, call := range []string{"testenv.Run(", "testenv.RunIn(", "testenv.Main(", "testenv.Offline(", "Offline()"} {
		if strings.Contains(src, call) {
			return true
		}
	}
	return false
}
