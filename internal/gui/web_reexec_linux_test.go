package gui

import (
	"context"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"syscall"
	"testing"
	"time"

	"github.com/yetone/magpie/internal/plugin"
	"github.com/yetone/magpie/internal/proc"
	"github.com/yetone/magpie/internal/testenv"
)

// The new image has the same PID but none of the old image's Wait goroutines.
func TestWebReexecReapsPluginHost(t *testing.T) {
	const stage = "MAGPIE_TEST_WEB_REEXEC"
	switch os.Getenv(stage) {
	case "start":
		dir := os.Getenv("MAGPIE_TEST_REEXEC_DIR") // the parent owns cleanup across exec
		pidPath := filepath.Join(dir, "pid")
		t.Setenv("MAGPIE_TEST_HOST_PID", pidPath)
		t.Setenv("MAGPIE_BUN", testenv.Bun(t))
		t.Setenv("MAGPIE_PLUGIN_MARKET", "off")
		spec := filepath.Join(dir, "index.js")
		if err := os.WriteFile(spec, []byte(`import { writeFileSync } from "node:fs";
export const Fixture = async () => {
  writeFileSync(process.env.MAGPIE_TEST_HOST_PID, String(process.pid));
  if (process.env.MAGPIE_TEST_HOST_STUCK === "1") {
    process.exit = () => {};
    setInterval(() => {}, 1000);
  }
  return {};
};
`), 0o600); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(plugin.Settle)
		if _, err := plugin.Add(t.Context(), spec); err != nil {
			t.Fatal(err)
		}
		if _, err := plugin.Providers(t.Context()); err != nil {
			t.Fatal(err)
		}
		if _, err := os.ReadFile(pidPath); err != nil {
			t.Fatal(err)
		}
		serving.Add(1)
		go func() {
			defer serving.Done()
			<-backendCtx.Done()
			if err := os.WriteFile(pidPath+".drained", nil, 0o600); err != nil {
				t.Error(err)
			}
		}()
		t.Setenv(stage, "after")
		webReexec.Store(true)
		w := &Web{srv: &http.Server{}, quit: make(chan struct{})}
		close(w.quit)
		ctx := context.Background()
		if os.Getenv("MAGPIE_TEST_WEB_CANCEL") == "1" {
			var cancel context.CancelFunc
			ctx, cancel = context.WithCancel(ctx)
			cancel()
			if err := w.Wait(ctx); err != nil {
				t.Fatal(err)
			}
			checkWebHostReaped(t)
			return
		}
		t.Fatalf("reexec returned: %v", w.Wait(ctx))
	case "after":
		if os.Getenv("MAGPIE_TEST_WEB_CANCEL") == "1" {
			t.Fatal("a cancelled Web.Wait restarted the process")
		}
		checkWebHostReaped(t)
	default:
		testenv.Bun(t) // fail with an explicit skip if the real host cannot run
		exe, err := os.Executable()
		if err != nil {
			t.Fatal(err)
		}
		ctx, cancel := context.WithTimeout(t.Context(), time.Minute)
		defer cancel()
		for _, mode := range []string{"normal", "stuck", "cancel"} {
			stuck, cancelled := "0", "0"
			if mode == "stuck" {
				stuck = "1"
			}
			if mode == "cancel" {
				cancelled = "1"
			}
			cmd := proc.CommandContext(ctx, exe, "-test.run=^TestWebReexecReapsPluginHost$", "-test.v")
			cmd.Env = append(os.Environ(), stage+"=start", "MAGPIE_TEST_HOST_STUCK="+stuck, "MAGPIE_TEST_WEB_CANCEL="+cancelled, "MAGPIE_TEST_REEXEC_DIR="+t.TempDir())
			if out, err := cmd.CombinedOutput(); err != nil {
				t.Errorf("web shutdown (%s): %v\n%s", mode, err, out)
			}
		}
	}
}

func checkWebHostReaped(t *testing.T) {
	t.Helper()
	if _, err := os.Stat(os.Getenv("MAGPIE_TEST_HOST_PID") + ".drained"); err != nil {
		t.Error("gateway backend was not drained before reexec:", err)
	}
	b, err := os.ReadFile(os.Getenv("MAGPIE_TEST_HOST_PID"))
	if err != nil {
		t.Fatal(err)
	}
	pid, err := strconv.Atoi(string(b))
	if err != nil || pid <= 0 {
		t.Fatalf("host PID %q: %v", b, err)
	}
	var status syscall.WaitStatus
	n, err := syscall.Wait4(pid, &status, syscall.WNOHANG, nil)
	if !errors.Is(err, syscall.ECHILD) {
		if n == 0 { // clean up the old-code failure, too
			_ = syscall.Kill(pid, syscall.SIGKILL)
			_, _ = syscall.Wait4(pid, &status, 0, nil)
		}
		t.Fatalf("plugin host was not reaped before reexec: wait4 = %d, %v", n, err)
	}
}
