# WebDAV 无条件写入被误认为受版本保护

## 现象与根因
`internal/davsync/dav.go:319` 在没有 ETag 时不发送 `If-Match`，但成功结果未标记 Blind；PUT 响应的可用 Last-Modified 也被丢弃。多个客户端写入时，覆盖可能被当成受版本保护的同步。HTTP 时间只有秒级精度，不能等同于 ETag 并发写入保护。

## 修复
没有 `If-Match` 的成功写入标记 Blind；仅使用通过现有安全检查的 PUT 响应时间作为读取版本。首次盲写后发现不同远端文件且无法确定共同基线时，停止同步、保留双方数据并要求手工协调，避免按时间误删本地配置。限制是正常远端更新若与该竞争不可区分，也需要手工协调。

## 可应用补丁
```diff
docs/subsystems/sync-backup.md                  | 18 ++++++-
 internal/davsync/dav.go                         |  8 +--
 internal/davsync/dav_initial_blind_test.go      | 72 +++++++++++++++++++++++++
 internal/davsync/dav_version_regression_test.go | 50 +++++++++++++++++
 internal/davsync/davsync.go                     | 12 +++--
 internal/davsync/davsync_test.go                |  2 +
 internal/davsync/gateway_keys_join_test.go      |  1 +
 internal/davsync/gateway_keys_parts_test.go     |  3 ++
 internal/davsync/lan_own_test.go                |  1 +
 internal/davsync/restore_test.go                |  3 ++
 10 files changed, 163 insertions(+), 7 deletions(-)

Changes:

docs/subsystems/sync-backup.md
@@ -83,7 +83,23 @@ Subsequent syncs compare each part with the hashes from the previous sync:
 local-only changes go up, remote-only changes come down, and a part changed
 on both sides keeps the newer version while retaining the replaced copy.
 A conditional write refused because another computer committed in between
-is retried through this same merge.
+is retried through this same merge. WebDAV writes without `If-Match` are
+marked blind, including a short-write retry without an ETag. On the next sync,
+if another file replaced the blind write, sync merges again from the state
+before it. If competing first writes differ, there is no common baseline:
+sync stops with an error and retains this computer's setup and the server's
+backup for manual reconciliation. The same happens if the other computer
+writes after the first write but before its writer reads it back; the two
+cases cannot be distinguished. The error persists on retry. Restore can bring
+the server's setup in while keeping an Undo copy of the local setup; the user
+can then reconcile that copy. This cannot
+recover another writer's changes that the blind write itself overwrote; that
+writer must sync again.
+A server must enforce `If-Match` for an ETag-based write to provide protection. HTTP dates have second
+resolution and are not used as write preconditions. A PUT's Last-Modified is
+used for a conditional read only when its response Date is at least a second
+later; otherwise the next sync downloads the file. No earlier timestamp is
+reused as the version of a new write.
 
 When a remote file is empty, damaged or another app's format, sync stops and
 describes it through `ServerFile`. Settings and the backend's `upload` command
  +17 -1

internal/davsync/dav.go
@@ -321,6 +321,7 @@ func (d *dav) put(ctx context.Context, data []byte, etag string) (version, error
 	if etag != "" {
 		h["If-Match"] = etag
 	}
+	blind := false
 	for try, writes := 0, 0; ; try++ {
 		res, err := d.send(ctx, http.MethodPut, d.url(folder, file), data, h)
 		if err != nil {
@@ -330,6 +331,7 @@ func (d *dav) put(ctx context.Context, data []byte, etag string) (version, error
 		switch {
 		case res.StatusCode >= 200 && res.StatusCode < 300:
 			writes++
+			blind = blind || h["If-Match"] == ""
 			he, cerr := d.check(ctx, d.url(folder, file), file, len(data))
 			if cerr != nil {
 				if writes < 3 {
@@ -348,9 +350,9 @@ func (d *dav) put(ctx context.Context, data []byte, etag string) (version, error
 				}
 				return version{}, cerr
 			}
-			// an ETag said here is the version written; with none, the
-			// next sync reads the file in full, as before
-			return version{ETag: res.Header.Get("ETag")}, nil
+			v := versionOf(res.Header)
+			v.Blind = blind
+			return v, nil
 		case res.StatusCode == http.StatusPreconditionFailed:
 			return version{}, errChanged
 		case res.StatusCode == http.StatusForbidden && (try > 0 || d.there(ctx, d.url(folder)+"/")):
  +5 -3

internal/davsync/dav_initial_blind_test.go
@@ -0,0 +1,72 @@
+package davsync
+
+import (
+	"context"
+	"net/http/httptest"
+	"os"
+	"testing"
+	"time"
+
+	"github.com/yetone/magpie/internal/backup"
+	"github.com/yetone/magpie/internal/provider"
+	"github.com/yetone/magpie/internal/settings"
+)
+
+func TestDAVInitialBlindWritesPreserveBothAfterOverwrite(t *testing.T) {
+	fake := &fakeDAV{files: map[string][]byte{}, etags: map[string]string{}, dirs: map[string]bool{"/dav": true}}
+	srv := httptest.NewServer(fake)
+	defer srv.Close()
+	cfg := Config{URL: srv.URL + "/dav/", User: "me", Password: "pw", Passphrase: "horse", Keys: true}
+	ctx := context.Background()
+	a, b := newComputer(t), newComputer(t)
+	a.use(t)
+	provider.Save(provider.Provider{ID: "deepseek", Name: "A provider", Chat: "https://api.deepseek.com/v1", Key: "a"})
+	if err := Configure(cfg); err != nil {
+		t.Fatal(err)
+	}
+	if err := Now(ctx); err != nil {
+		t.Fatal(err)
+	}
+
+	// B's initial GET observes the empty folder before A's PUT.
+	fake.mu.Lock()
+	delete(fake.files, backupFile)
+	delete(fake.etags, backupFile)
+	fake.mu.Unlock()
+	b.use(t)
+	if err := settings.Save(settings.Settings{Theme: "dark"}); err != nil {
+		t.Fatal(err)
+	}
+	if err := Configure(cfg); err != nil {
+		t.Fatal(err)
+	}
+	if err := Now(ctx); err != nil {
+		t.Fatal(err)
+	}
+
+	a.use(t)
+	older := time.Now().Add(-time.Hour)
+	if err := os.Chtimes(provider.Path(), older, older); err != nil {
+		t.Fatal(err)
+	}
+	err := Now(ctx)
+	if got := ids(); len(got) != 1 || got[0] != "deepseek=a" {
+		t.Fatalf("A's provider lost: %v", got)
+	}
+	if err == nil {
+		t.Fatal("competing initial writes must stop for reconciliation")
+	}
+	bundle, err := backup.Open(onServer(fake, backupFile), cfg.Passphrase)
+	if err != nil {
+		t.Fatal(err)
+	}
+	if len(bundle.Providers) != 0 || bundle.Settings.Theme != "dark" {
+		t.Fatalf("server's competing write was changed: %+v", bundle)
+	}
+	if err := Now(ctx); err == nil {
+		t.Fatal("retry must preserve the unresolved conflict")
+	}
+	if got := ids(); len(got) != 1 || got[0] != "deepseek=a" {
+		t.Fatalf("A's provider lost on retry: %v", got)
+	}
+}
  +72 -0

internal/davsync/dav_version_regression_test.go
@@ -0,0 +1,50 @@
+package davsync
+
+import (
+	"context"
+	"net/http"
+	"net/http/httptest"
+	"testing"
+)
+
+// A successful write without If-Match can have overwritten another writer.
+func TestDAVPutReportsBlindAndUsesOnlySafePutVersion(t *testing.T) {
+	for _, tc := range []struct {
+		name, match, lastModified, date, responseETag string
+		blind                                         bool
+		modified                                      string
+	}{
+		{"unconditional", "", "", "", "", true, ""},
+		{"conditional", `"old"`, "", "", `"new"`, false, ""},
+		{"old modification date", "", "Wed, 01 Jan 2020 00:00:00 GMT", "Wed, 01 Jan 2020 00:00:02 GMT", "", true, "Wed, 01 Jan 2020 00:00:00 GMT"},
+		{"same second", "", "Wed, 01 Jan 2020 00:00:00 GMT", "Wed, 01 Jan 2020 00:00:00 GMT", "", true, ""},
+	} {
+		t.Run(tc.name, func(t *testing.T) {
+			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
+				if r.Method == http.MethodHead {
+					w.Header().Set("Content-Length", "3")
+					return
+				}
+				if got := r.Header.Get("If-Match"); got != tc.match {
+					t.Errorf("If-Match = %q, want %q", got, tc.match)
+				}
+				w.Header().Set("Date", tc.date)
+				w.Header().Set("Last-Modified", tc.lastModified)
+				w.Header().Set("ETag", tc.responseETag)
+				w.WriteHeader(http.StatusCreated)
+			}))
+			defer srv.Close()
+			d, err := newDAV(Config{URL: srv.URL})
+			if err != nil {
+				t.Fatal(err)
+			}
+			v, err := d.put(context.Background(), []byte("abc"), tc.match)
+			if err != nil {
+				t.Fatal(err)
+			}
+			if v.Blind != tc.blind || v.Modified != tc.modified || v.ETag != tc.responseETag {
+				t.Fatalf("version = %+v, want blind=%v modified=%q etag=%q", v, tc.blind, tc.modified, tc.responseETag)
+			}
+		})
+	}
+}
  +50 -0

internal/davsync/davsync.go
@@ -222,8 +222,8 @@ type state struct {
 // what this computer changed counts as changed here again and goes back
 // up beside the other's, where taking the other's file as having seen
 // this one's would bring the old parts in over this computer's changes.
-// Remote and Local are nil for a write to an empty server: the next sync
-// is then a first one, as it would have been.
+// Remote and Local are nil for a write to an empty server; if another
+// initial write overwrites it, there is no common baseline to merge.
 type blind struct {
 	Sum    string            `json:"sum"`
 	Remote map[string]string `json:"remote,omitempty"`
@@ -1059,7 +1059,13 @@ func syncOnce(ctx context.Context, c Config, st *state) error {
 	// computer's write went over it (blind)
 	if b := st.Blind; b != nil {
 		if !unchanged && sum(data) != b.Sum {
-			st.Remote, st.Local = b.Remote, b.Local
+			good := cached(st.Sum)
+			if good == nil || !bytes.HasPrefix(good, data) || len(data) <= bytes.Index(good, []byte(`"data"`)) {
+				if b.Local == nil {
+					return errors.New("another computer wrote its initial backup at the same time: this computer's setup and the server's backup were kept; reconcile them before syncing again")
+				}
+				st.Remote, st.Local = b.Remote, b.Local
+			}
 		}
 		st.Blind = nil
 	}
  +9 -3

internal/davsync/davsync_test.go
@@ -407,6 +407,7 @@ func TestSyncProviderOrder(t *testing.T) {
 	}
 	Configure(cfg)
 	now(t)
+	now(t) // confirm the initial blind write before another computer changes it
 
 	b.use(t)
 	Configure(cfg)
@@ -473,6 +474,7 @@ func TestSyncLibrary(t *testing.T) {
 	if v := now(t); v.Error != "" || !v.Library {
 		t.Fatalf("a: %+v", v)
 	}
+	now(t) // confirm the initial blind write
 
 	b.use(t)
 	os.MkdirAll(filepath.Join(string(b), ".claude"), 0o755)
  +2 -0

internal/davsync/gateway_keys_join_test.go
@@ -77,6 +77,7 @@ func TestGatewayKeysJoinKeepsUnseenKeys(t *testing.T) {
 		t.Fatal(err)
 	}
 	now()
+	now() // confirm B's initial blind write
 
 	// Computer A holds several gateway keys of its own and joins after.
 	use(a)
  +1 -0

internal/davsync/gateway_keys_parts_test.go
@@ -143,6 +143,9 @@ func TestGatewayKeysConflict(t *testing.T) {
 				if err := Now(context.Background()); err != nil {
 					t.Fatal(err)
 				}
+				if err := Now(context.Background()); err != nil {
+					t.Fatal(err)
+				}
 				path := "/dav/magpie/magpie.magpie-backup"
 				fake.mu.Lock()
 				data := append([]byte(nil), fake.files[path]...)
  +3 -0

internal/davsync/lan_own_test.go
@@ -84,6 +84,7 @@ func TestLANSharingStaysOnItsComputer(t *testing.T) {
 		t.Fatal(err)
 	}
 	now()
+	now() // confirm home's initial blind write
 
 	// Work, sharing off, joins: home's theme comes in, home's sharing
 	// doesn't.
  +1 -0

internal/davsync/restore_test.go
@@ -107,6 +107,9 @@ func TestRestore(t *testing.T) {
 	if err := Now(ctx); err != nil {
 		t.Fatal(err)
 	}
+	if err := Now(ctx); err != nil { // confirm the initial blind write
+		t.Fatal(err)
+	}
 	b.use(t)
 	if err := Configure(cfg); err != nil {
 		t.Fatal(err)
  +3 -0
```

## 验证及限制
正常时间戳下首次盲写竞争测试在修复前丢失 provider，修复后首次和重试均保留本地与远端。`go test -tags nogui ./internal/davsync -count=1`、`go build ./...`、`go vet ./...` 通过；独立代码复审通过。未在真实 WebDAV 服务器验证；测试未覆盖此特定错误后的完整 Restore 流程。

分支在 skygunner/magpie，欢迎直接 cherry-pick 吸收；分支和提交链接在推送后回贴。

## Delivered

- issue: https://github.com/yetone/magpie/issues/4141
- branch: https://github.com/skygunner/magpie/tree/fix/4141-dav-put-version
- head commit: https://github.com/skygunner/magpie/commit/3452982b8ad99aa3c74f86c326a65f5f53ad1844
- comment: https://github.com/yetone/magpie/issues/4141#issuecomment-6485383967
