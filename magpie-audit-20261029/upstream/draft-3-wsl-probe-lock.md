# Windows：WSL 发行版探测持锁导致其他读取阻塞

## 现象与根因
`internal/agent/wsl.go:1093` 的 `wslDistros` 持有共享互斥锁，逐个调用 `wslProbe`；每次 `wslRun` 最长 30 秒。其他 WSL 状态读取在探测期间等待该锁。

## 修复
在锁外执行探测，并协调并发读取、刷新及已观察到的停止与重启；旧探测结果失效后不覆盖新状态。下面是基于 upstream/main `926ef769f` 的完整补丁。

```diff
internal/agent/rescan.go                      |   4 +
 internal/agent/wsl.go                         | 133 +++++++++++--
 internal/agent/wsl_probe_lock_windows_test.go | 271 ++++++++++++++++++++++++++
 3 files changed, 387 insertions(+), 21 deletions(-)

Changes:

internal/agent/rescan.go
@@ -25,6 +25,10 @@ func Rescan() {
 	// a running distro is asked again, and one that failed is retried now
 	// rather than in ten minutes
 	wsl.probed, wsl.failed = map[string]bool{}, map[string]time.Time{}
+	for n, done := range wsl.probing {
+		delete(wsl.probing, n)
+		close(done)
+	}
 	wsl.Unlock()
 
 	almaRead.Lock()
  +4 -0

internal/agent/wsl.go
@@ -977,6 +977,7 @@ var wsl struct {
 	seen    map[string]*distro
 	probed  map[string]bool
 	failed  map[string]time.Time
+	probing map[string]chan struct{}
 	dirty   bool
 }
 
@@ -1091,8 +1092,8 @@ func wslHung(hung bool) {
 func wslStatePath() string { return filepath.Join(filepath.Dir(provider.Path()), "wsl.json") }
 
 func wslDistros() []distro {
+retry:
 	wsl.Lock()
-	defer wsl.Unlock()
 	if wsl.seen == nil {
 		wsl.seen, wsl.probed, wsl.failed = map[string]*distro{}, map[string]bool{}, map[string]time.Time{}
 		if b, err := os.ReadFile(wslStatePath()); err == nil {
@@ -1115,38 +1116,116 @@ func wslDistros() []distro {
 			// none to run: a wsl.exe that couldn't list them isn't asked
 			// again which run, another timeout under the lock the GUI waits on
 			wsl.running, wsl.runOK, wsl.runAt = map[string]bool{}, wsl.listed, time.Now()
+			for n, done := range wsl.probing {
+				delete(wsl.probing, n)
+				close(done)
+			}
+			clear(wsl.probed)
 		}
 	}
 	wslRunningLocked()
-	installed := map[string]bool{}
-	cfg := wslConfig()
-	var out []distro
+	if wsl.listed {
+		for n, done := range wsl.probing {
+			if !slices.Contains(wsl.names, n) {
+				delete(wsl.probing, n)
+				close(done)
+			}
+		}
+		for n := range wsl.probed {
+			if !slices.Contains(wsl.names, n) {
+				delete(wsl.probed, n)
+			}
+		}
+	}
+	if wsl.probing == nil {
+		wsl.probing = map[string]chan struct{}{}
+	}
+	// A channel identifies a probe and wakes readers after it is merged or invalidated.
+	probing := map[string]chan struct{}{}
+	var names []string
 	for _, n := range wsl.names {
-		installed[n] = true
 		// a distro that stopped is probed again once it runs: WSL may have
 		// been restarted meanwhile in another networking mode
 		if !wsl.running[n] {
 			delete(wsl.probed, n)
+			continue
+		}
+		if wsl.probed[n] || wsl.probing[n] != nil {
+			continue
+		}
+		if t, ok := wsl.failed[n]; ok && time.Since(t) <= wslRetryAge {
+			continue
+		}
+		probe := make(chan struct{})
+		probing[n], wsl.probing[n] = probe, probe
+		names = append(names, n)
+	}
+	wsl.Unlock()
+
+	cfg := wslConfig()
+	results := make(map[string]*distro, len(probing))
+	for _, n := range names {
+		wsl.Lock()
+		current := wsl.probing[n] == probing[n] && wsl.running[n] && slices.Contains(wsl.names, n)
+		wsl.Unlock()
+		if current {
+			results[n] = wslProbe(n)
 		}
-		if wsl.running[n] && !wsl.probed[n] {
-			if t, ok := wsl.failed[n]; !ok || time.Since(t) > wslRetryAge {
-				if d := wslProbe(n); d == nil {
-					wsl.failed[n] = time.Now()
-				} else {
-					wsl.probed[n] = true
-					if wslKept(*d) {
-						if old := wsl.seen[n]; old != nil {
-							d.Values = old.Values
-							d.Was = wslWas(old.Was, old.Gateway, d.Gateway)
-						}
-						wsl.seen[n] = d
-					} else {
-						delete(wsl.seen, n)
+	}
+
+	wsl.Lock()
+	for _, n := range names {
+		if wsl.probing[n] != probing[n] {
+			continue
+		}
+		delete(wsl.probing, n)
+		close(probing[n])
+		if !wsl.running[n] || wsl.probed[n] || !slices.Contains(wsl.names, n) {
+			continue
+		}
+		d := results[n]
+		if d == nil {
+			wsl.failed[n] = time.Now()
+			continue
+		}
+		wsl.probed[n] = true
+		if wslKept(*d) {
+			if old := wsl.seen[n]; old != nil {
+				d.Values = old.Values
+				d.Was = wslWas(old.Was, old.Gateway, d.Gateway)
+			}
+			wsl.seen[n] = d
+		} else {
+			delete(wsl.seen, n)
+		}
  ... (1 deletion, 24 additions truncated)
@@ -1159,12 +1238,13 @@ func wslDistros() []distro {
 	}
 	// an unregistered distro is forgotten
 	for n := range wsl.seen {
-		if wsl.listed && !installed[n] {
+		if wsl.listed && !slices.Contains(wsl.names, n) {
 			delete(wsl.seen, n)
 			wsl.dirty = true
 		}
 	}
 	wslSaveLocked()
+	wsl.Unlock()
 	return out
 }
 
@@ -1225,6 +1305,17 @@ func wslRunningLocked() {
 	for _, n := range run {
 		wsl.running[n] = true
 	}
+	for n, done := range wsl.probing {
+		if !wsl.running[n] {
+			delete(wsl.probing, n)
+			close(done)
+		}
+	}
+	for n := range wsl.probed {
+		if !wsl.running[n] {
+			delete(wsl.probed, n)
+		}
+	}
 	wsl.runAt = time.Now()
 }
 
  +112 -21

internal/agent/wsl_probe_lock_windows_test.go
@@ -0,0 +1,271 @@
+package agent
+
+import (
+	"context"
+	"encoding/json"
+	"errors"
+	"os"
+	"os/exec"
+	"strings"
+	"sync"
+	"testing"
+	"testing/synctest"
+	"time"
+)
+
+func TestWSLProbeConcurrentReaderWaits(t *testing.T) {
+	if os.Getenv("MAGPIE_WSL_PROBE_TEST_CHILD") != "1" {
+		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
+		defer cancel()
+		cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestWSLProbeConcurrentReaderWaits$", "-test.count=1")
+		cmd.Env = append(os.Environ(), "MAGPIE_WSL_PROBE_TEST_CHILD=1")
+		out, err := cmd.CombinedOutput()
+		if ctx.Err() != nil {
+			t.Fatalf("probe test timed out: %s", out)
+		}
+		if err != nil {
+			t.Fatalf("probe test failed: %v\n%s", err, out)
+		}
+		return
+	}
+	syncHome(t)
+	fakeWSL(t, "", "", nil, nil)
+	wsl.Lock()
+	wsl.probing = nil
+	wsl.Unlock()
+	t.Cleanup(func() { wsl.Lock(); wsl.probing = nil; wsl.Unlock() })
+
+	synctest.Test(t, func(t *testing.T) {
+		started, release := make(chan struct{}), make(chan struct{})
+		var releaseOnce sync.Once
+		finishProbe := func() { releaseOnce.Do(func() { close(release) }) }
+		defer finishProbe()
+		var mu sync.Mutex
+		calls := 0
+		wslRun = func(_ time.Duration, args ...string) ([]byte, error) {
+			switch strings.Join(args, " ") {
+			case "-l -q", "-l --running -q":
+				return []byte("Ubuntu\n"), nil
+			}
+			if len(args) > 2 && args[0] == "-d" {
+				mu.Lock()
+				calls++
+				first := calls == 1
+				mu.Unlock()
+				if first {
+					close(started)
+					<-release
+				}
+				return []byte("home:/home/me\ndir:.codex\n"), nil
+			}
+			return nil, errors.New("unexpected command")
+		}
+		wslRoot = func(name string) string { return name }
+		first, second := make(chan []distro, 1), make(chan []distro, 1)
+		go func() { first <- wslDistros() }()
+		select {
+		case <-started:
+		case <-time.After(10 * time.Second):
+			t.Fatal("first probe did not start")
+		}
+		go func() { second <- wslDistros() }()
+		settled := make(chan struct{})
+		go func() { synctest.Wait(); close(settled) }()
+		select {
+		case <-settled:
+		case <-time.After(10 * time.Second):
+			t.Fatal("readers did not reach a blocked state")
+		}
+		// Wait returns only when both readers are blocked on bubble channels.
+
+		select {
+		case got := <-second:
+			t.Fatalf("second reader returned before probe completed: %+v", got)
+		default:
+		}
+		finishProbe()
+		for _, result := range []chan []distro{first, second} {
+			select {
+			case got := <-result:
+				if len(got) != 1 || got[0].Name != "Ubuntu" || got[0].Home != "/home/me" || !got[0].Running {
+					t.Errorf("unexpected distros: %+v", got)
+				}
+			case <-time.After(10 * time.Second):
+				t.Error("reader did not finish")
+			}
+		}
+		mu.Lock()
+		count := calls
+		mu.Unlock()
+		if count != 1 {
  ... (171 additions truncated)
  +271 -0
[full diff: rtk git diff --no-compact]
```

## 验证及限制
Linux 宿主 `go build ./...`、`go vet ./...` 与 `GOOS=windows` 的相同命令通过；独立代码复审通过。Windows 专用回归测试及真实 WSL 未运行，交错行为仍需在 Windows 实机验证。

分支在 skygunner/magpie，欢迎直接 cherry-pick 吸收。分支和提交链接在推送后回贴。

## Delivered

- issue: https://github.com/yetone/magpie/issues/4136
- branch: https://github.com/skygunner/magpie/tree/fix/4136-wsl-probe-lock
- head commit: https://github.com/skygunner/magpie/commit/788f18e38b92cdd1a3a1c1cff8cf846552850319
- comment: https://github.com/yetone/magpie/issues/4136#issuecomment-6485286748
