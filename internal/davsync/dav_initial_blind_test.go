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

func TestDAVInitialBlindWritesPreserveBothAfterOverwrite(t *testing.T) {
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
	older := time.Now().Add(-time.Hour)
	if err := os.Chtimes(provider.Path(), older, older); err != nil {
		t.Fatal(err)
	}
	err := Now(ctx)
	if got := ids(); len(got) != 1 || got[0] != "deepseek=a" {
		t.Fatalf("A's provider lost: %v", got)
	}
	if err == nil {
		t.Fatal("competing initial writes must stop for reconciliation")
	}
	bundle, err := backup.Open(onServer(fake, backupFile), cfg.Passphrase)
	if err != nil {
		t.Fatal(err)
	}
	if len(bundle.Providers) != 0 || bundle.Settings.Theme != "dark" {
		t.Fatalf("server's competing write was changed: %+v", bundle)
	}
	if err := Now(ctx); err == nil {
		t.Fatal("retry must preserve the unresolved conflict")
	}
	if got := ids(); len(got) != 1 || got[0] != "deepseek=a" {
		t.Fatalf("A's provider lost on retry: %v", got)
	}
}
