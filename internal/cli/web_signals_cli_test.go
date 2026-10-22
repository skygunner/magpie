//go:build !windows

package cli

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/yetone/magpie/internal/gateway"
	"github.com/yetone/magpie/internal/proc"
	"github.com/yetone/magpie/internal/provider"
	"github.com/yetone/magpie/internal/testenv"
)

// Run the web command with main's signal handler, not just Web.Wait:
// the global probe cleanup used to kill it before it could shut down.
func TestWebShutdown(t *testing.T) {
	if mode := os.Getenv("MAGPIE_TEST_WEB_SHUTDOWN"); mode != "" {
		if strings.HasPrefix(mode, "ignored-both") {
			// Go preserves inherited SIGINT, but installs its own SIGTERM
			// handler. Exercise both ignored at the command's entry explicitly.
			signal.Ignore(os.Interrupt, syscall.SIGTERM)
		}
		if err := provider.Save(provider.Provider{ID: "drain", Chat: os.Getenv("MAGPIE_TEST_WEB_UPSTREAM") + "/v1", Key: "fixture", Models: []string{"m"}}); err != nil {
			t.Fatal(err)
		}
		os.Args = []string{"magpie", "web", "--addr", "127.0.0.1:0", "--no-open"}
		if os.Getenv("MAGPIE_TEST_WEB_COMMAND") == "serve" {
			os.Args = []string{"magpie", "serve", "--web=127.0.0.1:0"}
		}
		Main("test")
		if gateway.Running() {
			t.Fatal("gateway still serves after web returned")
		}
		return
	}
	for _, command := range []string{"web", "serve"} {
		for _, mode := range []string{"SIGTERM", "SIGINT", "SIGHUP", "quit", "ignored-INT", "ignored-both", "ignored-both-HUP"} {
			t.Run(command+"/"+mode, func(t *testing.T) {
				entered, release := make(chan struct{}), make(chan struct{})
				var released sync.Once
				up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.URL.Path != "/v1/chat/completions" {
						http.NotFound(w, r)
						return
					}
					close(entered)
					select {
					case <-release:
					case <-r.Context().Done():
						return
					}
					w.Header().Set("Content-Type", "application/json")
					io.WriteString(w, `{"choices":[{"message":{"role":"assistant","content":"drained"},"finish_reason":"stop"}]}`)
				}))
				defer up.Close()
				defer released.Do(func() { close(release) })
				ln, err := net.Listen("tcp", "127.0.0.1:0")
				if err != nil {
					t.Fatal(err)
				}
				gatewayAddr := ln.Addr().String()
				ln.Close()
				home := t.TempDir()
				config := filepath.Join(home, "config", "magpie")
				if err := os.MkdirAll(config, 0o700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(config, "settings.json"), []byte(`{"noAutoUpdate":true}`), 0o600); err != nil {
					t.Fatal(err)
				}
				self, err := os.Executable()
				if err != nil {
					t.Fatal(err)
				}
				ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
				defer cancel()
				cmd := proc.CommandContext(ctx, self, "-test.run=^TestWebShutdown$")
				if mode == "ignored-INT" {
					cmd = proc.CommandContext(ctx, "/bin/sh", "-c", `trap "" INT; exec "$0" "$@"`, self, "-test.run=^TestWebShutdown$")
				}
				cmd.Env = []string{
					"MAGPIE_TEST_WEB_SHUTDOWN=" + mode, "MAGPIE_TEST_WEB_COMMAND=" + command, "MAGPIE_TEST_WEB_UPSTREAM=" + up.URL, "HOME=" + home, "USERPROFILE=" + home,
					"XDG_CONFIG_HOME=" + filepath.Dir(config), "XDG_CACHE_HOME=" + filepath.Join(home, "cache"),
					"XDG_DATA_HOME=" + filepath.Join(home, "data"), "XDG_STATE_HOME=" + filepath.Join(home, "state"),
					"PATH=" + home, "NO_COLOR=1", "MAGPIE_ADDR=" + gatewayAddr,
					"MAGPIE_WEB_KEY=web-shutdown-test-key", "HTTP_PROXY=http://127.0.0.1:1", "HTTPS_PROXY=http://127.0.0.1:1",
					testenv.Marker + "=" + os.Getenv(testenv.Marker), "TMPDIR=" + os.Getenv("TMPDIR"),
				}
				log, err := os.Create(filepath.Join(home, "web.log"))
				if err != nil {
					t.Fatal(err)
				}
				defer log.Close()
				cmd.Stdout, cmd.Stderr = log, log
				if err := cmd.Start(); err != nil {
					t.Fatal(err)
				}
				done := make(chan error, 1)
				go func() {
					done <- cmd.Wait()
					close(done)
				}()
				defer func() {
					cancel()
					// Always reap the child, including a failed readiness check.
					<-done
				}()
				link := regexp.MustCompile(`http://127\.0\.0\.1:[0-9]+/\?k=[a-zA-Z0-9-]+`)
				var linkURL string
				for linkURL == "" {
					b, _ := os.ReadFile(log.Name())
					linkURL = link.FindString(string(b))
					select {
					case err := <-done:
						t.Fatalf("web exited before ready: %v\n%s", err, b)
					case <-time.After(10 * time.Millisecond):
					}
				}
				jar, _ := cookiejar.New(nil)
				client := &http.Client{Jar: jar, Timeout: 3 * time.Second}
				resp, err := client.Get(linkURL)
				if err != nil {
					t.Fatal(err)
				}
				resp.Body.Close()
				if resp.StatusCode != http.StatusOK {
					t.Fatalf("web readiness: HTTP %d", resp.StatusCode)
				}
				if mode == "ignored-INT" || mode == "ignored-both" || mode == "ignored-both-HUP" {
					ignored := []os.Signal{os.Interrupt}
					if mode != "ignored-INT" {
						ignored = append(ignored, syscall.SIGTERM, syscall.SIGUSR1)
					}
					for _, sig := range ignored {
						if err := cmd.Process.Signal(sig); err != nil {
							t.Fatal(err)
						}
					}
					// Allow delivery before proving the page is still serving. A signal
					// context with no signals would accidentally catch SIGUSR1 too.
					select {
					case err := <-done:
						t.Fatalf("ignored signals ended web: %v", err)
					case <-time.After(300 * time.Millisecond):
					}
					resp, err := client.Get(linkURL)
					if err != nil {
						t.Fatal("ignored signals stopped the page:", err)
					}
					resp.Body.Close()
					if resp.StatusCode != http.StatusOK {
						t.Fatalf("page after ignored signals: HTTP %d", resp.StatusCode)
					}
				}
				// Hold a real gateway request until the page listener has closed.
				// Removing stopServing must cut this answer, not merely still exit 0.
				var answer chan error
				if mode != "SIGHUP" && mode != "ignored-both-HUP" {
					answer = make(chan error, 1)
					defer func() { cancel(); <-answer }()
					go func() {
						defer close(answer)
						req, _ := http.NewRequestWithContext(ctx, http.MethodPost, "http://"+gatewayAddr+"/v1/chat/completions", strings.NewReader(`{"model":"drain/m","messages":[{"role":"user","content":"hello"}]}`))
						req.Header.Set("Authorization", "Bearer "+gateway.Token)
						req.Header.Set("Content-Type", "application/json")
						r, err := client.Do(req)
						if err == nil {
							b, readErr := io.ReadAll(r.Body)
							r.Body.Close()
							if readErr != nil || r.StatusCode != http.StatusOK || !strings.Contains(string(b), "drained") {
								err = fmt.Errorf("drained reply: HTTP %d, %s (%v)", r.StatusCode, b, readErr)
							}
						}
						answer <- err
					}()
					select {
					case <-entered:
					case err := <-answer:
						t.Fatalf("request never reached the fixture: %v", err)
					case <-ctx.Done():
						t.Fatal("gateway request did not start")
					}
				}
				if mode == "quit" || mode == "ignored-both" {
					resp, err := client.Post(resp.Request.URL.String()+"api/window/quit", "application/json", nil)
					if err != nil {
						t.Fatal(err)
					}
					resp.Body.Close()
					if resp.StatusCode != http.StatusNoContent {
						t.Fatalf("quit: HTTP %d", resp.StatusCode)
					}
				} else {
					sig := map[string]os.Signal{"SIGTERM": syscall.SIGTERM, "SIGINT": os.Interrupt, "SIGHUP": syscall.SIGHUP, "ignored-INT": syscall.SIGTERM, "ignored-both-HUP": syscall.SIGHUP}[mode]
					if err := cmd.Process.Signal(sig); err != nil {
						t.Fatal(err)
					}
				}
				if answer != nil {
					u, _ := url.Parse(linkURL)
					for {
						conn, err := net.DialTimeout("tcp", u.Host, 100*time.Millisecond)
						if err != nil {
							break
						}
						conn.Close()
						select {
						case <-ctx.Done():
							t.Fatal("page did not stop")
						case <-time.After(10 * time.Millisecond):
						}
					}
					released.Do(func() { close(release) })
					if err := <-answer; err != nil {
						t.Error("in-flight gateway request was cut:", err)
					}
				}
				err = <-done
				b, _ := os.ReadFile(log.Name())
				if ctx.Err() != nil {
					t.Fatalf("web did not stop: %v\n%s", ctx.Err(), b)
				}
				if mode == "SIGHUP" || mode == "ignored-both-HUP" {
					if err == nil {
						t.Fatal("hangup should still terminate the process, not become a clean shutdown")
					}
				} else if err != nil {
					t.Fatalf("%s should exit successfully: %v\n%s", mode, err, b)
				}
			})
		}
	}
}
