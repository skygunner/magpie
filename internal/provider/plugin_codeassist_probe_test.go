package provider

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/yetone/magpie/internal/plugin"
	"github.com/yetone/magpie/internal/testenv"
)

// codeAssistVendor is the publisher endpoint a @ai-sdk/google plugin's
// fetch turns the Code Assist envelope into, as issue #3460's Vertex AI
// plugin does: it keeps what it was sent, and streams a Gemini answer.
func codeAssistVendor(t *testing.T) *[][]byte {
	t.Helper()
	var mu sync.Mutex
	got := &[][]byte{}
	testenv.FakeVendor(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		mu.Lock()
		*got = append(*got, append([]byte(r.URL.Path+"?"+r.URL.RawQuery+"\n"), b...))
		mu.Unlock()
		w.Header().Set("Content-Type", "text/event-stream")
		io.WriteString(w, `data: {"candidates":[{"content":{"parts":[{"text":"hi"}]}}]}`+"\n\n")
	}))
	return got
}

// fakePluginSignedIn is the fake plugin installed and one account signed
// in, as magpie lists it: its gcloud sign-in's provider is the one whose
// models issue #3460 reports on, fake-gemini on @ai-sdk/google.
func fakePluginSignedIn(t *testing.T, ctx context.Context) plugin.Provider {
	t.Helper()
	bun := testenv.Bun(t)
	claudeHome(t)
	t.Setenv("MAGPIE_BUN", bun)
	t.Cleanup(plugin.Settle)
	abs, _ := filepath.Abs("../plugin/testdata/fake/index.js")
	if _, err := plugin.Add(ctx, abs); err != nil {
		t.Fatal(err)
	}
	key, err := plugin.Import(ctx, "fakeco", map[string]any{"type": "oauth", "refresh": "r-ok", "access": "a", "expires": 9e15, "accountId": "ok@fake"})
	if err != nil {
		t.Fatal(err)
	}
	saveLogins(t, savedLogin{Agent: "plugin:fakeco", User: "ok@fake", Home: key, On: true})
	if _, err := plugin.Providers(ctx); err != nil {
		t.Fatal(err)
	}
	return mustPlugin(t)
}

// googleOnly is the plugin's provider holding only its @ai-sdk/google
// models, so Code Assist is all it speaks — the reporter's Vertex AI
// plugin, whose models are all Gemini's own.
func googleOnly(t *testing.T, pp plugin.Provider) Provider {
	t.Helper()
	var ms []plugin.Model
	for _, m := range pp.Models {
		if m.NPM == "@ai-sdk/google" {
			ms = append(ms, m)
		}
	}
	if len(ms) == 0 {
		t.Fatalf("the fake plugin lists no @ai-sdk/google model: %+v", pp.Models)
	}
	pp.Models = ms
	ls := pluginLogins(pp)
	if len(ls) == 0 {
		t.Fatal("the fake plugin's account isn't signed in")
	}
	return pluginProviderServing(pp, ls[0], nil)
}

// #3460: a plugin whose models are spoken to with @ai-sdk/google speaks
// Code Assist, and its fetch turns that envelope into the publisher's
// streamGenerateContent, so its models can be tested there rather than
// being turned away with "this provider can't be sent a test request".
func TestPluginGoogleModelIsTestable(t *testing.T) {
	got := codeAssistVendor(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	p := googleOnly(t, fakePluginSignedIn(t, ctx))

	if why := p.ModelTest(); why != "" {
		t.Errorf("the GUI's Test models is off: %q", why)
	}
	rs := p.TestModels(ctx, []string{"fake-gemini"})
	if len(rs) != 1 || !rs[0].OK || rs[0].Protocol != CodeAssist {
		t.Fatalf("fake-gemini tested: %+v", rs)
	}
	if len(*got) != 1 {
		t.Fatalf("the vendor got %d requests, want 1", len(*got))
	}
	line, body, _ := strings.Cut(string((*got)[0]), "\n")
	if !strings.HasPrefix(line, "/v1/models/fake-gemini:streamGenerateContent?alt=sse") {
		t.Errorf("sent to %q, want the publisher's streamGenerateContent", line)
	}
	var req struct {
		Contents []struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"contents"`
		GenerationConfig struct {
			MaxOutputTokens int `json:"maxOutputTokens"`
		} `json:"generationConfig"`
	}
	if err := json.Unmarshal([]byte(body), &req); err != nil || len(req.Contents) != 1 || req.Contents[0].Parts[0].Text != "hi" || req.GenerationConfig.MaxOutputTokens != 16 {
		t.Errorf("sent body %s: want Gemini's generateContent request (%v)", body, err)
	}
}

// #3460: the provider-level test (`magpie provider test <id>`) tries the
// Code Assist endpoint too, as it loops what the provider speaks.
func TestPluginTestProbesCodeAssist(t *testing.T) {
	got := codeAssistVendor(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	p := googleOnly(t, fakePluginSignedIn(t, ctx))

	for _, r := range p.Test(ctx) {
		if r.Protocol != CodeAssist {
			continue
		}
		if !r.OK || r.Model != "fake-gemini" {
			t.Fatalf("Code Assist tested: %+v", r)
		}
		for _, sent := range *got {
			line, _, _ := strings.Cut(string(sent), "\n")
			if strings.HasPrefix(line, "/v1/models/fake-gemini:streamGenerateContent?alt=sse") {
				return
			}
		}
		t.Fatalf("no probe reached the publisher's streamGenerateContent: %q", *got)
	}
	t.Fatalf("%s served no Code Assist probe", p.ID)
}
