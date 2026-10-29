# #4118 — pii 过期 miss 列表（补丁分支）

## 现象与触发
并发请求在 `backend.find` 让出模型回合（yield）后重新拿到 turn 时，仍沿用让出前读到的 `miss` 列表。若另一个请求在这段时间里把剩余文本读进了 cache，`b.missing(texts, miss, got)` 返回空，但 `miss` 非空：严格模式走到 `the local masking model has no text to read for N texts`，调用方报 masking 失败；`Bounded` 模式下则可能带着空结果提前结束。两者都表现为“本可以脱敏却返回失败/空”。

## 根因
`internal/pii/find.go` 的分块循环（upstream/main `926ef769f`，循环体 211 行起）：

```go
units := b.missing(texts, miss, got)
if lenient {
    ...
}
chunks, parts := joinAt(units, b.size())
if len(chunks) == 0 {
    done <- result{nil, fmt.Errorf("the local masking model has no text to read for %d texts", len(miss))}
    return
}
```

`miss` 是 yield 之前 `known(texts, got)` 的结果，`units` 是“现在还需要问模型的”。两者为空/非空不一致时，代码把“cache 已经补上了”误当成“没有文本可读”。这正是 #4118 正文里描述的 candidate。

## 修复
分支实现的就是 #4118 正文里的 candidate，没有另外发明路径。
`units` 为空时重新 `known(texts, got)`：若已无 miss，用 cache 的答案结束；若仍有 miss（例如被淘汰），保留原有错误路径，不把分类器失败伪装成成功。

```diff
diff --git a/internal/pii/cache_yield_regression_test.go b/internal/pii/cache_yield_regression_test.go
new file mode 100644
index 000000000..fe09df321
--- /dev/null
+++ b/internal/pii/cache_yield_regression_test.go
@@ -0,0 +1,42 @@
+package pii
+
+import (
+	"context"
+	"crypto/sha256"
+	"sync/atomic"
+	"testing"
+
+	"github.com/yetone/magpie/internal/redact"
+)
+
+func TestCacheFilledWhileYielding(t *testing.T) {
+	for _, bounded := range []bool{false, true} {
+		t.Run(map[bool]string{false: "strict", true: "bounded"}[bounded], func(t *testing.T) {
+			r := newReader(t, 12)
+			texts := []string{"Alice12345", "zzzzzzzzzz"}
+			var calls atomic.Int32
+			r.heldBack = func() error {
+				if calls.Add(1) == 3 {
+					r.cache.mu.Lock()
+					r.cache.put(sha256.Sum256([]byte(texts[0])), []redact.Found{{
+						Start: 0, End: 5, Label: "private_person", Score: .9,
+					}})
+					r.cache.mu.Unlock()
+				}
+				return nil
+			}
+			ctx := context.Background()
+			if bounded {
+				ctx = Bounded(ctx)
+			}
+			found, err := r.find(ctx, texts)
+			if err != nil {
+				t.Fatalf("cache was filled before the next chunk: %v", err)
+			}
+			if len(found) != 2 || len(found[0]) != 1 ||
+				found[0][0].Start != 0 || found[0][0].End != 5 {
+				t.Fatalf("cached detection must be returned: %#v", found)
+			}
+		})
+	}
+}
diff --git a/internal/pii/find.go b/internal/pii/find.go
index 2d277df0b..001dc73a7 100644
--- a/internal/pii/find.go
+++ b/internal/pii/find.go
@@ -211,6 +211,18 @@ func (b *backend) find(ctx context.Context, texts []string) ([][]redact.Found, e
 				return
 			}
 			units := b.missing(texts, miss, got)
+			if len(units) == 0 {
+				// nothing of miss is left to ask: the yield this one gave
+				// way in let another request read and keep it, and miss was
+				// read before that. Ask known again, so its answers are this
+				// one's: asked of miss as it was, find answered a request it
+				// had all the answers for with none found, or with an error
+				// (masking failed, so nothing was sent)
+				out, miss = b.known(texts, got)
+				if len(miss) == 0 {
+					break
+				}
+			}
 			if lenient {
 				// the backlog past its cap is left: its texts go with
 				// what the rules find

```

## 验证
在 `upstream/main` `926ef769f` 上（不是只对 v0.1.1957）：

- 只回退 `find.go`、保留测试：`go test ./internal/pii -run TestCacheFilledWhileYielding -count=1` → FAIL，两种模式分别报 `no text to read for 1 texts` 与空 span。
- 恢复修复：同命令 → ok。
- `go test ./internal/pii -count=1`、`go test -race ./internal/pii -count=1`、`go build ./...`、`go vet ./...` 全绿；`gofmt -l internal/pii` 空。

## 分支
- 分支：https://github.com/skygunner/magpie/tree/fix/4118-pii-stale-miss （skygunner/magpie，基于 upstream/main 926ef769f）
- commit：https://github.com/skygunner/magpie/commit/ab3ed532271ca649b4b5a74df6d6885cc76793cc
- 完整 diff 见上；欢迎直接 cherry-pick 吸收。这个分支只实现正文 candidate，没有改动分类器失败路径的 fail-closed 语义。

## Delivered

- issue: https://github.com/yetone/magpie/issues/4118
- branch: https://github.com/skygunner/magpie/tree/fix/4118-pii-stale-miss
- head commit: https://github.com/skygunner/magpie/commit/ab3ed532271ca649b4b5a74df6d6885cc76793cc
- comment: https://github.com/yetone/magpie/issues/4118#issuecomment-6484900780
