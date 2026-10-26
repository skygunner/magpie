package plugin

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// pruneDist took out a tarball plugins/package.json no longer names while
// bun.lock's packages section still names it (#3622, gkd2323c): bun
// resolves that section on the next add, a tarball it names that isn't
// there fails the add, and one stale entry fails every plugin's install
// and update, whatever it is adding. A tarball the lock still names stays
// until the lock stops naming it too.
func TestDistPruneKeepsWhatTheLockNames(t *testing.T) {
	sandbox(t)
	const pkg = "@magpie-community/opencode-zed-auth"
	const other = "@magpie-community/opencode-workbuddy-auth"
	tgz := func(v string) string { return "magpie-community-opencode-zed-auth-" + v + ".tgz" }
	// npm has zed nothing (it comes from the dist branch), workbuddy as
	// npm has it
	heldRegistry(t, other, map[string]time.Time{"0.1.34": time.Now().Add(-72 * time.Hour)}, "0.1.34")
	fakeDist(t, map[string][]string{pkg: {"0.1.3", "0.1.4"}}, false)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	if _, err := Add(ctx, pkg); err != nil {
		t.Fatalf("Add: %v", err)
	}
	if err := installAt(ctx, pkg, "0.1.3"); err != nil {
		t.Fatalf("installAt 0.1.3: %v", err)
	}
	// the drift the issue reports: package.json moved on to 0.1.4 while
	// the lock's packages entry kept 0.1.3, whose tarball is on disk
	pj := filepath.Join(Dir(), "package.json")
	b, err := os.ReadFile(pj)
	if err != nil {
		t.Fatal(err)
	}
	b = bytes.ReplaceAll(b, []byte(tgz("0.1.3")), []byte(tgz("0.1.4")))
	if err := os.WriteFile(pj, b, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(Dir(), "dist", tgz("0.1.4")), npmTarball(t, pkg, "0.1.4"), 0o644); err != nil {
		t.Fatal(err)
	}

	pruneDist()
	if _, err := os.Stat(filepath.Join(Dir(), "dist", tgz("0.1.3"))); err != nil {
		t.Fatalf("pruneDist took out the tarball bun.lock still names: %v", err)
	}
	bun, err := Bun(ctx)
	if err != nil {
		t.Fatal(err)
	}
	// an add of a plugin the drift doesn't touch, as the issue's update of
	// three unrelated ones was
	if err := bunAdd(ctx, bun, other+"@0.1.34"); err != nil {
		t.Fatalf("an add of an unrelated plugin fails while the lock names a tarball that is gone: %v", err)
	}
}
