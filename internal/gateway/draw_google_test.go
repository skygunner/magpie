package gateway

import (
	"encoding/base64"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/yetone/magpie/internal/provider"
)

// A drawing at Google is asked for on Gemini's own API with the model
// named once, however it is spelled: Google's list gives it as "models/<id>",
// which a provider/model id keeps, and the path adds one "models/" itself
// (#3313: the doubled path matched no route and came back a bare 404).
func TestGoogleDrawingNamesTheModelOnce(t *testing.T) {
	fresh(t)
	if err := provider.Save(provider.Provider{ID: "google", Name: "Google Gemini", Key: "key",
		Chat: "https://generativelanguage.googleapis.com/v1beta/openai", Models: []string{"models/gemini-nano-banana-2.1"}}); err != nil {
		t.Fatal(err)
	}
	s := New()
	jpeg := []byte("\xff\xd8\xff\xe0 a fake jpeg")
	var path, sent string
	s.client = &http.Client{Transport: countTransport(func(r *http.Request) (*http.Response, error) {
		path = r.URL.Path
		b, _ := io.ReadAll(r.Body)
		sent = string(b)
		if r.URL.Path != "/v1beta/models/gemini-nano-banana-2.1:generateContent" {
			// Google answers a path no route matches with a bare 404
			return &http.Response{StatusCode: http.StatusNotFound, Header: http.Header{}, Body: http.NoBody}, nil
		}
		reply := `{"candidates":[{"content":{"parts":[{"inlineData":{"mimeType":"image/jpeg","data":"` +
			base64.StdEncoding.EncodeToString(jpeg) + `"}}]}}],"usageMetadata":{"promptTokenCount":9,"candidatesTokenCount":1290}}`
		return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": {"application/json"}},
			Body: io.NopCloser(strings.NewReader(reply))}, nil
	})}
	// the report's own request and Image generation model
	code, a, raw := postImages(t, s, "/v1/images/generations", "application/json",
		`{"model":"google/models/gemini-nano-banana-2.1","prompt":"a cat","n":1}`)
	if code != http.StatusOK || len(a.Data) != 1 {
		t.Fatalf("%d %s", code, raw)
	}
	if path != "/v1beta/models/gemini-nano-banana-2.1:generateContent" {
		t.Fatalf("Google was asked at %q", path)
	}
	if !strings.Contains(sent, `"responseModalities":["TEXT","IMAGE"]`) {
		t.Errorf("not asked for an image: %s", sent)
	}
	if b, _ := base64.StdEncoding.DecodeString(a.Data[0].B64); string(b) != string(jpeg) {
		t.Errorf("image %q", b)
	}
	if !strings.Contains(raw, `"mime_type":"image/jpeg"`) {
		t.Errorf("a JPEG's type is lost: %s", raw)
	}
}
