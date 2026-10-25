package gateway

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/tidwall/gjson"
	"github.com/yetone/magpie/internal/provider"
)

// #3566 (congee949): Kimi Code refused a tool whose typed $ref target had
// an untyped oneOf of objects deeper inside it — 400 "not a valid moonshot
// flavored json schema … At path 'properties.x': detected infinite
// recursion without termination condition" — though the reporter's schema
// has no cycle and typing the union with object alone made the same route
// answer 200. A union of objects inside a typed $ref target is given the
// type its branches imply. A union of other types, one whose branches
// don't say, an anyOf, and a union that only leads back to itself are left
// as they were, as is every other provider's schema.
func TestKimiTypedRefObjectUnion(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	kimi, err := provider.FromPreset("kimi-code-cn")
	if err != nil {
		t.Fatal(err)
	}
	kimi.ID, kimi.Key, kimi.Models = "kimi", "k", []string{"m"}
	for _, p := range []provider.Provider{kimi,
		{ID: "other", Name: "Other", Key: "k", Chat: "https://other.test/v1", Anthropic: "https://other.test", Models: []string{"m"}},
	} {
		if err := provider.Save(p); err != nil {
			t.Fatal(err)
		}
	}
	got := map[string]string{}
	s := New()
	s.client = &http.Client{Transport: countTransport(func(r *http.Request) (*http.Response, error) {
		b, _ := io.ReadAll(r.Body)
		got[r.URL.Host+r.URL.Path] = string(b)
		return &http.Response{StatusCode: 400, Header: http.Header{"Content-Type": {"application/json"}},
			Body: io.NopCloser(strings.NewReader(`{"error":{"message":"no","type":"invalid_request_error"}}`))}, nil
	})}
	// the reporter's minimal parameters, with unions and a cycle beside
	// them that must stay as they are
	schema := `{"type":"object","properties":{"x":{"$ref":"#/$defs/X"},"z":{"$ref":"#/$defs/Z"}},` +
		`"$defs":{"X":{"type":"object","properties":{` +
		`"y":{"oneOf":[{"type":"object","properties":{"mode":{"type":"string","const":"a"}}},` +
		`{"type":"object","properties":{"mode":{"type":"string","const":"b"}}}]},` +
		`"mixed":{"oneOf":[{"type":"object","properties":{"n":{"type":"string"}}},{"type":"string"}]},` +
		`"silent":{"oneOf":[{}]},` +
		`"refs":{"anyOf":[{"type":"object","properties":{"n":{"type":"string"}}}]}},` +
		`"required":["y"]},` +
		`"Z":{"type":"object","properties":{"c":{"oneOf":[{"$ref":"#/$defs/Q"}]}},"required":["c"]},` +
		`"Q":{"$ref":"#/$defs/Q"}}}`
	want := strings.NewReplacer(
		`"y":{`, `"y":{"type":"object",`,
	).Replace(schema)
	for _, r := range []struct{ path, tools, upstream, otherUp string }{
		{"/v1/chat/completions", `"tools":[{"type":"function","function":{"name":"probe","parameters":%s}}]`, "api.kimi.com/coding/v1/chat/completions", "other.test/v1/chat/completions"},
		{"/v1/messages", `"tools":[{"name":"probe","input_schema":%s}]`, "api.kimi.com/coding/v1/messages", "other.test/v1/messages"},
	} {
		for _, id := range []string{"kimi", "other"} {
			clear(got)
			body := `{"model":"` + id + `/m","messages":[{"role":"user","content":"Do not call tools. Reply exactly OK."}],"max_tokens":16,` +
				strings.Replace(r.tools, "%s", schema, 1) + `}`
			rec := httptest.NewRecorder()
			s.Handler().ServeHTTP(rec, httptest.NewRequest("POST", r.path, strings.NewReader(body)))
			up := r.upstream
			if id == "other" {
				up = r.otherUp
			}
			b, ok := got[up]
			if !ok {
				t.Fatalf("%s %s: nothing sent to %s, only %v (%d %s)", r.path, id, up, got, rec.Code, rec.Body)
			}
			if !json.Valid([]byte(b)) {
				t.Fatalf("%s %s: invalid JSON sent:\n%s", r.path, id, b)
			}
			sent := gjson.Get(b, "tools.0.function.parameters")
			if !sent.Exists() {
				sent = gjson.Get(b, "tools.0.input_schema")
			}
			w := schema
			if id == "kimi" {
				w = want
			}
			if sent.Raw != w {
				t.Errorf("%s %s was sent:\n%s\nwant:\n%s", r.path, id, sent.Raw, w)
			}
		}
	}
}

// kimiToolRefTypes builds its body rather than writing into the one it is
// given, so a request's own bytes — a cache's prefix, another route's —
// stay as they were.
func TestKimiTypedRefObjectUnionLeavesItsInput(t *testing.T) {
	p, err := provider.FromPreset("kimi-code-cn")
	if err != nil {
		t.Fatal(err)
	}
	body := []byte(`{"model":"m","tools":[{"type":"function","function":{"name":"probe","parameters":` +
		`{"type":"object","properties":{"x":{"$ref":"#/$defs/X"}},` +
		`"$defs":{"X":{"type":"object","properties":{"y":{"oneOf":[{"type":"object","properties":{"mode":{"const":"a"}}}]}}}}}}}]}`)
	before := string(body)
	out := kimiToolRefTypes(p, provider.Chat, body)
	if string(body) != before {
		t.Fatalf("the request's bytes changed: %s", body)
	}
	if gjson.GetBytes(out, "tools.0.function.parameters.$defs.X.properties.y.type").String() != "object" {
		t.Fatalf("the union was not given object: %s", out)
	}
}
