已重接到当前 main（`3a89dd53`，重接为 [`8ecb86f8`](https://github.com/skygunner/magpie/commit/8ecb86f8c2c886ca1117d82be201b2debc7ea1e0)，净 diff 对 main 只剩这一处）。

```diff
diff --git a/internal/gateway/continuation.go b/internal/gateway/continuation.go
index 19dcf807..9303cf82 100644
--- a/internal/gateway/continuation.go
+++ b/internal/gateway/continuation.go
@@ -541,7 +541,8 @@ func (s *Server) streamTranslated(w http.ResponseWriter, r *http.Request, p prov
 				serr = readSSEAlive(rd, func(_, data string) error {
 					tail.see([]byte(data))
 					st := upstreamStop([]byte(data))
-					u.add(Usage{Upstream: upstreamOf([]byte(data)), Stop: st, Tier: upstreamTier([]byte(data))})
+					up, slug := upstreamOf([]byte(data))
+					u.add(Usage{Upstream: up, CanonicalSlug: slug, Stop: st, Tier: upstreamTier([]byte(data))})
 					ended = ended || replyEnds(actual, data, st)
 					if err := dec(data, attemptSee); err != nil {
 						return err
diff --git a/internal/gateway/gateway.go b/internal/gateway/gateway.go
index d17c50d1..ae5b6d4c 100644
--- a/internal/gateway/gateway.go
+++ b/internal/gateway/gateway.go
@@ -2948,7 +2948,7 @@ func (s *Server) serve(w http.ResponseWriter, r *http.Request, from provider.Pro
 		wentTier = at.tier
 		ended = time.Now()
 		try := Try{ID: c.rest, Model: c.model, Effort: wentAt, Picked: picked, Fixed: fixed, RuleEffort: ruled, Fast: fast, Start: began, Done: true, Status: call.Status, Millis: ended.Sub(began).Milliseconds(), Error: call.Error, Queued: queued,
-			Served: call.Usage.Served, Upstream: call.Usage.Upstream, Pin: c.p.ClinePassAsked(c.model), Tier: wentTier, TierServed: call.Usage.Tier, Auto: autoPicked(),
+			Served: call.Usage.Served, Upstream: call.Usage.Upstream, CanonicalSlug: call.Usage.CanonicalSlug, Pin: c.p.ClinePassAsked(c.model), Tier: wentTier, TierServed: call.Usage.Tier, Auto: autoPicked(),
 			Net: nt.timing(began, queued)}
 		if resetFirst != nil {
 			// the reset spent for it before it was asked
@@ -3129,7 +3129,7 @@ func (s *Server) serve(w http.ResponseWriter, r *http.Request, from provider.Pro
 			if call.To != "" {
 				rec := usage.Record{RouteID: tr.ID, Time: began, Agent: call.Agent, Via: call.Via, Provider: call.Provider, Host: where, Model: c.model,
 					ProviderKeyID: providerKeyID, ProviderKeyName: providerKeyName, ProviderAccount: providerAccount,
-					Requested: call.Model, Served: call.Usage.Served, Upstream: call.Usage.Upstream,
+					Requested: call.Model, Served: call.Usage.Served, Upstream: call.Usage.Upstream, CanonicalSlug: call.Usage.CanonicalSlug,
 					Input: call.Usage.billed().Input, Output: call.Usage.billed().Output, CacheRead: call.Usage.billed().CacheRead,
 					CacheWrite: call.Usage.billed().CacheWrite, CacheWrite1h: call.Usage.billed().CacheWrite1h, CacheUnknown: call.Usage.billed().cacheNotCounted(), Reasoning: call.Usage.billed().Reasoning, Credits: call.Usage.Credits, Effort: wentAt, Millis: try.Millis, Status: call.Status,
 					TierAsked: tierAsked, TierSent: wentTier, TierServed: call.Usage.Tier,
@@ -3796,7 +3796,7 @@ func (s *Server) serve(w http.ResponseWriter, r *http.Request, from provider.Pro
 		}
 		if n := len(t.Tries); n > 0 && call.Status < 400 {
 			t.Served, t.Swapped, t.Routed = t.Tries[n-1].Served, t.Tries[n-1].Swapped, t.Tries[n-1].Routed
-			t.Upstream = t.Tries[n-1].Upstream
+			t.Upstream, t.CanonicalSlug = t.Tries[n-1].Upstream, t.Tries[n-1].CanonicalSlug
 		}
 		t.Archive = call.archiveName()
 	})
@@ -3806,7 +3806,7 @@ func (s *Server) serve(w http.ResponseWriter, r *http.Request, from provider.Pro
 	if call.To != "" {
 		rec := usage.Record{RouteID: tr.ID, Time: start, Agent: call.Agent, Via: call.Via, Provider: call.Provider, Host: where, Model: model,
 			ProviderKeyID: providerKeyID, ProviderKeyName: providerKeyName, ProviderAccount: providerAccount,
-			Requested: call.Model, Served: call.Usage.Served, Upstream: call.Usage.Upstream,
+			Requested: call.Model, Served: call.Usage.Served, Upstream: call.Usage.Upstream, CanonicalSlug: call.Usage.CanonicalSlug,
 			Input: call.Usage.billed().Input, Output: call.Usage.billed().Output, CacheRead: call.Usage.billed().CacheRead,
 			CacheWrite: call.Usage.billed().CacheWrite, CacheWrite1h: call.Usage.billed().CacheWrite1h, CacheUnknown: call.Usage.billed().cacheNotCounted(), Reasoning: call.Usage.billed().Reasoning, Credits: call.Usage.Credits, Effort: wentAt, Millis: call.Millis, Status: call.Status,
 			TierAsked: tierAsked, TierSent: wentTier, TierServed: call.Usage.Tier,
@@ -6275,7 +6275,8 @@ func (s *Server) translate(w http.ResponseWriter, r *http.Request, p provider.Pr
 	see, held := written(zenSee(zen, col.add))
 	err = readSSE(rd, func(_, data string) error {
 		tail.see([]byte(data))
-		u.add(Usage{Upstream: upstreamOf([]byte(data)), Tier: upstreamTier([]byte(data))})
+		up, slug := upstreamOf([]byte(data))
+		u.add(Usage{Upstream: up, CanonicalSlug: slug, Tier: upstreamTier([]byte(data))})
 		ended = ended || replyEnds(actual, data, upstreamStop([]byte(data)))
 		return dec(data, see)
 	})
diff --git a/internal/gateway/ir.go b/internal/gateway/ir.go
index 2909c4fc..628e1e3f 100644
--- a/internal/gateway/ir.go
+++ b/internal/gateway/ir.go
@@ -389,8 +389,11 @@ type Usage struct {
 	// one — which may not be the one it was asked for
 	Served string `json:"served,omitempty"`
 	// Upstream: the provider an aggregator says answered behind it
-	// (OpenRouter's "provider": DeepInfra, Novita …), when it says one
-	Upstream string `json:"upstream,omitempty"`
+	// (OpenRouter's "provider": DeepInfra, Novita …), when it says one,
+	// and CanonicalSlug the slug it names that provider by (ClinePass's
+	// deepseek/deepseek-v4.1-flash, private/… for a private pool, #3713)
+	Upstream      string `json:"upstream,omitempty"`
+	CanonicalSlug string `json:"canonicalSlug,omitempty"`
 	// RequestID: the id the vendor gave the request, from its reply's
 	// headers (Claude Code's own for a subscription); ErrType: what a
 	// failed request's error body called the error
@@ -471,6 +474,9 @@ func (u *Usage) add(v Usage) {
 	if v.Upstream != "" {
 		u.Upstream = v.Upstream
 	}
+	if v.CanonicalSlug != "" {
+		u.CanonicalSlug = v.CanonicalSlug
+	}
 	if v.Tier != "" {
 		u.Tier = v.Tier
 	}
diff --git a/internal/gateway/sniff.go b/internal/gateway/sniff.go
index d618a393..6d076b61 100644
--- a/internal/gateway/sniff.go
+++ b/internal/gateway/sniff.go
@@ -126,8 +126,13 @@ func (s *usageSniffer) parse(b []byte) {
 			s.u.ErrType = provider.ErrorType(body)
 		}
 	}
-	if up := upstreamOf(b); up != "" {
-		s.u.Upstream = up
+	if up, slug := upstreamOf(b); up != "" || slug != "" {
+		if up != "" {
+			s.u.Upstream = up
+		}
+		if slug != "" {
+			s.u.CanonicalSlug = slug
+		}
 	}
 	if st := upstreamStop(b); st != "" {
 		s.u.Stop = st
diff --git a/internal/gateway/trace.go b/internal/gateway/trace.go
index a837893f..fedf705f 100644
--- a/internal/gateway/trace.go
+++ b/internal/gateway/trace.go
@@ -131,8 +131,10 @@ type Route struct {
 	Served  string `json:"served,omitempty"`
 	Swapped bool   `json:"swapped,omitempty"`
 	Routed  bool   `json:"routed,omitempty"`
-	// Upstream: the provider an aggregator said answered behind it
-	Upstream string `json:"upstream,omitempty"`
+	// Upstream: the provider an aggregator said answered behind it, and
+	// CanonicalSlug the slug it named it by (#3713)
+	Upstream      string `json:"upstream,omitempty"`
+	CanonicalSlug string `json:"canonicalSlug,omitempty"`
 	// Prompt: what the prompt held, part by part, and the window it went
 	// into (prompt.go)
 	Prompt *Prompt `json:"prompt,omitempty"`
@@ -358,8 +360,10 @@ type Try struct {
 	Swapped bool   `json:"swapped,omitempty"`
 	Routed  bool   `json:"routed,omitempty"`
 	// Upstream: the provider an aggregator said answered behind it
-	// (OpenRouter's DeepInfra, Novita …)
-	Upstream string `json:"upstream,omitempty"`
+	// (OpenRouter's DeepInfra, Novita …), and CanonicalSlug the slug it
+	// named that provider by (ClinePass's, #3713)
+	Upstream      string `json:"upstream,omitempty"`
+	CanonicalSlug string `json:"canonicalSlug,omitempty"`
 	// Pin: the upstream the try asked the Cline API to serve it from
 	// (provider.ClinePassAsked), "" when it asked for none
 	Pin string `json:"pin,omitempty"`
diff --git a/internal/gateway/upstream.go b/internal/gateway/upstream.go
index 29c7f946..dc5eb002 100644
--- a/internal/gateway/upstream.go
+++ b/internal/gateway/upstream.go
@@ -9,25 +9,30 @@ import (
 )
 
 // upstreamOf is the provider an aggregator's reply (a JSON body, or one
-// event of its stream) says answered behind it, "" when it says none.
+// event of its stream) says answered behind it, "" when it says none, and
+// the canonical slug it names that provider by, "" when it names none.
 // OpenRouter names it in a top-level "provider" — a Chat body and every
 // chunk of its stream, an Anthropic Messages body — and in a Messages
 // stream's message_start, as message.provider (seen 2026-10-05: DeepInfra,
 // StreamLake). Its Responses replies name none. ClinePass (api.cline.bot)
 // names it in a Chat choice's provider_metadata, as
 // gateway.routing.finalProvider: under delta in a stream's chunk, under
-// message in one body (#2847, smart-lty). Only those places count: a
-// choice's content is the model's, and providerAttempts inside routing
-// lists every provider tried, not the one that answered.
-func upstreamOf(b []byte) string {
-	if !bytes.Contains(b, []byte(`"provider"`)) && !bytes.Contains(b, []byte(`"finalProvider"`)) {
-		return ""
+// message in one body (#2847, smart-lty); the slug there,
+// gateway.routing.canonicalSlug, is what tells the model really served
+// apart — the official deepseek/deepseek-v4.1-flash from a private pool's
+// private/… (#3713, xdeers2). Only those places count: a choice's content
+// is the model's, and providerAttempts and modelAttempts inside routing
+// list what was tried, not what answered.
+func upstreamOf(b []byte) (provider, slug string) {
+	if !bytes.Contains(b, []byte(`"provider"`)) && !bytes.Contains(b, []byte(`"finalProvider"`)) && !bytes.Contains(b, []byte(`"canonicalSlug"`)) {
+		return "", ""
 	}
 	type routed struct {
 		ProviderMetadata struct {
 			Gateway struct {
 				Routing struct {
 					FinalProvider json.RawMessage `json:"finalProvider"`
+					CanonicalSlug json.RawMessage `json:"canonicalSlug"`
 				} `json:"routing"`
 			} `json:"gateway"`
 		} `json:"provider_metadata"`
@@ -46,12 +51,19 @@ func upstreamOf(b []byte) string {
 		} `json:"choices"`
 	}
 	if json.Unmarshal(b, &v) != nil {
-		return ""
+		return "", ""
 	}
 	raws := []json.RawMessage{v.Provider, v.Message.Provider, v.Response.Provider}
+	slugs := []json.RawMessage(nil)
 	for _, c := range v.Choices {
 		raws = append(raws, c.Delta.ProviderMetadata.Gateway.Routing.FinalProvider, c.Message.ProviderMetadata.Gateway.Routing.FinalProvider)
+		slugs = append(slugs, c.Delta.ProviderMetadata.Gateway.Routing.CanonicalSlug, c.Message.ProviderMetadata.Gateway.Routing.CanonicalSlug)
 	}
+	return stringIn(raws), stringIn(slugs)
+}
+
+// stringIn is the first of raws that is a short, non-empty JSON string.
+func stringIn(raws []json.RawMessage) string {
 	for _, raw := range raws {
 		var s string // a request's provider options echoed back are an object
 		if json.Unmarshal(raw, &s) == nil {
diff --git a/internal/gateway/upstream_test.go b/internal/gateway/upstream_test.go
index 181277d2..015481c6 100644
--- a/internal/gateway/upstream_test.go
+++ b/internal/gateway/upstream_test.go
@@ -3,6 +3,7 @@ package gateway
 import (
 	"bufio"
 	"bytes"
+	"encoding/json"
 	"io"
 	"net/http"
 	"net/http/httptest"
@@ -135,7 +136,7 @@ func TestUpstreamOf(t *testing.T) {
 		`not json "provider"`:                                           "",
 		`{"type":"response.created","response":{"provider":"Baseten"}}`: "Baseten",
 	} {
-		if got := upstreamOf([]byte(in)); got != want {
+		if got, _ := upstreamOf([]byte(in)); got != want {
 			t.Errorf("upstreamOf(%s) = %q; want %q", in, got, want)
 		}
 	}
@@ -173,24 +174,87 @@ func TestUpstreamOfClinePass(t *testing.T) {
 			last = d
 		}
 	}
-	if got := upstreamOf([]byte(last)); got != "deepseek" {
-		t.Errorf("real ClinePass chunk: upstreamOf = %q; want deepseek", got)
+	const slug = "deepseek/deepseek-v4.1-flash"
+	if got, gs := upstreamOf([]byte(last)); got != "deepseek" || gs != slug {
+		t.Errorf("real ClinePass chunk: upstreamOf = %q, %q; want deepseek, %q", got, gs, slug)
 	}
 	whole := strings.Replace(last, `"delta":{"provider_metadata"`, `"message":{"role":"assistant","content":"hi","provider_metadata"`, 1)
-	if got := upstreamOf([]byte(whole)); got != "deepseek" {
-		t.Errorf("one body: upstreamOf = %q; want deepseek", got)
+	if got, gs := upstreamOf([]byte(whole)); got != "deepseek" || gs != slug {
+		t.Errorf("one body: upstreamOf = %q, %q; want deepseek, %q", got, gs, slug)
 	}
-	for in, want := range map[string]string{
-		`{"choices":[{"delta":{"provider_metadata":{"gateway":{"routing":{"finalProvider":"novita"}}}}}]}`:                                         "novita",
-		`{"choices":[{"delta":{"content":"provider_metadata gateway routing \"finalProvider\":\"deepseek\""}}]}`:                                   "",
-		`{"choices":[{"delta":{"provider_metadata":{"gateway":{"routing":{"modelAttempts":[{"providerAttempts":[{"provider":"deepseek"}]}]}}}}}]}`: "",
-		`{"choices":[{"delta":{"provider_metadata":{"gateway":{"routing":{"finalProvider":{"id":"x"}}}}}}]}`:                                       "",
-		`{"choices":[{"delta":{"tool_calls":[{"function":{"arguments":"{\"finalProvider\":\"x\"}"}}]}}]}`:                                          "",
-		`{"provider":"Novita","choices":[{"delta":{"provider_metadata":{"gateway":{"routing":{"finalProvider":"deepseek"}}}}}]}`:                   "Novita",
+	for _, c := range []struct{ in, want, slug string }{
+		{in: `{"choices":[{"delta":{"provider_metadata":{"gateway":{"routing":{"finalProvider":"novita"}}}}}]}`, want: "novita"},
+		{in: `{"choices":[{"delta":{"content":"provider_metadata gateway routing \"finalProvider\":\"deepseek\""}}]}`},
+		{in: `{"choices":[{"delta":{"provider_metadata":{"gateway":{"routing":{"modelAttempts":[{"providerAttempts":[{"provider":"deepseek"}]}]}}}}}]}`},
+		{in: `{"choices":[{"delta":{"provider_metadata":{"gateway":{"routing":{"finalProvider":{"id":"x"}}}}}}]}`},
+		{in: `{"choices":[{"delta":{"tool_calls":[{"function":{"arguments":"{\"finalProvider\":\"x\"}"}}]}}]}`},
+		{in: `{"provider":"Novita","choices":[{"delta":{"provider_metadata":{"gateway":{"routing":{"finalProvider":"deepseek"}}}}}]}`, want: "Novita"},
+		// the slug is the one beside the provider that answered, never one
+		// of what was tried: a private pool's slug in providerAttempts says
+		// nothing about the answer (#3713)
+		{in: `{"choices":[{"delta":{"provider_metadata":{"gateway":{"routing":{"canonicalSlug":"deepseek/deepseek-v4.1-flash","finalProvider":"deepseek"}}}}}]}`, want: "deepseek", slug: slug},
+		{in: `{"choices":[{"message":{"provider_metadata":{"gateway":{"routing":{"canonicalSlug":"private/deepseek-v4p1-flash-1017-contributor"}}}}}]}`, slug: "private/deepseek-v4p1-flash-1017-contributor"},
+		{in: `{"choices":[{"delta":{"provider_metadata":{"gateway":{"routing":{"modelAttempts":[{"providerAttempts":[{"canonicalSlug":"private/deepseek-v4p1-flash-1017-contributor","provider":"deepseek"}]}]}}}}}]}`},
+		{in: `{"choices":[{"delta":{"content":"gateway routing \"canonicalSlug\":\"private/x\""}}]}`},
+		{in: `{"choices":[{"delta":{"provider_metadata":{"gateway":{"routing":{"canonicalSlug":{"id":"x"}}}}}}]}`},
 	} {
-		if got := upstreamOf([]byte(in)); got != want {
-			t.Errorf("upstreamOf(%s) = %q; want %q", in, got, want)
+		if got, gs := upstreamOf([]byte(c.in)); got != c.want || gs != c.slug {
+			t.Errorf("upstreamOf(%s) = %q, %q; want %q, %q", c.in, got, gs, c.want, c.slug)
+		}
+	}
+}
+
+// The canonicalSlug a ClinePass reply names beside the provider it answered
+// from (#3713, xdeers2) is kept with it, on the route, its try, the request
+// log and the usage record: the provider name alone cannot tell a private
+// pool (openai-compatible-private, slug private/…) from the official one.
+func TestClinePassSlugRecorded(t *testing.T) {
+	const slug = "deepseek/deepseek-v4.1-flash"
+	stream := clinePassStream(t)
+	fresh(t)
+	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
+		w.Header().Set("Content-Type", "text/event-stream")
+		for _, l := range stream {
+			io.WriteString(w, l+"\n\n")
 		}
+	}))
+	t.Cleanup(up.Close)
+	p := provider.Provider{ID: "fake", Name: "Fake", Key: "k", Chat: up.URL + "/v1", Models: []string{"sol"}}
+	if err := provider.Save(p); err != nil {
+		t.Fatal(err)
+	}
+	body := `{"model":"fake/sol","max_tokens":100,"stream":true,"messages":[{"role":"user","content":"hi"}]}`
+	s := New()
+	rec := httptest.NewRecorder()
+	s.Handler().ServeHTTP(rec, httptest.NewRequest("POST", "/v1/chat/completions", strings.NewReader(body)))
+	if rec.Code != 200 {
+		t.Fatalf("%d %s", rec.Code, rec.Body)
+	}
+	carries := func(what string, v any) {
+		t.Helper()
+		b, err := json.Marshal(v)
+		if err != nil {
+			t.Fatal(err)
+		}
+		if !strings.Contains(string(b), `"canonicalSlug":"`+slug+`"`) {
+			t.Errorf("%s carries no canonicalSlug: %s", what, b)
+		}
+	}
+	if r := lastRoute(s); r.Upstream != "deepseek" || len(r.Tries) != 1 {
+		t.Fatalf("route upstream %q, tries %+v", r.Upstream, r.Tries)
+	} else {
+		carries("the route", r)
+		carries("its try", r.Tries[0])
+	}
+	if calls := s.Recent(); len(calls) == 0 {
+		t.Fatal("no request log")
+	} else {
+		carries("the request log", calls[0].Usage)
+	}
+	if u := usage.Load(time.Time{}); len(u) != 1 {
+		t.Fatalf("usage %+v", u)
+	} else {
+		carries("the usage record", u[0])
 	}
 }
 
diff --git a/internal/usage/request_page.go b/internal/usage/request_page.go
index fa256e98..b192b138 100644
--- a/internal/usage/request_page.go
+++ b/internal/usage/request_page.go
@@ -136,13 +136,13 @@ type rowChunk struct {
 // rowMsg is the Text of a row's Claude message id, after rowText's, and
 // rowError its error's
 const (
-	rowMsg   = 34
+	rowMsg   = 35
 	rowError = 9
 	// rowClient is the Text of the client's own name, when the user counts
 	// it as another agent (Row.Client, #1958)
 	rowClient = 33
 	// rowTexts is how many strings a row has: rowText's and the message id
-	rowTexts = 35
+	rowTexts = 36
 )
 
 // rowVaried are the strings that differ call by call: the error, the
@@ -208,8 +208,8 @@ func (c *rowChunk) shape(t [rowTexts]uint32) uint32 {
 // reason ("Upstream stop reason", #1927) and the service tiers (#1784)
 // among them: a string left out comes back empty from the cache, in the
 // page and in /v1/magpie/usage/requests (TestRowTextHoldsEveryString)
-func rowText(r *Row) [34]*string {
-	return [34]*string{&r.Agent, &r.Provider, &r.Host, &r.SessionProvider, &r.SessionAccount, &r.Model, &r.Requested, &r.Served, &r.Effort, &r.Error, &r.ErrType, &r.RequestID, &r.Endpoint, &r.Session, &r.NativeSession, &r.Kind, &r.Source, &r.Via, &r.ProviderKeyID, &r.ProviderKeyName, &r.CallerKeyID, &r.CallerKeyName, &r.Archive, &r.Operation, &r.ProviderAccount, &r.ResponseID, &r.Upstream, &r.Subagent, &r.ParentAgent, &r.Stop, &r.TierAsked, &r.TierSent, &r.TierServed, &r.Client}
+func rowText(r *Row) [35]*string {
+	return [35]*string{&r.Agent, &r.Provider, &r.Host, &r.SessionProvider, &r.SessionAccount, &r.Model, &r.Requested, &r.Served, &r.Effort, &r.Error, &r.ErrType, &r.RequestID, &r.Endpoint, &r.Session, &r.NativeSession, &r.Kind, &r.Source, &r.Via, &r.ProviderKeyID, &r.ProviderKeyName, &r.CallerKeyID, &r.CallerKeyName, &r.Archive, &r.Operation, &r.ProviderAccount, &r.ResponseID, &r.Upstream, &r.Subagent, &r.ParentAgent, &r.Stop, &r.TierAsked, &r.TierSent, &r.TierServed, &r.Client, &r.CanonicalSlug}
 }
 func (c *rowChunk) add(r Row, msg string, order int64, failed bool) {
 	if c.dict == nil {
diff --git a/internal/usage/usage.go b/internal/usage/usage.go
index b94e83f0..8bd7298c 100644
--- a/internal/usage/usage.go
+++ b/internal/usage/usage.go
@@ -59,10 +59,13 @@ type Record struct {
 	Requested string `json:"req,omitempty"`
 	Served    string `json:"served,omitempty"`
 	// Upstream: the provider an aggregator (OpenRouter …) said answered
-	// behind it — DeepInfra, Novita — when its reply named one
-	Upstream string `json:"upstream,omitempty"`
-	Input    int    `json:"in"`
-	Output   int    `json:"out"`
+	// behind it — DeepInfra, Novita — when its reply named one, and
+	// CanonicalSlug the slug it named that provider by (ClinePass's
+	// deepseek/deepseek-v4.1-flash, private/… for a private pool, #3713)
+	Upstream      string `json:"upstream,omitempty"`
+	CanonicalSlug string `json:"canonicalSlug,omitempty"`
+	Input         int    `json:"in"`
+	Output        int    `json:"out"`
 	// CacheRead is the prompt tokens read from the vendor's cache, and
 	// CacheWrite those written to it (Anthropic's cache creation,
 	// cache_creation_input_tokens); neither is in Input. Both are written
```

**重接后的红绿**（go1.26.8，linux/amd64）：

- 红：干净 `3a89dd53` + 一个只读 JSON 的独立测试（`TestRedProofSlugOnEveryRecord`，断言 route / try / 请求日志 / usage 记录四处都带 `"canonicalSlug":"deepseek/deepseek-v4.1-flash"`）→ 四处全红，例如 route `…"served":"deepseek/deepseek-v4.1-flash","swapped":true,"upstream":"deepseek"…` 没有该字段；
- 绿：`8ecb86f8` 同一测试通过；`go test -tags nogui ./internal/gateway/ -count=1 -run 'TestClinePassSlugRecorded|TestUpstreamOfClinePass|TestClinePassUpstreamRecorded'` 5/5 通过；usage 包 557/557。

一个如实的边界：补丁自带的 `TestUpstreamOfClinePass` 依赖 `upstreamOf` 改为双返回值，所以它在 base 上根本编不过（不是"红"而是编译错）；红证是上面那个不依赖签名的独立测试给出的。`go test -tags nogui ./internal/gateway/ -count=1` 在补丁后仍有 `TestClaudeClientCacheAbsentUnchanged` 一红，在干净 base 上同样红，与本次改动无关。

