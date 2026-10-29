package davsync

import (
	"context"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/yetone/magpie/internal/backup"
	"github.com/yetone/magpie/internal/provider"
	"github.com/yetone/magpie/internal/settings"
)

func TestDAVInitialBlindWritesMergeAfterOverwrite(t *testing.T) {
	fake := &fakeDAV{files: map[string][]byte{}, etags: map[string]string{}, dirs: map[string]bool{"/dav": true}}
	srv := httptest.NewServer(fake)
	defer srv.Close()
	cfg := Config{URL: srv.URL + "/dav/", User: "me", Password: "pw", Passphrase: "horse", Keys: true}
	ctx := context.Background()
	a, b := newComputer(t), newComputer(t)
	a.use(t)
	provider.Save(provider.Provider{ID: "deepseek", Name: "A provider", Chat: "https://api.deepseek.com/v1", Key: "a"})
	if err := Configure(cfg); err != nil {
		t.Fatal(err)
	}
	if err := Now(ctx); err != nil {
		t.Fatal(err)
	}

	// B's initial GET observes the empty folder before A's PUT.
	fake.mu.Lock()
	delete(fake.files, backupFile)
	delete(fake.etags, backupFile)
	fake.mu.Unlock()
	b.use(t)
	if err := settings.Save(settings.Settings{Theme: "dark"}); err != nil {
		t.Fatal(err)
	}
	if err := Configure(cfg); err != nil {
		t.Fatal(err)
	}
	if err := Now(ctx); err != nil {
		t.Fatal(err)
	}

	a.use(t)
	newer := time.Now().Add(time.Hour)
	if err := os.Chtimes(provider.Path(), newer, newer); err != nil {
		t.Fatal(err)
	}
	if err := Now(ctx); err != nil {
		t.Fatal(err)
	}
	if got := ids(); len(got) != 1 || got[0] != "deepseek=a" {
		t.Fatalf("A's provider lost: %v", got)
	}
	if got := settings.Load().Theme; got != "dark" {
		t.Fatalf("B's theme not merged: %q", got)
	}
	bundle, err := backup.Open(onServer(fake, backupFile), cfg.Passphrase)
	if err != nil {
		t.Fatal(err)
	}
	if len(bundle.Providers) != 1 || bundle.Providers[0].ID != "deepseek" || bundle.Settings.Theme != "dark" {
		t.Fatalf("server did not merge both changes: %+v", bundle)
	}
}
