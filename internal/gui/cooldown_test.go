package gui

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/yetone/magpie/internal/provider"
)

func TestCooldownAPI(t *testing.T) {
	home := t.TempDir()
	for _, name := range []string{"HOME", "USERPROFILE", "XDG_CONFIG_HOME", "APPDATA", "LOCALAPPDATA"} {
		t.Setenv(name, home)
	}
	p := provider.Provider{ID: "cooldown-api", Chat: "https://example.invalid/v1", Key: "secret", QuotaRest: 540, RestCap: 20}
	if err := provider.Save(p); err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	cooldownRoutes(mux)
	call := func(method, url, body string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, httptest.NewRequest(method, url, strings.NewReader(body)))
		return w
	}
	url := "/api/cooldown?id=cooldown-api&ref=" + provider.KeyID("secret")
	w := call("POST", url, `{"rateCap":64,"quotaRest":30,"models":{"m":60}}`)
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body)
	}
	var got struct{ Effective provider.CooldownValues }
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Effective.RateCap != 64 || got.Effective.QuotaRest != 30 || got.Effective.Source["rateCap"] != "account" {
		t.Fatalf("%#v", got)
	}
	if strings.Contains(w.Body.String(), "secret") {
		t.Fatal("key leaked")
	}
	w = call("POST", url, `{"rateCap":0}`)
	if w.Code == 200 {
		t.Fatal("accepted invalid cap")
	}
	w = call("POST", "/api/cooldown?id=cooldown-api&ref=unknown", `{}`)
	if w.Code == 200 {
		t.Fatal("unknown key accepted")
	}
	w = call("POST", url, `{}`)
	if w.Code != 200 {
		t.Fatal(w.Body)
	}
	w = call("GET", url, "")
	_ = json.Unmarshal(w.Body.Bytes(), &got)
	if got.Effective.RateCap != 20 || got.Effective.QuotaRest != 540 {
		t.Fatal("inherit did not restore legacy", w.Body)
	}
}

func TestCooldownGlobalValidationAndSave(t *testing.T) {
	home := t.TempDir()
	for _, name := range []string{"HOME", "USERPROFILE", "XDG_CONFIG_HOME", "APPDATA", "LOCALAPPDATA"} {
		t.Setenv(name, home)
	}
	mux := http.NewServeMux()
	cooldownRoutes(mux)
	post := func(body string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, httptest.NewRequest("POST", "/api/cooldown", strings.NewReader(body)))
		return w
	}
	if w := post(`{"rateStart":1000}`); w.Code == 200 {
		t.Fatal("accepted start above inherited cap")
	}
	if w := post(`{"rateStart":4,"rateCap":32,"quotaRest":40}`); w.Code != 200 {
		t.Fatal(w.Code, w.Body)
	}
	t.Cleanup(func() { post(`{}`) })
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest("GET", "/api/cooldown", nil))
	var got struct{ Effective provider.CooldownValues }
	_ = json.Unmarshal(w.Body.Bytes(), &got)
	if got.Effective.RateStart != 4 || got.Effective.RateCap != 32 || got.Effective.QuotaRest != 40 {
		t.Fatalf("%s", w.Body)
	}
}

func TestCooldownListsGroupResponseWait(t *testing.T) {
	home := t.TempDir()
	for _, name := range []string{"HOME", "USERPROFILE", "XDG_CONFIG_HOME", "APPDATA", "LOCALAPPDATA"} {
		t.Setenv(name, home)
	}
	remove := provider.ServeGroups()
	defer remove()
	p := provider.Provider{ID: "cooldown-wait", Chat: "https://example.invalid/v1", Key: "k", Models: []string{"m"}}
	if err := provider.Save(p); err != nil {
		t.Fatal(err)
	}
	if err := provider.SaveGroup(provider.Group{ID: "cooldown-wait", Name: "Wait", Members: []string{p.ID + "/m"}, FirstToken: 25}); err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	cooldownRoutes(mux)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest("GET", "/api/cooldown?id="+p.ID, nil))
	var got struct {
		Groups []struct {
			Model   string
			Seconds int
		}
	}
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if len(got.Groups) == 0 || got.Groups[0].Seconds != 25 {
		t.Fatalf("missing group wait: %s", w.Body)
	}
}
