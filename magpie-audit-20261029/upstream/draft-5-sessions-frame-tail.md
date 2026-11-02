# sessions：损坏的 zstd 帧让会话和用量停在旧尾部

## 现象与根因
`internal/sessions/frames.go:212-237` 把损坏头或解码失败当作尚未写完，调用方会保存尾部位置；后续数据不再计入会话和用量。直接跳过损坏帧也不安全：可能丢失 fork 的 `session/end-seed` 边界或跨帧行，导致重复计数。

## 修复及边界
完整帧解码错误及明确损坏头返回错误，摘要保留上次成功结果，调用缓存不发布部分结果；未完成帧仍等待。提高持久缓存版本，丢弃可能错误的旧缓存。该方案避免静默错误计数，**不自动恢复损坏帧之后的数据**；需要修复源文件并改变大小或修改时间以触发重读，保持元数据不变的修复可能无法检测。

## 可应用 diff
```diff
docs/subsystems/gateway-sessions.md      |  15 ++-
 internal/sessions/call_cache.go          |   3 +-
 internal/sessions/calls_dsh.go           |   2 +-
 internal/sessions/dsh.go                 |   6 +
 internal/sessions/frames.go              |   8 +-
 internal/sessions/frames_corrupt_test.go | 189 +++++++++++++++++++++++++++++++
 internal/sessions/frames_review_test.go  |  98 ++++++++++++++++
 internal/sessions/sessions.go            |   3 +-
 8 files changed, 316 insertions(+), 8 deletions(-)

Changes:

docs/subsystems/gateway-sessions.md
@@ -423,7 +423,20 @@ usage, and a fork's copy of its parent's entries (written before its
 header) is the parent's. The other agents the Sessions page reads (Cline,
 Qoder, Grok, Droid, Cursor, Hermes, Alma) have no reader yet (#1607). The
 Context page reads the gateway's own traces, so it shows none of these
-calls. Verification: `TestMiniMaxCodeCalls`, `TestKimiCode*` (the
+calls. For packed dsh sessions, [`scanFrames`](../../internal/sessions/frames.go)
+stops at a malformed header or decoder error, including a checksum mismatch:
+the lost frame can contain a fork's end-seed marker or part of a line. An
+incomplete frame waits for more bytes, since it can contain a valid frame's
+magic as data. On a read error the summary retains its previous parse, if any,
+and the calls reader retains its previous cache; without one they expose no
+partial data. A changed size or modification time triggers a new read; the
+old tail is reused only if its sampled prefix still matches. A repair that
+preserves both may leave cached data unchanged until cache invalidation, and a
+rewrite missed by the sampled prefix can also retain an earlier parse. The
+summary and calls cache versions were raised to discard results produced by
+the former skip behavior. Later records remain unavailable until the source is
+repaired and detected.
+Verification: `TestMiniMaxCodeCalls`, `TestKimiCode*` (the
 reporter's real wire files), `TestCodeBuddyCalls`, `TestWorkBuddyCalls*`,
 `TestPiCalls*` (Pi 1.1.0's and omp 18.9.1's own files) and
 `TestWSLSessionsListed`.
  +14 -1

internal/sessions/call_cache.go
@@ -40,10 +40,11 @@ import (
 // call's own time.
 // v14: Codex calls keep the thread that made the request.
 // v15: Codex calls keep the compaction marker independently of its duration.
+// v16: discard dsh calls read past a corrupt frame.
 // What each version names is what a cache of the one before it cannot serve:
 // the version is part of the directory's name, so a cache written by an older
 // one is never read (pruneCalls).
-const callCacheVersion = "calls-v15"
+const callCacheVersion = "calls-v16"
 
 // CallCacheVersion is callCacheVersion, for a cache of what is made of the
 // calls (usage's requests-v1.bin).
  +2 -1

internal/sessions/calls_dsh.go
@@ -108,7 +108,7 @@ func readDshCalls(f file, old *callFile) (st *callFile, kept int, ok bool) {
 		next, err = scanAt(f.path, off, nil, func(b []byte, _, _ int64) bool { return each(b) })
 		end = next
 	}
-	if !ok || err != nil && !headed {
+	if !ok || err != nil {
 		return &callFile{Path: f.path, Agent: "dsh"}, 0, false
 	}
 	c.Tail = nil
  +1 -1

internal/sessions/dsh.go
@@ -303,6 +303,12 @@ func parseDsh(f file, old *state) *state {
 		// not a session's file: only its size and time
 		return &state{Size: f.size, Mod: f.mod.UnixNano()}
 	}
+	if err != nil {
+		if old != nil {
+			return old
+		}
+		return &state{}
+	}
 	c.Tail = nil
 	if err == nil {
 		c.Tail = markTail(f.path, next, nextSkip, end)
  +6 -0

internal/sessions/frames.go
@@ -211,9 +211,10 @@ func scanFrames(path string, off, skip int64, fn func([]byte) bool) (next, nextS
 		var n int64
 		frame, n, err = readFrame(r, frame, frameSize(f, at))
 		if err != nil {
-			// the end, a frame still being written, or not one at all:
-			// what was read stands, the rest waits for the next read
-			err = nil
+			// An incomplete frame waits; a malformed header has no safe boundary.
+			if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
+				err = nil
+			}
 			return
 		}
 		start := at
@@ -226,7 +227,6 @@ func scanFrames(path string, off, skip int64, fn func([]byte) bool) (next, nextS
 			continue
 		}
 		if out, err = dec.DecodeAll(frame, out[:0]); err != nil {
-			err = nil
 			end = start
 			return
 		}
  +4 -4

internal/sessions/frames_corrupt_test.go
@@ -0,0 +1,189 @@
+package sessions
+
+import (
+	"bytes"
+	"errors"
+	"os"
+	"path/filepath"
+	"reflect"
+	"testing"
+
+	"github.com/klauspost/compress/zstd"
+)
+
+func TestScanFramesCorruption(t *testing.T) {
+	enc, err := zstd.NewWriter(nil, zstd.WithEncoderCRC(true))
+	if err != nil {
+		t.Fatal(err)
+	}
+	defer enc.Close()
+	frame := func(s string) []byte { return enc.EncodeAll([]byte(s), nil) }
+	first, last := frame("first\n"), frame("last\n")
+	bad := frame("broken\n")
+	bad[len(bad)-1] ^= 0xff
+	malformed := []byte{0, 0, 0, 0}
+
+	for _, tc := range []struct {
+		name    string
+		middle  []byte
+		want    []string
+		wantErr error
+	}{
+		{"complete corrupt frame", bad, []string{"first"}, zstd.ErrCRCMismatch},
+		{"malformed header", malformed, []string{"first"}, errFrame},
+	} {
+		t.Run(tc.name, func(t *testing.T) {
+			p := filepath.Join(t.TempDir(), "session.zstd")
+			data := append(append(bytes.Clone(first), tc.middle...), last...)
+			if err := os.WriteFile(p, data, 0o600); err != nil {
+				t.Fatal(err)
+			}
+			var lines []string
+			next, skip, end, err := scanFrames(p, 0, 0, func(b []byte) bool { lines = append(lines, string(b)); return true })
+			if !reflect.DeepEqual(lines, tc.want) {
+				t.Fatalf("lines = %q, want %q", lines, tc.want)
+			}
+			if !errors.Is(err, tc.wantErr) || next != int64(len(first)) || end != next || skip != 0 {
+				t.Fatalf("corrupt frame: next=%d skip=%d end=%d err=%v", next, skip, end, err)
+			}
+		})
+	}
+}
+
+func TestScanFramesIncompleteRetained(t *testing.T) {
+	enc, err := zstd.NewWriter(nil)
+	if err != nil {
+		t.Fatal(err)
+	}
+	defer enc.Close()
+	first, last := enc.EncodeAll([]byte("first\n"), nil), enc.EncodeAll([]byte("last\n"), nil)
+	p := filepath.Join(t.TempDir(), "session.zstd")
+	cut := len(last) / 2
+	if err := os.WriteFile(p, append(bytes.Clone(first), last[:cut]...), 0o600); err != nil {
+		t.Fatal(err)
+	}
+	var lines []string
+	next, skip, end, err := scanFrames(p, 0, 0, func(b []byte) bool { lines = append(lines, string(b)); return true })
+	if err != nil || next != int64(len(first)) || end != next || skip != 0 || !reflect.DeepEqual(lines, []string{"first"}) {
+		t.Fatalf("partial: lines=%q next=%d skip=%d end=%d err=%v", lines, next, skip, end, err)
+	}
+	f, err := os.OpenFile(p, os.O_APPEND|os.O_WRONLY, 0)
+	if err != nil {
+		t.Fatal(err)
+	}
+	if _, err := f.Write(last[cut:]); err != nil {
+		t.Fatal(err)
+	}
+	if err := f.Close(); err != nil {
+		t.Fatal(err)
+	}
+	_, _, _, err = scanFrames(p, next, skip, func(b []byte) bool { lines = append(lines, string(b)); return true })
+	if err != nil || !reflect.DeepEqual(lines, []string{"first", "last"}) {
+		t.Fatalf("completed: lines=%q err=%v", lines, err)
+	}
+}
+
+func TestDshReadersStopAtCorruptFrame(t *testing.T) {
+	enc, err := zstd.NewWriter(nil, zstd.WithEncoderCRC(true))
+	if err != nil {
+		t.Fatal(err)
+	}
+	defer enc.Close()
+	head := enc.EncodeAll([]byte("{\"type\":\"session\",\"id\":\"s\",\"version\":3}\n"), nil)
+	bad := enc.EncodeAll([]byte("broken\n"), nil)
+	bad[len(bad)-1] ^= 0xff
+	reply := enc.EncodeAll([]byte("{\"type\":\"assistant/message\",\"seq\":1,\"time\":1790503290000,\"data\":{\"message\":{\"source\":{\"model\":\"m\"}},\"usage\":{\"inputTokens\":7,\"outputTokens\":1}}}\n"), nil)
+	p := filepath.Join(t.TempDir(), "session.zstd")
+	data := append(append(bytes.Clone(head), bad...), reply...)
+	if err := os.WriteFile(p, data, 0o600); err != nil {
+		t.Fatal(err)
+	}
  ... (89 additions truncated)
  +189 -0

internal/sessions/frames_review_test.go
@@ -0,0 +1,98 @@
+package sessions
+
+import (
+	"bytes"
+	"errors"
+	"os"
+	"path/filepath"
+	"testing"
+	"time"
+
+	"github.com/klauspost/compress/zstd"
+)
+
+func TestCorruptFrameDoesNotExposeLineContinuation(t *testing.T) {
+	enc, err := zstd.NewWriter(nil, zstd.WithEncoderCRC(true))
+	if err != nil {
+		t.Fatal(err)
+	}
+	defer enc.Close()
+	bad := enc.EncodeAll([]byte("lost prefix"), nil)
+	bad[len(bad)-1] ^= 1
+	data := append(bad, enc.EncodeAll([]byte("suffix\n"), nil)...)
+	p := filepath.Join(t.TempDir(), "x.zstd")
+	if err := os.WriteFile(p, data, 0600); err != nil {
+		t.Fatal(err)
+	}
+	var lines []string
+	next, _, end, err := scanFrames(p, 0, 0, func(b []byte) bool { lines = append(lines, string(b)); return true })
+	if !errors.Is(err, zstd.ErrCRCMismatch) || len(lines) != 0 || next != 0 || end != 0 {
+		t.Fatalf("lines=%q next=%d end=%d err=%v", lines, next, end, err)
+	}
+}
+
+func TestDshCorruptSeedCacheRepair(t *testing.T) {
+	setup(t)
+	enc, err := zstd.NewWriter(nil, zstd.WithEncoderCRC(true))
+	if err != nil {
+		t.Fatal(err)
+	}
+	defer enc.Close()
+	frame := func(s string) []byte { return enc.EncodeAll([]byte(s), nil) }
+	head := frame("{\"type\":\"session\",\"id\":\"fork\",\"version\":3,\"isSeeded\":true}\n")
+	inherited := frame("{\"type\":\"assistant/message\",\"seq\":1,\"time\":1790503290000,\"data\":{\"message\":{\"source\":{\"model\":\"m\"}},\"usage\":{\"inputTokens\":99,\"outputTokens\":1}}}\n")
+	seed := frame("{\"type\":\"session/end-seed\",\"data\":{\"inherited\":true}}\n")
+	bad := bytes.Clone(seed)
+	bad[len(bad)-1] ^= 1
+	own := frame("{\"type\":\"assistant/message\",\"seq\":2,\"time\":1790503291000,\"data\":{\"message\":{\"source\":{\"model\":\"m\"}},\"usage\":{\"inputTokens\":7,\"outputTokens\":1}}}\n")
+	p := filepath.Join(t.TempDir(), "session.zstd")
+	build := func(mid []byte) []byte {
+		b := append(bytes.Clone(head), inherited...)
+		b = append(b, mid...)
+		return append(b, own...)
+	}
+	if err := os.WriteFile(p, build(bad), 0600); err != nil {
+		t.Fatal(err)
+	}
+	info, err := os.Stat(p)
+	if err != nil {
+		t.Fatal(err)
+	}
+	f := file{agent: "dsh", path: p, size: info.Size(), mod: info.ModTime(), main: true}
+	mu.Lock()
+	cache = make(map[string]*state)
+	refresh([]file{f}, nil)
+	summary := cache[p]
+	mu.Unlock()
+	if len(summary.Models) != 0 {
+		t.Fatalf("corrupt fork counted inherited or later usage: %+v", summary.Models)
+	}
+	calls := readCalls(f)
+	if len(calls.Calls) != 0 {
+		t.Fatalf("corrupt fork counted calls: %+v", calls.Calls)
+	}
+	if err := os.WriteFile(p, build(seed), 0600); err != nil {
+		t.Fatal(err)
+	}
+	fixedTime := info.ModTime().Add(time.Second)
+	if err := os.Chtimes(p, fixedTime, fixedTime); err != nil {
+		t.Fatal(err)
+	}
+	info, err = os.Stat(p)
+	if err != nil {
+		t.Fatal(err)
+	}
+	f.size, f.mod = info.Size(), info.ModTime()
+	mu.Lock()
+	refresh([]file{f}, nil)
+	summary = cache[p]
+	mu.Unlock()
+	m, ok := summary.Models.get("m")
+	if !ok || m.Input != 7 {
+		t.Fatalf("repaired summary: %+v", summary.Models)
+	}
+	calls = readCalls(f)
+	if len(calls.Calls) != 1 || calls.Calls[0].Input != 7 {
+		t.Fatalf("repaired calls: %+v", calls.Calls)
+	}
+}
  +98 -0

internal/sessions/sessions.go
@@ -938,7 +938,8 @@ func CachePath() string { return filepath.Join(filepath.Dir(catalog.CachePath())
 // 24: Claude Code's compact summaries' uuids (Compacts).
 // 25: the skills Codex calls up by reading their SKILL.md.
 // 26: Grok Build's tool calls and skills.
-const cacheVersion = 26
+// 27: discard dsh parses that skipped a corrupt frame.
+const cacheVersion = 27
 
 type cacheFile struct {
 	Version int               `json:"version"`
  +2 -1
[full diff: rtk git diff --no-compact]
```

## 验证
原实现的损坏帧、fork 边界和跨帧行回归失败；修复后针对性测试、`go test ./internal/sessions -count=1`、相关 race 测试、`go build ./...`、`go vet ./...` 通过；独立复审通过。未运行全仓测试或跨平台测试，未针对既有成功缓存完整演练修复。

分支在 skygunner/magpie，欢迎直接 cherry-pick 吸收；分支和提交链接在推送后回贴。

## Delivered

- issue: https://github.com/yetone/magpie/issues/4144
- branch: https://github.com/skygunner/magpie/tree/fix/4144-sessions-frame-tail
- head commit: https://github.com/skygunner/magpie/commit/ba4f3ef69a0fbfd85c0250b9e8d8dcef0ba3866f
- comment: https://github.com/yetone/magpie/issues/4144#issuecomment-6485933827
