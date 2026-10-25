package pii

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"

	"github.com/yetone/magpie/internal/settings"
)

// #3561 (yuancjun): the privacy model's downloads built their client at
// package init, cloning http.DefaultTransport before netproxy.Install gave
// it Settings' Proxy - so on a proxy-only network the feature never turned
// on (TLS handshake timeout, then dial timeout, both direct). The client
// asks netproxy.Func now, as update's downloader already did. The test is
// the report's network: the only way to the "release host" is the proxy,
// and a download that bypasses it finds no server at all.
func TestDownloadFollowsTheSettingsProxy(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("XDG_CONFIG_HOME", home+"/.config")
	t.Setenv("XDG_CACHE_HOME", home+"/.cache")

	var zipBody bytes.Buffer
	zw := zip.NewWriter(&zipBody)
	w, err := zw.Create("pf-serve")
	if err != nil {
		t.Fatal(err)
	}
	w.Write([]byte("serve"))
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}

	var viaProxy atomic.Int64
	proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		viaProxy.Add(1)
		w.Write(zipBody.Bytes())
	}))
	t.Cleanup(proxy.Close)

	cfg := settings.Load()
	cfg.Proxy = proxy.URL
	if err := settings.Save(cfg); err != nil {
		t.Fatal(err)
	}

	oldBase, oldTag := serveBase, serveTag
	serveBase, serveTag = "http://releases.example.test/download/", "pf-v1"
	t.Cleanup(func() { serveBase, serveTag = oldBase, oldTag })

	path := filepath.Join(t.TempDir(), "pf.zip")
	sum := sha256.Sum256(zipBody.Bytes())
	if err := fetch(context.Background(), serveBase+serveTag+"/pf-serve-x.zip", path, -1, hex.EncodeToString(sum[:]), new(atomic.Int64)); err != nil {
		t.Fatalf("fetch: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
	if viaProxy.Load() == 0 {
		t.Fatal("the download went direct: the proxy saw nothing, and the fake host has no other way in")
	}
}
