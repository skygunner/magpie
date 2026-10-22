package gateway

import (
	"bytes"
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/yetone/magpie/internal/provider"
)

// A provider's quota-rest is a deadline, not a new duration after an update.
func TestQuotaRestSurvivesGatewayRestart(t *testing.T) {
	d, srv := deadChain(t, "dead-a", "dead-b")
	if err := provider.SetQuotaRest("mockdead", 18*60); err != nil {
		t.Fatal(err)
	}
	if got := postDead(t, srv); got.Code != 429 {
		t.Fatalf("initial quota refusal: %d", got.Code)
	}
	if got := d.took(); len(got) != 2 {
		t.Fatalf("initial vendor calls: %v", got)
	}
	id := "mockdead#" + provider.KeyID("dead-a")
	before, ok := restOf(id)
	if !ok || before.By != "quota" {
		t.Fatalf("initial rest: %+v, %v", before, ok)
	}

	// A process restart discards its in-memory state and starts a new gateway.
	restingUntil.Lock()
	restingUntil.m, restingUntil.note = map[string]time.Time{}, map[string]Rest{}
	restingUntil.from = ""
	restingUntil.Unlock()
	srv = New()
	after, ok := restOf(id)
	if !ok || !after.Until.Equal(before.Until) || after.Why != failQuota {
		t.Fatalf("rest after restart: %+v, %v; want %+v", after, ok, before)
	}
	if got := postDead(t, srv); got.Code != 429 {
		t.Fatalf("rested request: %d", got.Code)
	}
	if got := d.took(); len(got) != 0 {
		t.Fatalf("vendor was asked during saved rest: %v", got)
	}

	// A manual un-rest must also stay lifted after another restart.
	if !srv.Unrest(id) {
		t.Fatal("saved rest cannot be lifted")
	}
	restingUntil.Lock()
	restingUntil.m, restingUntil.note = map[string]time.Time{}, map[string]Rest{}
	restingUntil.from = ""
	restingUntil.Unlock()
	New()
	if r, ok := restOf(id); ok {
		t.Fatalf("lifted rest returned after restart: %+v", r)
	}
}

func TestSaveRestsReportsFailures(t *testing.T) {
	for _, step := range []string{"read", "write"} {
		t.Run(step, func(t *testing.T) {
			t.Setenv("XDG_CONFIG_HOME", t.TempDir())
			path := restsPath()
			dir := filepath.Dir(path)
			if err := os.MkdirAll(dir, 0700); err != nil {
				t.Fatal(err)
			}
			if step == "read" {
				if err := os.Mkdir(path, 0700); err != nil {
					t.Fatal(err)
				}
			} else {
				if runtime.GOOS == "windows" || os.Geteuid() == 0 {
					t.Skip("requires Unix directory permissions")
				}
				if err := os.Chmod(dir, 0500); err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() { os.Chmod(dir, 0700) })
			}
			var out bytes.Buffer
			prev := log.Writer()
			log.SetOutput(&out)
			t.Cleanup(func() { log.SetOutput(prev) })
			saveRests()
			if !strings.Contains(out.String(), "gateway rests: saving "+path+":") {
				t.Fatalf("failed rest %s was silent: %q", step, out.String())
			}
			if step == "read" {
				if info, err := os.Stat(path); err != nil || !info.IsDir() {
					t.Fatalf("unreadable path replaced: %v, %v", info, err)
				}
			}
		})
	}
}

func TestSaveRestsKeepsMalformedLedger(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	path := restsPath()
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	bad := []byte(`{"unfinished`)
	if err := os.WriteFile(path, bad, 0600); err != nil {
		t.Fatal(err)
	}
	saveRests()
	kept, err := filepath.Glob(path + ".bad-*")
	if err != nil || len(kept) != 1 {
		t.Fatalf("malformed ledger copies: %v, %v", kept, err)
	}
	if got, err := os.ReadFile(kept[0]); err != nil || !bytes.Equal(got, bad) {
		t.Fatalf("kept %q, %v", got, err)
	}
	var saved map[string]savedRest
	if b, err := os.ReadFile(path); err != nil || json.Unmarshal(b, &saved) != nil {
		t.Fatalf("new ledger %q, %v", b, err)
	}
}
