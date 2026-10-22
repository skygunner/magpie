package edit

import (
	"bytes"
	"encoding/json"
	"github.com/tidwall/gjson"
	"github.com/tidwall/jsonc"
	"testing"
)

// A numeric key under a JSON array indexes it: the element at an index the
// array has is replaced, one at its length is appended, anything past that
// is refused. The array never becomes an object and no element is dropped.
func TestJSONSetArrayIndex(t *testing.T) {
	one := "{\n  \"customModels\": [\n    {\"model\": \"mine\"}\n  ]\n}\n"
	two := "{\n  \"customModels\": [\n    {\"model\": \"mine\"},\n    {\"model\": \"old\"}\n  ]\n}\n"
	tests := []struct {
		name    string
		in      string
		path    string
		value   any
		want    string // the file after SetJSON
		wantErr bool
	}{
		{
			name:  "replace the element at its index",
			in:    two,
			path:  "customModels.1",
			value: map[string]any{"model": "new"},
			want:  "{\n  \"customModels\": [\n    {\"model\": \"mine\"},\n    {\n      \"model\": \"new\"\n    }\n  ]\n}\n",
		},
		{
			name:  "replace a member of an element",
			in:    two,
			path:  "customModels.1.model",
			value: "new",
			want:  "{\n  \"customModels\": [\n    {\"model\": \"mine\"},\n    {\"model\": \"new\"}\n  ]\n}\n",
		},
		{
			name:  "insert a member into an element",
			in:    "{\n  \"customModels\": [\n    {\n      \"name\": \"mine\"\n    }\n  ]\n}\n",
			path:  "customModels.0.model",
			value: "x",
			want:  "{\n  \"customModels\": [\n    {\n      \"model\": \"x\",\n      \"name\": \"mine\"\n    }\n  ]\n}\n",
		},
		{
			name:  "append at the length",
			in:    one,
			path:  "customModels.1",
			value: map[string]any{"model": "other"},
			want:  "{\n  \"customModels\": [\n    {\"model\": \"mine\"},\n    {\n      \"model\": \"other\"\n    }\n  ]\n}\n",
		},
		{
			name:  "append at the length through a deeper key",
			in:    one,
			path:  "customModels.1.model",
			value: "new",
			want:  "{\n  \"customModels\": [\n    {\"model\": \"mine\"},\n    {\n      \"model\": \"new\"\n    }\n  ]\n}\n",
		},
		{
			name:  "append into an empty array",
			in:    "{\n  \"customModels\": []\n}\n",
			path:  "customModels.0",
			value: map[string]any{"model": "mine"},
			want:  "{\n  \"customModels\": [\n    {\n      \"model\": \"mine\"\n    }\n  ]\n}\n",
		},
		{
			name:  "create the array when the key is missing",
			in:    "{\n  \"a\": 1\n}\n",
			path:  "customModels.0.model",
			value: "x",
			want:  "{\n  \"customModels\": [\n    {\n      \"model\": \"x\"\n    }\n  ],\n  \"a\": 1\n}\n",
		},
		{
			name:    "refuse past the length",
			in:      one,
			path:    "customModels.2",
			value:   map[string]any{"model": "other"},
			wantErr: true,
		},
		{
			name:    "refuse a non-numeric key under an array",
			in:      one,
			path:    "customModels.model",
			value:   "x",
			wantErr: true,
		},
		{
			name:    "refuse creating past the array that is missing",
			in:      "{\n  \"a\": 1\n}\n",
			path:    "customModels.1.model",
			value:   "x",
			wantErr: true,
		},
		{
			name:  "a numeric key under an object stays a key",
			in:    "{\n  \"settings\": {}\n}\n",
			path:  "settings.0.model",
			value: "x",
			want:  "{\n  \"settings\": {\n    \"0\": {\n      \"model\": \"x\"\n    }\n  }\n}\n",
		},
		{
			name:  "append into a compact file",
			in:    `{"customModels":[{"model":"mine"}]}`,
			path:  "customModels.1",
			value: map[string]any{"model": "other"},
			want:  `{"customModels":[{"model":"mine"},{"model":"other"}]}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := tmpFile(t, "settings.json", tt.in)
			err := SetJSON(p, KV{Path: tt.path, Value: tt.value})
			if tt.wantErr {
				if err == nil {
					t.Fatalf("SetJSON(%q): want an error, got nil; file:\n%s", tt.path, read(t, p))
				}
				if got := read(t, p); got != tt.in {
					t.Fatalf("a refused set changed the file:\n%s\nwas:\n%s", got, tt.in)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got := read(t, p); got != tt.want {
				t.Fatalf("got:\n%s\nwant:\n%s", got, tt.want)
			}
		})
	}
}

// SetJSON and its preview share array semantics, including JSONC punctuation.
func TestJSONSetArrayComments(t *testing.T) {
	for _, in := range []string{
		`{"a":[/*keep*/]}`,
		"{\n  \"a\": [\n    // keep\n  ]\n}\n",
		`{"a":[{"model":"mine"},/*keep*/]}`,
		"{\n  \"a\": [\n    {\"model\":\"mine\"} // keep\n  ]\n}\n",
		"{\r\n  \"a\": [\r\n    {\"model\":\"mine\"}, // keep\r\n  ]\r\n}\r\n",
	} {
		t.Run(in, func(t *testing.T) {
			before := gjson.GetBytes(jsonc.ToJSON([]byte(in)), "a").Array()
			key := "a.0.model"
			if len(before) != 0 {
				key = "a.1.model"
			}
			kv := KV{Path: key, Value: "new"}
			preview, err := PatchJSON([]byte(in), []KV{kv}, nil)
			if err != nil {
				t.Fatal(err)
			}
			p := tmpFile(t, "settings.json", in)
			if err := SetJSON(p, kv); err != nil {
				t.Fatal(err)
			}
			out := []byte(read(t, p))
			clean := jsonc.ToJSON(out)
			if !json.Valid(clean) {
				t.Fatalf("invalid JSONC: %s", out)
			}
			a := gjson.GetBytes(clean, "a")
			if !a.IsArray() || len(a.Array()) != len(before)+1 || gjson.GetBytes(clean, key).String() != "new" {
				t.Fatalf("not appended: %s", out)
			}
			if len(before) > 0 && a.Array()[0].Raw != before[0].Raw {
				t.Fatalf("existing element changed: %s", out)
			}
			if !bytes.Contains(out, []byte("keep")) {
				t.Fatalf("comment lost: %s", out)
			}
			if !bytes.Equal(preview, out) {
				t.Fatalf("preview differs: %s vs %s", preview, out)
			}
			if bytes.Contains([]byte(in), []byte("\r\n")) && bytes.Contains(bytes.ReplaceAll(out, []byte("\r\n"), nil), []byte("\n")) {
				t.Fatalf("CRLF changed: %q", out)
			}
		})
	}
}

func TestJSONSetArrayRefusalKeepsAllKeys(t *testing.T) {
	in := `{"other":"mine","a":[{"model":"mine"}]}`
	p := tmpFile(t, "settings.json", in)
	if err := SetJSON(p, KV{"other", "changed"}, KV{"a.3.model", "new"}); err == nil {
		t.Fatal("past-end index was accepted")
	}
	if got := read(t, p); got != in {
		t.Fatalf("refused edit changed the file: %s", got)
	}
}

func TestJSONSetNestedArray(t *testing.T) {
	p := tmpFile(t, "settings.json", `{"other":true,"a":[{"nested":[{"model":"mine"}]}]}`)
	if err := SetJSON(p, KV{"a.0.nested.1.model", "new"}); err != nil {
		t.Fatal(err)
	}
	want := `{"other":true,"a":[{"nested":[{"model":"mine"},{"model":"new"}]}]}`
	if got := read(t, p); got != want {
		t.Fatalf("got %s, want %s", got, want)
	}
}
