package gui

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/yetone/magpie/internal/library"
	"github.com/yetone/magpie/internal/mcpauth"
	"github.com/yetone/magpie/internal/mcpauth/mcpauthtest"
)

// A browser on another computer ends on the callback address this machine
// listens on, which won't load there: the address is pasted back and
// finishes the sign-in, as a provider sign-in's does (#3683).
func TestMCPSignInFinishesFromPastedCallbackURL(t *testing.T) {
	sandboxHome(t)
	f := mcpauthtest.New(t)
	if _, err := library.SaveServer("", library.Server{Name: "cloudflare-api", Transport: "http", URL: f.URL}); err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()
	Handler(webHost{}, nil).ServeHTTP(rec, httptest.NewRequest("POST", "/api/library/mcp-signin", strings.NewReader(`{"name":"cloudflare-api"}`)))
	if rec.Code != http.StatusOK {
		t.Fatalf("sign-in: %d %s", rec.Code, rec.Body)
	}
	var start struct {
		ID            string `json:"id"`
		URL           string `json:"url"`
		State         string `json:"state"`
		PasteCallback bool   `json:"pasteCallback"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &start); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { mcpauth.Cancel(start.ID) })
	if start.State != "waiting" || !start.PasteCallback {
		t.Fatalf("sign-in doesn't take a pasted callback URL: %+v", start)
	}
	// the sign-in page sends the browser to the callback port here
	c := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	resp, err := c.Get(start.URL)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	back := resp.Header.Get("Location")
	if !strings.Contains(back, "/callback?") {
		t.Fatalf("the sign-in page sent the browser to %q", back)
	}
	body, err := json.Marshal(map[string]string{"url": back})
	if err != nil {
		t.Fatal(err)
	}
	rec = httptest.NewRecorder()
	Handler(webHost{}, nil).ServeHTTP(rec, httptest.NewRequest("POST", "/api/library/mcp-signin/"+start.ID+"/callback", strings.NewReader(string(body))))
	if rec.Code != http.StatusOK {
		t.Fatalf("paste callback: %d %s", rec.Code, rec.Body)
	}
	if st, ok := mcpauth.Progress(start.ID); !ok || st.State != "done" {
		t.Fatalf("sign-in after pasting: %+v", st)
	}
	if f.Exchanged != 1 {
		t.Fatalf("the code was exchanged %d times", f.Exchanged)
	}
}
