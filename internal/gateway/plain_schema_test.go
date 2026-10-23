package gateway

import (
	"encoding/json"
	"reflect"
	"testing"
)

// Antigravity refuses a function declaration over any keyword outside its
// schema subset (#187: exclusiveMinimum from zcode's and dsh's tools).
func TestPlainSchemaKeepsOnlyWhatAntigravityTakes(t *testing.T) {
	in := `{"type":"object","$schema":"http://json-schema.org/draft-07/schema#","additionalProperties":false,
	"properties":{
		"limit":{"type":"integer","exclusiveMinimum":0,"maximum":100,"description":"how many"},
		"offset":{"type":"integer","exclusiveMinimum":0,"minimum":1},
		"ratio":{"type":"number","exclusiveMaximum":1},
		"url":{"type":"string","format":"uri","title":"URL","minLength":1,"pattern":"^https?://"},
		"tags":{"type":"array","items":{"type":"string","uniqueItems":true},"minItems":1,"uniqueItems":true,"contains":{"type":"string"}},
		"opts":{"type":"object","properties":{"deep":{"type":["boolean","null"],"default":false}},"minProperties":1,"dependentRequired":{"a":["b"]},"not":{"type":"null"}}
	},"required":["limit","gone"]}`
	want := `{"type":"object",
	"properties":{
		"limit":{"type":"integer","minimum":0,"maximum":100,"description":"how many"},
		"offset":{"type":"integer","minimum":1},
		"ratio":{"type":"number","maximum":1},
		"url":{"type":"string","minLength":1,"pattern":"^https?://"},
		"tags":{"type":"array","items":{"type":"string"},"minItems":1},
		"opts":{"type":"object","properties":{"deep":{"type":"boolean","nullable":true}},"minProperties":1}
	},"required":["limit"]}`
	var got, exp any
	if err := json.Unmarshal(plainSchema(json.RawMessage(in)), &got); err != nil {
		t.Fatal(err)
	}
	json.Unmarshal([]byte(want), &exp)
	if !reflect.DeepEqual(got, exp) {
		b, _ := json.Marshal(got)
		t.Errorf("got %s", b)
	}
}

// A type union beside items keeps its array half: folded to its first
// member, ["string", "array"] with items became a string carrying items,
// and Antigravity holds Schema.items to type array — a 400 before anything
// was answered, on every turn that carried the tool (biomcp's search,
// #3121, rgoliveira). A union with no array half loses the array keywords
// rather than carrying them on another type.
func TestPlainSchemaUnionBesideItemsKeepsTheArrayHalf(t *testing.T) {
	disease := `{"type":["string","array"],"items":{"type":"string","minLength":1,"maxLength":256},"minItems":1,"maxItems":3,"minLength":1,"maxLength":256,"uniqueItems":true}`
	var got map[string]any
	if err := json.Unmarshal(plainSchema(json.RawMessage(disease)), &got); err != nil {
		t.Fatal(err)
	}
	if got["type"] != "array" || got["items"] == nil || got["minItems"] != float64(1) || got["maxItems"] != float64(3) {
		b, _ := json.Marshal(got)
		t.Fatalf("the union folded wrong: %s", b)
	}
	// a union with no array half: the string stands, items goes
	plain := `{"type":["string","null"],"items":{"type":"string"}}`
	got = nil
	if err := json.Unmarshal(plainSchema(json.RawMessage(plain)), &got); err != nil {
		t.Fatal(err)
	}
	if got["type"] != "string" || got["nullable"] != true || got["items"] != nil {
		b, _ := json.Marshal(got)
		t.Fatalf("the non-array union kept array keywords: %s", b)
	}
}

// The array half of a union leaves the object half's keywords behind with
// it: codex_apps' params are typed ["object","array"] and carry both halves'
// keywords, and Antigravity answers the same 400 over properties on an array
// as over items on a non-array (#3370, yxr1995-maker).
func TestPlainSchemaUnionArrayHalfLeavesObjectKeywordsBehind(t *testing.T) {
	codexApps := `{"type": ["object", "array"], "description": "Tool params as a real JSON object (an array for the handful of array-body tools) ...", "items": {"type": "string"}, "properties": {}, "minProperties": 1, "required": ["x"]}`
	var got map[string]any
	if err := json.Unmarshal(plainSchema(json.RawMessage(codexApps)), &got); err != nil {
		t.Fatal(err)
	}
	if got["type"] != "array" || got["items"] == nil {
		b, _ := json.Marshal(got)
		t.Fatalf("the union folded wrong: %s", b)
	}
	for _, k := range []string{"properties", "minProperties", "required"} {
		if got[k] != nil {
			b, _ := json.Marshal(got)
			t.Fatalf("the array half kept the object half's %s: %s", k, b)
		}
	}
}

// Either order of the union folds the same way: ["array","object"] beside
// both halves' keywords left the object half's properties on the array
// (#3370 review), where ["object","array"] did not.
func TestPlainSchemaUnionFoldIsOrderIndependent(t *testing.T) {
	both := `"items": {"type": "string"}, "properties": {}, "minProperties": 1, "maxProperties": 2, "required": ["x"]`
	for _, u := range []string{`{"type": ["object", "array"], ` + both + `}`, `{"type": ["array", "object"], ` + both + `}`} {
		var got map[string]any
		if err := json.Unmarshal(plainSchema(json.RawMessage(u)), &got); err != nil {
			t.Fatal(err)
		}
		if got["type"] != "array" || got["items"] == nil {
			b, _ := json.Marshal(got)
			t.Fatalf("%s folded wrong: %s", u, b)
		}
		for _, k := range []string{"properties", "minProperties", "maxProperties", "required"} {
			if got[k] != nil {
				b, _ := json.Marshal(got)
				t.Fatalf("%s kept the object half's %s: %s", u, k, b)
			}
		}
	}
	// a pure object keeps its properties and required
	var got map[string]any
	if err := json.Unmarshal(plainSchema(json.RawMessage(`{"type": "object", "properties": {"x": {"type": "string"}}, "required": ["x"]}`)), &got); err != nil {
		t.Fatal(err)
	}
	if got["properties"] == nil || got["required"] == nil {
		b, _ := json.Marshal(got)
		t.Fatalf("a plain object lost its keywords: %s", b)
	}
}
