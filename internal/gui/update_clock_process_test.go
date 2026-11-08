package gui

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/yetone/magpie/internal/proc"
	"github.com/yetone/magpie/internal/settings"
)

// A new process keeps the last check's interval but a manual check still runs (#5528).
func TestUpdateClockAcrossProcesses(t *testing.T) {
	if mode := os.Getenv("MAGPIE_CLOCK_TEST"); mode != "" {
		u := &updater{}
		s := settings.Load()
		if mode == "configure" {
			s.UpdateEvery = 1440
			if err := settings.Save(s); err != nil {
				t.Fatal(err)
			}
			fmt.Printf("CLOCK_PATH=%s\n", updateClockFile())
		} else if mode == "manual" {
			u.check()
		} else {
			u.checkIfDue(s)
		}
		return
	}
	home := t.TempDir()
	var count atomic.Int32
	feed := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { count.Add(1); fmt.Fprint(w, `{"version":"0.1.1"}`) }))
	defer feed.Close()
	run := func(mode string) string {
		t.Helper()
		deadline, ok := t.Deadline()
		if !ok {
			deadline = time.Now().Add(time.Minute)
		}
		ctx, cancel := context.WithDeadline(context.Background(), deadline)
		defer cancel()
		cmd := proc.CommandContext(ctx, os.Args[0], "-test.run=^TestUpdateClockAcrossProcesses$")
		cmd.Env = append(os.Environ(), "HOME="+home, "USERPROFILE="+home, "XDG_CONFIG_HOME="+filepath.Join(home, "config"), "APPDATA="+filepath.Join(home, "AppData", "Roaming"), "LOCALAPPDATA="+filepath.Join(home, "AppData", "Local"), "MAGPIE_UPDATE_FEED="+feed.URL, "MAGPIE_CLOCK_TEST="+mode)
		b, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("%s: %v %s", mode, err, b)
		}
		return string(b)
	}
	configured := run("configure")
	var clock string
	for _, line := range strings.Split(configured, "\n") {
		if path, ok := strings.CutPrefix(line, "CLOCK_PATH="); ok {
			clock = strings.TrimSuffix(path, "\r")
		}
	}
	if clock == "" {
		t.Fatalf("child did not report clock path: %s", configured)
	}
	run("auto")
	run("auto")
	if got := count.Load(); got != 1 {
		t.Fatalf("relaunch inside interval made %d checks, want 1", got)
	}
	run("manual")
	if got := count.Load(); got != 2 {
		t.Fatalf("manual check count %d, want 2", got)
	}
	for _, tc := range []struct{ name, value string }{
		{"within-24h", time.Now().Add(-8 * time.Hour).UTC().Format(time.RFC3339Nano)},
		{"elapsed", time.Now().Add(-48 * time.Hour).UTC().Format(time.RFC3339Nano)},
		{"invalid", "not a timestamp"},
		{"future", time.Now().Add(48 * time.Hour).UTC().Format(time.RFC3339Nano)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := os.WriteFile(clock, []byte(tc.value), 0600); err != nil {
				t.Fatal(err)
			}
			before := count.Load()
			run("auto")
			want := before + 1
			if tc.name == "within-24h" {
				want = before
			}
			if got := count.Load(); got != want {
				t.Fatalf("%s: got %d checks, want %d", tc.name, got, want)
			}
		})
	}
	if err := os.Remove(clock); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(clock, 0700); err != nil {
		t.Fatal(err)
	}
	before := count.Load()
	run("auto")
	if got := count.Load(); got != before+1 {
		t.Fatalf("unreadable clock: got %d checks, want %d", got, before+1)
	}
}
