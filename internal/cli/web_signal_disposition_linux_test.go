//go:build linux

package cli

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"testing"
	"time"

	"github.com/yetone/magpie/internal/proc"
)

// A separate process owns the signal dispositions these checks change.
func TestWebContextKeepsIgnoredSignals(t *testing.T) {
	if os.Getenv("MAGPIE_TEST_SIGNAL_DISPOSITION") == "1" {
		sigs := []os.Signal{os.Interrupt, syscall.SIGTERM, syscall.SIGHUP, syscall.SIGUSR1}
		signal.Ignore(sigs...)
		endProbesOnSignal()
		if len(caught) != 0 {
			t.Fatalf("ignored stop signals were caught: %v", caught)
		}
		ctx, stop := webContext()
		defer stop()
		for _, sig := range sigs {
			if !signal.Ignored(sig) {
				t.Errorf("%v is no longer ignored", sig)
			}
		}
		if ctx.Err() != nil {
			t.Fatalf("web context canceled: %v", ctx.Err())
		}
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	cmd := proc.CommandContext(ctx, os.Args[0], "-test.run=^TestWebContextKeepsIgnoredSignals$")
	cmd.Env = append(os.Environ(), "MAGPIE_TEST_SIGNAL_DISPOSITION=1")
	out, err := cmd.CombinedOutput()
	if ctx.Err() != nil {
		t.Fatalf("signal-disposition child timed out: %v\n%s", ctx.Err(), out)
	}
	if err != nil {
		t.Fatalf("signal-disposition child failed: %v\n%s", err, out)
	}
}
