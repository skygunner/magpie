package gateway

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"github.com/yetone/magpie/internal/provider"
	"github.com/yetone/magpie/internal/settings"
	"github.com/yetone/magpie/internal/usage"
)

func TestCodexImageRelayUsesConfiguredTimeout(t *testing.T) {
	setup(t, provider.Chat, &fake{t: t})
	chatgpt(t, func(w http.ResponseWriter, r *http.Request) { io.WriteString(w, `{}`) })
	cfg := settings.Load()
	cfg.ImageTimeout = 30
	if err := settings.Save(cfg); err != nil {
		t.Fatal(err)
	}
	s := New()
	d := &deadlines{next: s.client.Transport}
	s.client = &http.Client{Transport: d}
	req := httptest.NewRequest("POST", CodexPath+"/images/generations", strings.NewReader(`{"model":"gpt-image-2"}`))
	req.Header.Set("Authorization", "Bearer chatgpt-token")
	s.Handler().ServeHTTP(httptest.NewRecorder(), req)
	if len(d.got) != 1 || !d.draw[0] || d.got[0] < 29*time.Minute || d.got[0] > 30*time.Minute {
		t.Fatalf("image request deadline %v, drawing markers %v", d.got, d.draw)
	}
}

type imageTimeoutTransport struct{ started chan struct{} }

func (tr imageTimeoutTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	close(tr.started)
	<-r.Context().Done()
	return nil, r.Context().Err()
}

func TestCodexImageRelayTimeoutRecorded(t *testing.T) {
	setup(t, provider.Chat, &fake{t: t})
	chatgpt(t, func(w http.ResponseWriter, r *http.Request) { t.Error("unexpected HTTP request") })
	synctest.Test(t, func(t *testing.T) {
		started := make(chan struct{})
		s := New()
		s.client = &http.Client{Transport: imageTimeoutTransport{started}}
		req := httptest.NewRequest("POST", CodexPath+"/images/generations", strings.NewReader(`{"model":"gpt-image-2"}`))
		req.Header.Set("Authorization", "Bearer chatgpt-token")
		rec := httptest.NewRecorder()
		s.Handler().ServeHTTP(rec, req)
		select {
		case <-started:
		default:
			t.Fatal("image request did not reach transport")
		}
		routes, rows := s.Trace(t.Context(), 0, 0).Routes, usage.Load(time.Time{})
		if rec.Code != 504 || len(routes) != 1 || len(rows) != 1 {
			t.Fatalf("status %d, routes %d, rows %d", rec.Code, len(routes), len(rows))
		}
		if routes[0].Status != 504 || rows[0].Status != 504 || routes[0].Error == "" || rows[0].Error == "" {
			t.Fatalf("route %+v, row %+v", routes[0], rows[0])
		}
	})
}

func TestCodexImageRelayKeyRefusal(t *testing.T) {
	setup(t, provider.Chat, &fake{t: t})
	up := chatgpt(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		io.WriteString(w, `{"error":{"message":"invalid key"}}`)
	})
	old := codexAPIBase
	codexAPIBase = up.URL + "/backend-api/codex"
	t.Cleanup(func() { codexAPIBase = old })
	s := New()
	req := httptest.NewRequest("POST", CodexPath+"/images/generations", strings.NewReader(`{"model":"gpt-image-2"}`))
	req.Header.Set("Authorization", "Bearer sk-test-private")
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)
	routes, rows := s.Trace(t.Context(), 0, 0).Routes, usage.Load(time.Time{})
	if rec.Code != 401 || len(routes) != 1 || len(rows) != 1 {
		t.Fatalf("status %d, routes %d, rows %d", rec.Code, len(routes), len(rows))
	}
	if routes[0].Status != 401 || rows[0].Status != 401 || routes[0].Error == "" || rows[0].Error == "" {
		t.Fatalf("route %+v, row %+v", routes[0], rows[0])
	}
}

func TestCodexImageRelayConnectionFailure(t *testing.T) {
	setup(t, provider.Chat, &fake{t: t})
	chatgpt(t, func(w http.ResponseWriter, r *http.Request) {
		conn, _, err := w.(http.Hijacker).Hijack()
		if err != nil {
			t.Error(err)
			return
		}
		conn.Close()
	})
	s := New()
	req := httptest.NewRequest("POST", CodexPath+"/images/generations", strings.NewReader(`{"model":"gpt-image-2"}`))
	req.Header.Set("Authorization", "Bearer chatgpt-token")
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)
	routes, rows := s.Trace(t.Context(), 0, 0).Routes, usage.Load(time.Time{})
	if rec.Code != 502 || len(routes) != 1 || len(rows) != 1 {
		t.Fatalf("status %d, routes %d, rows %d", rec.Code, len(routes), len(rows))
	}
	if routes[0].Status != 502 || rows[0].Status != 502 || routes[0].Error == "" || rows[0].Error == "" {
		t.Fatalf("route %+v, row %+v", routes[0], rows[0])
	}
}

func TestCodexImageRelayRecorded(t *testing.T) {
	for _, tc := range []struct {
		name   string
		status int
		body   string
	}{
		{"success", 200, `{"data":[{"b64_json":"private-image-bytes"}]}`},
		{"failure", 400, `{"error":{"message":"image request refused"}}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			setup(t, provider.Chat, &fake{t: t})
			chatgpt(t, func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tc.status)
				io.WriteString(w, tc.body)
			})
			s := New()
			rec := httptest.NewRecorder()
			req := httptest.NewRequest("POST", CodexPath+"/images/generations", strings.NewReader(`{"model":"gpt-image-2","prompt":"private prompt"}`))
			req.Header.Set("Authorization", "Bearer private-credential")
			s.Handler().ServeHTTP(rec, req)
			if rec.Code != tc.status || rec.Body.String() != tc.body {
				t.Fatalf("response %d %q", rec.Code, rec.Body.String())
			}
			traces := s.Trace(t.Context(), 0, 0).Routes
			rows := usage.Load(time.Time{})
			if len(traces) != 1 || len(rows) != 1 {
				t.Fatalf("want one routing trace and request row: got %d and %d", len(traces), len(rows))
			}
			tr, row := traces[0], rows[0]
			if !tr.Done || tr.Status != tc.status || row.Status != tc.status || row.RouteID != tr.ID || row.Endpoint != req.URL.Path || row.Model != "gpt-image-2" {
				t.Fatalf("trace %+v, row %+v", tr, row)
			}
			if tc.status >= 400 && (tr.Error == "" || row.Error == "") {
				t.Fatalf("missing error: trace %+v, row %+v", tr, row)
			}
			encoded, err := json.Marshal([]any{tr, row})
			if err != nil {
				t.Fatal(err)
			}
			for _, secret := range []string{"private prompt", "private-image-bytes", "private-credential"} {
				if strings.Contains(string(encoded), secret) {
					t.Errorf("trace or persisted usage contains %q", secret)
				}
			}
		})
	}
}
