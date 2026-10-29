package davsync

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/yetone/magpie/internal/backup"
	"github.com/yetone/magpie/internal/provider"
)

// A backup bigger than the 64 MiB a sync reads was read up to the cap and
// handed on cut. backup.Open takes a cut backup for a damaged one
// (ErrCorrupt), so a sync rebuilt over an intact file that was merely
// bigger, or offered to replace it. The read boundary now says the file is
// too large, an error of its own that no sync, restore or Upload classifies
// as damage.

// overCap is a body that begins like a sealed backup and runs past the cap,
// as a real backup bigger than magpie reads does.
func overCap(t *testing.T) []byte {
	t.Helper()
	body := append([]byte(`{"format":"magpie-backup","version":1,"kdf":"pbkdf2-sha256","iterations":120000,"salt":"AAAA","nonce":"AAAA","data":"`), bytes.Repeat([]byte("A"), maxRead)...)
	body = append(body, '"', '}')
	if len(body) <= maxRead {
		t.Fatalf("the body is %d bytes, not over the %d cap", len(body), maxRead)
	}
	return body
}

func TestReadSaysTooLargeRatherThanCut(t *testing.T) {
	body := overCap(t)
	// Cut at the cap, as the read did: what every caller took for damage.
	if _, err := backup.Open(body[:maxRead], "horse"); !errors.Is(err, backup.ErrCorrupt) {
		t.Fatalf("a body cut at the cap reads as %v, not ErrCorrupt: the misclassification is gone from this test", err)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Write(body)
	}))
	defer srv.Close()
	d, err := newDAV(Config{URL: srv.URL + "/dav/", User: "me", Password: "pw"})
	if err != nil {
		t.Fatal(err)
	}
	data, _, err := d.get(context.Background(), version{})
	if !errors.Is(err, errFileTooLarge) {
		t.Fatalf("a WebDAV backup over the cap: %v, %d bytes, want errFileTooLarge", err, len(data))
	}
	if data != nil {
		t.Fatalf("a file too large came back cut: %d bytes", len(data))
	}
	var nb *notBackup
	if errors.As(described(Config{}, data, err), &nb) {
		t.Fatalf("too large is said as a file to replace (%q)", nb.f.What)
	}

	f, srv3 := newFakeS3(t)
	s, err := newS3(f.config(srv3))
	if err != nil {
		t.Fatal(err)
	}
	f.mu.Lock()
	f.objects["team x+y/magpie/magpie.magpie-backup"] = body
	f.etags["team x+y/magpie/magpie.magpie-backup"] = `"big"`
	f.mu.Unlock()
	if data, _, err := s.get(context.Background(), version{}); !errors.Is(err, errFileTooLarge) || data != nil {
		t.Fatalf("an S3 backup over the cap: %v, %d bytes, want errFileTooLarge", err, len(data))
	}
}

// The cap itself is read whole: a file of exactly maxRead is not too large.
func TestWriteBackupSizeLimit(t *testing.T) {
	requests := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()
	d, err := newDAV(Config{URL: srv.URL})
	if err != nil {
		t.Fatal(err)
	}
	s, err := newS3(Config{URL: "s3://bucket", Endpoint: srv.URL, User: "key", Password: "secret"})
	if err != nil {
		t.Fatal(err)
	}
	data := make([]byte, maxRead+1)
	for name, remote := range map[string]remote{"WebDAV": d, "S3": s} {
		t.Run(name, func(t *testing.T) {
			requests = 0
			if _, err := remote.put(context.Background(), data, ""); !errors.Is(err, errFileTooLarge) {
				t.Fatalf("oversized backup: %v, want errFileTooLarge", err)
			}
			if requests != 0 {
				t.Fatalf("oversized backup made %d requests", requests)
			}
			if _, err := remote.put(context.Background(), data[:maxRead], ""); err != nil {
				t.Fatalf("backup exactly at the limit: %v", err)
			}
			if requests == 0 {
				t.Fatal("backup at the limit was not sent")
			}
		})
	}
}

