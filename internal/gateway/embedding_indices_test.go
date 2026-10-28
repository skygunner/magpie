package gateway

import (
	"strings"
	"testing"
)

// An embeddings vendor that serializes index omitempty (Google's
// OpenAI-compat shape) drops index 0, and clients that sort data by index
// crash on the absent field. magpie fills only absent and null indices
// with the item's position, on both URL aliases; an index that is there
// and every other field are kept as the vendor said them (#3302).
func TestEmbeddingsSupplyMissingIndices(t *testing.T) {
	s, l := shelved(t)
	l.onEmbeddings = `{"object":"list","data":[{"object":"embedding","embedding":[1.5,-2.5]},{"object":"embedding","index":1,"embedding":[0.3,0.4]}],"model":"embed-1","usage":{"prompt_tokens":5,"total_tokens":5}}`
	code, raw := postRetrieval(t, s, "/v1/embeddings", `{"model":"embed-1","input":"hi"}`)
	if code != 200 {
		t.Fatalf("%d %s", code, raw)
	}
	if !strings.Contains(raw, `"index":0`) || !strings.Contains(raw, `"index":1`) {
		t.Fatalf("indices were not filled: %s", raw)
	}
	if strings.Contains(raw, "1.5") == false || !strings.Contains(raw, `"embedding":[1.5,-2.5]`) {
		t.Fatalf("the first vector was not kept: %s", raw)
	}
	if !strings.Contains(raw, `"prompt_tokens":5`) {
		t.Fatalf("usage was not kept: %s", raw)
	}
	if strings.Contains(raw, "  ") || strings.Contains(raw, "\n") {
		t.Fatalf("the answer was re-indented: %s", raw)
	}
	l.onEmbeddings = `{"object":"list","data":[{"object":"embedding","index":null,"embedding":[9.9]}]}`
	code, raw = postRetrieval(t, s, "/embeddings", `{"model":"embed-1","input":"hi"}`)
	if code != 200 || !strings.Contains(raw, `"index":0`) || strings.Contains(raw, "null") {
		t.Fatalf("bare alias, null index: %d %s", code, raw)
	}
	l.onEmbeddings = `{"object":"list","data":[{"object":"embedding","index":7,"embedding":[0.1]},{"object":"embedding","index":0,"embedding":[0.2]}]}`
	_, raw = postRetrieval(t, s, "/v1/embeddings", `{"model":"embed-1","input":"hi"}`)
	if !strings.Contains(raw, `"index":7`) || !strings.Contains(raw, `"index":0`) {
		t.Fatalf("explicit indices were touched: %s", raw)
	}
}

// A rerank reply, a complete embeddings answer and anything that isn't an
// embeddings list go to the client as the vendor said them.
func TestEmbeddingIndicesPreserveOtherAnswers(t *testing.T) {
	s, l := shelved(t)
	_, raw := postRetrieval(t, s, "/v1/rerank", `{"model":"rerank-1","query":"q","documents":["a","b"]}`)
	if !strings.Contains(raw, `"relevance_score":0.9`) {
		t.Fatalf("rerank reply changed: %s", raw)
	}
	l.onEmbeddings = `{"object":"list","data":[{"object":"embedding","index":0,"embedding":[1.1]},{"object":"embedding","index":1,"embedding":[2.2]}]}`
	_, raw = postRetrieval(t, s, "/v1/embeddings", `{"model":"embed-1","input":"hi"}`)
	if !strings.Contains(raw, `"index":0`) || !strings.Contains(raw, `"index":1`) {
		t.Fatalf("a complete answer changed: %s", raw)
	}
	l.onEmbeddings = `not json at all`
	_, raw = postRetrieval(t, s, "/v1/embeddings", `{"model":"embed-1","input":"hi"}`)
	if !strings.Contains(raw, "not json at all") {
		t.Fatalf("a malformed answer changed: %s", raw)
	}
	l.onEmbeddings = ""
	if code, raw := postRetrieval(t, s, "/v1/embeddings", `{"model":"lib/chat-model","input":"hi"}`); code != 400 || !strings.Contains(raw, "not an embedding model") {
		t.Fatalf("error pass-through changed: %d %s", code, raw)
	}
}
