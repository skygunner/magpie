# WebDAV/S3 备份超过 64 MiB 被截断，可能被误判损坏并覆盖

## 现象与根因
在 upstream/main `926ef769f`，`internal/davsync/dav.go:291` 与 `s3.go:234` 使用 `io.ReadAll(io.LimitReader(res.Body, 64<<20))`，超过上限的响应会被静默截断。备份解码把截断数据识别为损坏，恢复流程可能用本机副本覆盖远端完整备份。写入端也曾允许写出超过读取上限的备份。

## 修复
读取上限加一字节检测超限，返回独立错误，避免进入损坏恢复；WebDAV/S3 的备份和共享 usage 写入超过上限时在发请求前拒绝。64 MiB 上限本身保持不变。

## 可应用的补丁
```diff
docs/subsystems/sync-backup.md   |   5 +
 internal/davsync/dav.go          |   5 +-
 internal/davsync/notbackup.go    |  28 ++++-
 internal/davsync/readcap_test.go | 235 +++++++++++++++++++++++++++++++++++++++
 internal/davsync/s3.go           |   5 +-
 internal/davsync/usage.go        |  10 +-
 6 files changed, 281 insertions(+), 7 deletions(-)

Changes:

docs/subsystems/sync-backup.md
@@ -85,6 +85,11 @@ on both sides keeps the newer version while retaining the replaced copy.
 A conditional write refused because another computer committed in between
 is retried through this same merge.
 
+WebDAV and S3 backup reads and writes are limited to 64 MiB. A file beyond
+that limit returns a size error before it can be treated as damaged or written;
+a file exactly at the limit is allowed. Oversized shared usage files also
+return a size error when read. GitHub applies its own 64 MiB sync-file limit.
+
 When a remote file is empty, damaged or another app's format, sync stops and
 describes it through `ServerFile`. Settings and the backend's `upload` command
 can replace it with this computer's setup, keeping the old file first.
  +5 -0

internal/davsync/dav.go
@@ -288,7 +288,7 @@ func (d *dav) get(ctx context.Context, have version) (data []byte, v version, er
 	case res.StatusCode != http.StatusOK:
 		return nil, version{}, fmt.Errorf("reading %s from the WebDAV server: HTTP %d", file, res.StatusCode)
 	}
-	data, err = io.ReadAll(io.LimitReader(res.Body, 64<<20))
+	data, err = readCapped(res.Body)
 	if err != nil {
 		return nil, version{}, err
 	}
@@ -317,6 +317,9 @@ func (d *dav) get(ctx context.Context, have version) (data []byte, v version, er
 // backup" on every sync after, so what a write kept is read back and
 // compared; a short write is tried again and told as itself in the end.
 func (d *dav) put(ctx context.Context, data []byte, etag string) (version, error) {
+	if len(data) > maxRead {
+		return version{}, errFileTooLarge
+	}
 	h := map[string]string{"Content-Type": "application/octet-stream"}
 	if etag != "" {
 		h["If-Match"] = etag
  +4 -1

internal/davsync/notbackup.go
@@ -230,8 +230,8 @@ func unpacked(kind string, data []byte, encoding string) ([]byte, error) {
 		defer d.Close()
 		return readAll(d, nil)
 	case enc == "deflate" && len(data) > 0 && data[0] != '{':
-		if out, err := readAll(zlib.NewReader(bytes.NewReader(data))); err == nil {
-			return out, nil
+		if out, err := readAll(zlib.NewReader(bytes.NewReader(data))); err == nil || errors.Is(err, errFileTooLarge) {
+			return out, err
 		}
 		return readAll(flate.NewReader(bytes.NewReader(data)), nil)
 	case enc == "br" && len(data) > 0 && data[0] != '{':
@@ -244,5 +244,27 @@ func readAll[R io.Reader](r R, err error) ([]byte, error) {
 	if err != nil {
 		return nil, err
 	}
-	return io.ReadAll(io.LimitReader(r, 64<<20))
+	return readCapped(r)
+}
+
+// maxRead is the most of a file the sync reads: a backup or a usage file.
+const maxRead = 64 << 20
+
+// errFileTooLarge is a file on the server past maxRead. It is an error of
+// its own, never a short read: a body cut at the cap is a backup cut short
+// to backup.Open (ErrCorrupt), and a sync then takes an intact file that was
+// only bigger for damage, rebuilding or offering to replace it.
+var errFileTooLarge = errors.New("the sync file is larger than the 64 MiB limit")
+
+// readCapped reads r whole, up to maxRead: past it is errFileTooLarge, so
+// the cap never cuts a body short.
+func readCapped(r io.Reader) ([]byte, error) {
+	b, err := io.ReadAll(io.LimitReader(r, maxRead+1))
+	if err != nil {
+		return nil, err
+	}
+	if len(b) > maxRead {
+		return nil, errFileTooLarge
+	}
+	return b, nil
 }
  +25 -3

internal/davsync/readcap_test.go
@@ -0,0 +1,235 @@
+package davsync
+
+import (
+	"bytes"
+	"context"
+	"errors"
+	"net/http"
+	"net/http/httptest"
+	"testing"
+
+	"github.com/yetone/magpie/internal/backup"
+	"github.com/yetone/magpie/internal/provider"
+)
+
+// A backup bigger than the 64 MiB a sync reads was read up to the cap and
+// handed on cut. backup.Open takes a cut backup for a damaged one
+// (ErrCorrupt), so a sync rebuilt over an intact file that was merely
+// bigger, or offered to replace it. The read boundary now says the file is
+// too large, an error of its own that no sync, restore or Upload classifies
+// as damage.
+
+// overCap is a body that begins like a sealed backup and runs past the cap,
+// as a real backup bigger than magpie reads does.
+func overCap(t *testing.T) []byte {
+	t.Helper()
+	body := append([]byte(`{"format":"magpie-backup","version":1,"kdf":"pbkdf2-sha256","iterations":120000,"salt":"AAAA","nonce":"AAAA","data":"`), bytes.Repeat([]byte("A"), maxRead)...)
+	body = append(body, '"', '}')
+	if len(body) <= maxRead {
+		t.Fatalf("the body is %d bytes, not over the %d cap", len(body), maxRead)
+	}
+	return body
+}
+
+func TestReadSaysTooLargeRatherThanCut(t *testing.T) {
+	body := overCap(t)
+	// Cut at the cap, as the read did: what every caller took for damage.
+	if _, err := backup.Open(body[:maxRead], "horse"); !errors.Is(err, backup.ErrCorrupt) {
+		t.Fatalf("a body cut at the cap reads as %v, not ErrCorrupt: the misclassification is gone from this test", err)
+	}
+
+	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
+		if r.Method != http.MethodGet {
+			w.WriteHeader(http.StatusNotFound)
+			return
+		}
+		w.Write(body)
+	}))
+	defer srv.Close()
+	d, err := newDAV(Config{URL: srv.URL + "/dav/", User: "me", Password: "pw"})
+	if err != nil {
+		t.Fatal(err)
+	}
+	data, _, err := d.get(context.Background(), version{})
+	if !errors.Is(err, errFileTooLarge) {
+		t.Fatalf("a WebDAV backup over the cap: %v, %d bytes, want errFileTooLarge", err, len(data))
+	}
+	if data != nil {
+		t.Fatalf("a file too large came back cut: %d bytes", len(data))
+	}
+	var nb *notBackup
+	if errors.As(described(Config{}, data, err), &nb) {
+		t.Fatalf("too large is said as a file to replace (%q)", nb.f.What)
+	}
+
+	f, srv3 := newFakeS3(t)
+	s, err := newS3(f.config(srv3))
+	if err != nil {
+		t.Fatal(err)
+	}
+	f.mu.Lock()
+	f.objects["team x+y/magpie/magpie.magpie-backup"] = body
+	f.etags["team x+y/magpie/magpie.magpie-backup"] = `"big"`
+	f.mu.Unlock()
+	if data, _, err := s.get(context.Background(), version{}); !errors.Is(err, errFileTooLarge) || data != nil {
+		t.Fatalf("an S3 backup over the cap: %v, %d bytes, want errFileTooLarge", err, len(data))
+	}
+}
+
+// The cap itself is read whole: a file of exactly maxRead is not too large.
+func TestWriteBackupSizeLimit(t *testing.T) {
+	requests := 0
+	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
+		requests++
+		w.WriteHeader(http.StatusCreated)
+	}))
+	defer srv.Close()
+	d, err := newDAV(Config{URL: srv.URL})
+	if err != nil {
+		t.Fatal(err)
+	}
+	s, err := newS3(Config{URL: "s3://bucket", Endpoint: srv.URL, User: "key", Password: "secret"})
+	if err != nil {
+		t.Fatal(err)
+	}
+	data := make([]byte, maxRead+1)
+	for name, remote := range map[string]remote{"WebDAV": d, "S3": s} {
+		t.Run(name, func(t *testing.T) {
+			requests = 0
+			if _, err := remote.put(context.Background(), data, ""); !errors.Is(err, errFileTooLarge) {
+				t.Fatalf("oversized backup: %v, want errFileTooLarge", err)
  ... (135 additions truncated)
  +235 -0

internal/davsync/s3.go
@@ -231,7 +231,7 @@ func (s *s3) get(ctx context.Context, have version) (data []byte, v version, err
 		}
 		return nil, version{}, s.explain("read", e)
 	}
-	data, err = io.ReadAll(io.LimitReader(res.Body, 64<<20))
+	data, err = readCapped(res.Body)
 	if err != nil {
 		return nil, version{}, err
 	}
@@ -251,6 +251,9 @@ var unconditional sync.Map
 // where there is none yet — and errChanged when another computer wrote in
 // between.
 func (s *s3) put(ctx context.Context, data []byte, etag string) (version, error) {
+	if len(data) > maxRead {
+		return version{}, errFileTooLarge
+	}
 	where := s.endpoint.String() + " " + s.bucket
 	if _, no := unconditional.Load(where); !no {
 		h := map[string]string{"Content-Type": "application/octet-stream"}
  +4 -1

internal/davsync/usage.go
@@ -361,10 +361,13 @@ func (d *dav) read(ctx context.Context, name string) ([]byte, error) {
 	if res.StatusCode != http.StatusOK {
 		return nil, fmt.Errorf("reading %s from the WebDAV server: HTTP %d", name, res.StatusCode)
 	}
-	return io.ReadAll(io.LimitReader(res.Body, 64<<20))
+	return readCapped(res.Body)
 }
 
 func (d *dav) write(ctx context.Context, name string, data []byte) error {
+	if len(data) > maxRead {
+		return errFileTooLarge
+	}
 	for try, writes := 0, 0; ; try++ {
 		res, err := d.send(ctx, http.MethodPut, d.usageURL(name), data, map[string]string{"Content-Type": "application/octet-stream"})
 		if err != nil {
@@ -474,10 +477,13 @@ func (s *s3) read(ctx context.Context, name string) ([]byte, error) {
 	if res.StatusCode != http.StatusOK {
 		return nil, s.explain("read", readError(res))
 	}
-	return io.ReadAll(io.LimitReader(res.Body, 64<<20))
+	return readCapped(res.Body)
 }
 
 func (s *s3) write(ctx context.Context, name string, data []byte) error {
+	if len(data) > maxRead {
+		return errFileTooLarge
+	}
 	res, err := s.sendTo(ctx, http.MethodPut, s.urlOf(s.usageKey(name)), data, map[string]string{"Content-Type": "application/octet-stream"})
 	if err != nil {
 		return err
  +8 -2
[full diff: rtk git diff --no-compact]
```

## 验证与限制
回归先红后绿：原实现超限读取返回截断数据、同步可覆盖远端；写入超限也曾成功。修复后 WebDAV/S3 超限被拒绝，恰好 64 MiB 允许；`go test ./internal/davsync -count=1`、`go build ./...`、`go vet ./...` 通过。使用本地 HTTP 测试服务器，未测试真实 WebDAV/S3。

## 分支
分支与提交链接将在 issue 创建并推送后回贴。分支在 skygunner/magpie，欢迎直接 cherry-pick 吸收。

## Delivered

- issue: https://github.com/yetone/magpie/issues/4137
- branch: https://github.com/skygunner/magpie/tree/fix/4137-davsync-read-cap
- head commit: https://github.com/skygunner/magpie/commit/3ff971ddd56463a7bed3c8a2230ba8783ae57d23
- comment: https://github.com/yetone/magpie/issues/4137#issuecomment-6485262880
