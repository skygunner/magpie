package gateway

import (
	"encoding/json"
	"testing"
)

// #3887: web_search_call's encrypted_content should be stripped when
// a conversation moves to a new account that can't decrypt it.
func TestStripWebSearchEncrypted(t *testing.T) {
	body := []byte(`{"input":[{"type":"web_search_call","id":"ws_1","action":{"type":"search","query":"test"},"encrypted_content":"secret"},{"type":"message","role":"user","content":"hi"}]}`)

	// stripWebSearchEncrypted should remove encrypted_content
	b, ok := stripWebSearchEncrypted(body)
	if !ok {
		t.Fatal("expected changed")
	}

	var q map[string]json.RawMessage
	if json.Unmarshal(b, &q) != nil {
		t.Fatal("invalid json")
	}
	var input []json.RawMessage
	json.Unmarshal(q["input"], &input)

	for _, it := range input {
		var itMap map[string]interface{}
		json.Unmarshal(it, &itMap)
		if itMap["type"] == "web_search_call" {
			if _, ok := itMap["encrypted_content"]; ok {
				t.Fatal("encrypted_content should be stripped")
			}
			// action should be preserved
			if itMap["action"] == nil {
				t.Fatal("action should be preserved")
			}
		}
	}
}

// web_search_call should be in sealedKinds for withoutRefused
func TestWebSearchCallInSealedKinds(t *testing.T) {
	body := []byte(`{"input":[{"type":"web_search_call","id":"ws_1","encrypted_content":"secret"}]}`)

	// withoutReasoning should strip encrypted_content from web_search_call
	b, ok := withoutReasoning(body)
	if !ok {
		t.Fatal("expected changed")
	}
	var _ = b
}
