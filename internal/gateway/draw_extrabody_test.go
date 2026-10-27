package gateway

import (
	"net/http/httptest"
	"testing"

	"github.com/tidwall/gjson"
	"github.com/yetone/magpie/internal/provider"
)

// fromb612's relay (#3914): Ark's Seedream behind a company gateway draws
// with the "AI生成" mark. magpie asks for watermark false only of a provider
// at ark.*.volces.com (isArk), and a provider's extraBody reached no image
// request, so a relay had no way to ask for an image without it.
func TestExtraBodyReachesDrawnImage(t *testing.T) {
	fresh(t)
	e := &easel{}
	up := httptest.NewServer(e)
	t.Cleanup(up.Close)
	if err := provider.Save(provider.Provider{ID: "relay", Name: "内网网关", Chat: up.URL + "/v1", Key: "key", Models: []string{"Doubao-Seedream-5.0-pro", "painter-image-preview"}}); err != nil {
		t.Fatal(err)
	}
	withExtraBody(t, "relay", `{"watermark":false,"sequential_image_generation":"disabled"}`)

	s := New()
	code, _, raw := postImages(t, s, "/v1/images/generations", "application/json", `{"model":"relay/Doubao-Seedream-5.0-pro","prompt":"一只喜鹊"}`)
	if code != 200 {
		t.Fatalf("status %d: %s", code, raw)
	}
	sent := e.got("/v1/images/generations")
	if len(sent) != 1 {
		t.Fatalf("the vendor was sent %d requests: %v", len(sent), sent)
	}
	if got := gjson.Get(sent[0], "watermark"); got.Type != gjson.False {
		t.Errorf("sent without watermark false: %s", sent[0])
	}
	if got := gjson.Get(sent[0], "sequential_image_generation").String(); got != "disabled" {
		t.Errorf("sequential_image_generation %q in %s", got, sent[0])
	}
	// the drawing itself is magpie's, not the user's fields' to change
	if got := gjson.Get(sent[0], "prompt").String(); got != "一只喜鹊" {
		t.Errorf("prompt %q in %s", got, sent[0])
	}
	if got := gjson.Get(sent[0], "model").String(); got != "Doubao-Seedream-5.0-pro" {
		t.Errorf("model %q in %s", got, sent[0])
	}

	// a model magpie draws over chat completions takes them too
	code, _, raw = postImages(t, s, "/v1/images/generations", "application/json", `{"model":"relay/painter-image-preview","prompt":"一只喜鹊"}`)
	if code != 200 {
		t.Fatalf("chat drawing status %d: %s", code, raw)
	}
	chat := e.got("/v1/chat/completions")
	if len(chat) != 1 {
		t.Fatalf("the vendor was sent %d chat requests: %v", len(chat), chat)
	}
	if got := gjson.Get(chat[0], "watermark"); got.Type != gjson.False {
		t.Errorf("chat drawing sent without watermark false: %s", chat[0])
	}
}
