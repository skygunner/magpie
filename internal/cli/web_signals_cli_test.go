//go:build linux

package cli

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/yetone/magpie/internal/access"
	"github.com/yetone/magpie/internal/gateway"
	"github.com/yetone/magpie/internal/proc"
	"github.com/yetone/magpie/internal/provider"
	"github.com/yetone/magpie/internal/settings"
	"github.com/yetone/magpie/internal/testenv"
)

// Run the web command with main's signal handler, not just Web.Wait:
// the global probe cleanup used to kill it before it could shut down.
func TestWebShutdown(t *testing.T) {
	if mode := os.Getenv("MAGPIE_TEST_WEB_SHUTDOWN"); mode != "" {
		f := os.NewFile(3, "gateway-listener")
		ln, err := net.FileListener(f)
		f.Close()
		if err != nil {
			t.Fatal(err)
		}
		gateway.Inherit(ln, nil)
		if os.Getenv("MAGPIE_TEST_WEB_COMMAND") == "page" {
			s := settings.Load()
			s.LAN = true
			if err := settings.Save(s); err != nil {
				t.Fatal(err)
			}
			key, err := access.Update("add-key", access.Change{Name: "test"})
			if err != nil {
				t.Fatal(err)
			}
			fmt.Println("TEST_GATEWAY_KEY=" + key)
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
	for _, command := range []string{"web", "serve", "page"} {
		for _, mode := range []string{"SIGTERM", "SIGINT", "SIGHUP", "quit"} {
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
				file, err := ln.(*net.TCPListener).File()
				if err != nil {
					t.Fatal(err)
				}
				defer file.Close()
				defer ln.Close()
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
				// The child inherits this bound listener; no unowned port exists.
				cmd.ExtraFiles = []*os.File{file}
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
				stdout, err := cmd.StdoutPipe()
				if err != nil {
					t.Fatal(err)
				}
				cmd.Stderr = log
				if err := cmd.Start(); err != nil {
					t.Fatal(err)
				}
				file.Close()
				ln.Close()
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
				ready := make(chan string, 1)
				keyReady := make(chan string, 1)
				go func() {
					scanner := bufio.NewScanner(stdout)
					for scanner.Scan() {
						if k, ok := strings.CutPrefix(scanner.Text(), "TEST_GATEWAY_KEY="); ok {
							keyReady <- k
						}
						if u := link.FindString(scanner.Text()); u != "" {
							ready <- u
							return
						}
					}
					close(ready)
				}()
				var linkURL string
				select {
				case linkURL = <-ready:
					if linkURL == "" {
						t.Fatal("web exited before ready")
					}
				case <-ctx.Done():
					t.Fatal("web did not become ready")
				}
				requestAddr, requestKey := gatewayAddr, gateway.Token
				if command == "page" {
					requestAddr = strings.TrimPrefix(strings.Split(linkURL, "/?")[0], "http://")
					select {
					case requestKey = <-keyReady:
					case <-ctx.Done():
						t.Fatal("gateway key not ready")
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
				// A live idle connection observes the server closing before the
				// held gateway response is released, without timing or polling.
				pageConn, err := net.Dial("tcp", resp.Request.URL.Host)
				if err != nil {
					t.Fatal(err)
				}
				defer pageConn.Close()
				if _, err := fmt.Fprintf(pageConn, "GET / HTTP/1.1\r\nHost: %s\r\n\r\n", resp.Request.URL.Host); err != nil {
					t.Fatal(err)
				}
				pageReader := bufio.NewReader(pageConn)
				pageResp, err := http.ReadResponse(pageReader, nil)
				if err != nil {
					t.Fatal(err)
				}
				if _, err := io.Copy(io.Discard, pageResp.Body); err != nil {
					t.Fatal(err)
				}
				pageResp.Body.Close()
				// Hold a real gateway request until the page listener has closed.
				// Removing stopServing must cut this answer, not merely still exit 0.
				var answer chan error
				if mode != "SIGHUP" {
					answer = make(chan error, 1)
					defer func() { cancel(); <-answer }()
					go func() {
						defer close(answer)
						req, _ := http.NewRequestWithContext(ctx, http.MethodPost, "http://"+requestAddr+"/v1/chat/completions", strings.NewReader(`{"model":"drain/m","messages":[{"role":"user","content":"hello"}]}`))
						req.Header.Set("Authorization", "Bearer "+requestKey)
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
				if mode == "quit" {
					resp, err := client.Post(resp.Request.URL.String()+"api/window/quit", "application/json", nil)
					if err != nil {
						t.Fatal(err)
					}
					resp.Body.Close()
					if resp.StatusCode != http.StatusNoContent {
						t.Fatalf("quit: HTTP %d", resp.StatusCode)
					}
				} else {
					sig := map[string]os.Signal{"SIGTERM": syscall.SIGTERM, "SIGINT": os.Interrupt, "SIGHUP": syscall.SIGHUP}[mode]
					if err := cmd.Process.Signal(sig); err != nil {
						t.Fatal(err)
					}
				}
				if answer != nil {
					if err := pageConn.SetReadDeadline(time.Now().Add(15 * time.Second)); err != nil {
						t.Fatal(err)
					}
					if _, err := pageReader.ReadByte(); err != io.EOF {
						t.Fatalf("page connection did not close: %v", err)
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
				if mode == "SIGHUP" {
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
