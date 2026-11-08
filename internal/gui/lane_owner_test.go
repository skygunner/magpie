package gui

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/yetone/magpie/internal/gateway"
)

// The lane limit fixture must not read another gateway's lanes (#5150).
func TestLaneLimitRoutesIsolatedFromOwner(t *testing.T) {
	owner := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/magpie/concurrency" {
			w.Write([]byte(`{"data":{"fixture-owner":{"busy":3,"waiting":0,"limit":0}}}`))
			return
		}
		http.NotFound(w, r)
	}))
	defer owner.Close()
	t.Setenv("MAGPIE_ADDR", strings.TrimPrefix(owner.URL, "http://"))
	TestLaneLimitRoutes(t)
}

// A console without a local gateway displays its owner's lanes (#5150).
func TestLanesShowRemoteOwner(t *testing.T) {
	owner := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/magpie/concurrency" {
			http.NotFound(w, r)
			return
		}
		w.Write([]byte(`{"data":{"fixture-owner":{"busy":3,"waiting":0,"limit":0}}}`))
	}))
	defer owner.Close()
	t.Setenv("MAGPIE_ADDR", strings.TrimPrefix(owner.URL, "http://"))
	previous := served.Swap(nil)
	t.Cleanup(func() { served.Store(previous) })

	mux := http.NewServeMux()
	providerRoutes(mux, nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest("GET", "/api/lanes", nil))
	var got map[string]gateway.Lane
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if w.Code != http.StatusOK || len(got) != 1 || got["fixture-owner"].Busy != 3 {
		t.Fatalf("owner lanes: %d %s", w.Code, w.Body)
	}
}