func TestWriteUsageSizeLimit(t *testing.T) {
	requests := make(chan struct{}, 10)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodHead {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		requests <- struct{}{}
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()
	d, err := newDAV(Config{URL: srv.URL})
	if err != nil {
		t.Fatal(err)
	}
	s, err := newS3(Config{URL: "s3://bucket", Endpoint: srv.URL, User: "key", Password: "secret"})
	if err != nil {
		t.Fatal(err)
	}
	data := make([]byte, maxRead+1)
	for name, store := range map[string]files{"WebDAV": d, "S3": s} {
		t.Run(name, func(t *testing.T) {
			if err := store.write(context.Background(), "day.usage", data); !errors.Is(err, errFileTooLarge) {
				t.Fatalf("oversized usage: %v, want errFileTooLarge", err)
			}
			select {
			case <-requests:
				t.Fatal("oversized usage made a request")
			default:
			}
			if err := store.write(context.Background(), "day.usage", data[:maxRead]); err != nil {
				t.Fatalf("usage exactly at the limit: %v", err)
			}
			select {
			case <-requests:
			default:
				t.Fatal("usage at the limit was not sent")
			}
		})
	}
}

func TestReadCappedBoundary(t *testing.T) {
	if b, err := readCapped(bytes.NewReader(make([]byte, maxRead))); err != nil || len(b) != maxRead {
		t.Fatalf("a file of exactly the cap: %v, %d bytes", err, len(b))
	}
	if b, err := readCapped(bytes.NewReader(make([]byte, maxRead+1))); !errors.Is(err, errFileTooLarge) || b != nil {
		t.Fatalf("a file a byte over the cap: %v, %d bytes", err, len(b))
	}
}

// Genuinely bad bytes are unchanged: a real backup cut short still opens as
// ErrCorrupt, and a sync still says it as a backup to rebuild or replace.
func TestCutBackupStillReadsAsCorrupt(t *testing.T) {
	full, err := backup.Seal(backup.Bundle{Version: backup.BundleVersion, Keys: true}, "horse")
	if err != nil {
		t.Fatal(err)
	}
	cut := full[:len(full)/2]
	if _, err := backup.Open(cut, "horse"); !errors.Is(err, backup.ErrCorrupt) {
		t.Fatalf("a backup cut short reads as %v, want ErrCorrupt", err)
	}
	var nb *notBackup
	if err := described(Config{}, cut, backup.ErrCorrupt); !errors.As(err, &nb) || !nb.f.Replace || nb.f.What != "cut" {
		t.Fatalf("a backup cut short: %v, want a cut file that can be replaced", err)
	}
}

// A sync whose server file is this computer's own, read whole before and
// since grown past the cap, used to take the cut read for its cached copy
// and rebuild: it kept the intact body aside as damaged and wrote over it.
// It now says too large and writes nothing.
func TestSyncSaysTooLargeRatherThanRebuilding(t *testing.T) {
	fake := &fakeDAV{files: map[string][]byte{}, etags: map[string]string{}, dirs: map[string]bool{"/dav": true}}
	srv := httptest.NewServer(fake)
	defer srv.Close()
	ctx := context.Background()
	cfg := Config{URL: srv.URL + "/dav/", User: "me", Password: "pw", Passphrase: "horse", Keys: true, Agents: true}

	a := newComputer(t)
	a.use(t)
	provider.Save(provider.Provider{ID: "deepseek", Name: "DeepSeek", Chat: "https://api.deepseek.com/v1", Key: "k1"})
	if err := Configure(cfg); err != nil {
		t.Fatal(err)
	}
	if err := Now(ctx); err != nil {
		t.Fatal(err)
	}

	whole, err := backup.Seal(backup.Bundle{Version: backup.BundleVersion, Keys: true,
		Agents: map[string]string{"big": string(bytes.Repeat([]byte("a"), 50<<20))}}, "horse")
	if err != nil {
		t.Fatal(err)
	}
	if len(whole) <= maxRead {
		t.Fatalf("the sealed backup is %d bytes, not over the %d cap", len(whole), maxRead)
	}
	// this computer read that whole file before: its cached copy is the file
	st := loadState()
	st.Sum, st.Server = sum(whole), version{ETag: `"big"`}
	saveState(st)
	remember(whole)
	fake.mu.Lock()
	fake.files[backupFile], fake.etags[backupFile] = whole, `"big"`
	fake.puts = 0
	fake.mu.Unlock()

	// something changed here, so the server's file is opened and merged
	provider.Save(provider.Provider{ID: "openai", Name: "OpenAI", Chat: "https://api.openai.com/v1", Key: "k2"})
	if err := Now(ctx); !errors.Is(err, errFileTooLarge) {
		t.Fatalf("a backup over the cap: %v, want errFileTooLarge", err)
	}
	fake.mu.Lock()
	defer fake.mu.Unlock()
	if fake.puts != 0 {
		t.Fatalf("the sync wrote over a file it couldn't read whole: %d writes", fake.puts)
	}
	if !bytes.Equal(fake.files[backupFile], whole) {
		t.Fatal("the server's file changed")
	}
}
