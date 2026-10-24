package gateway

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
)

// Consecutive searches before the first answer text must retain every excerpt.
func TestReviewMultipleSearchCitations(t *testing.T) {
	first := []Hit{{Title: "One", URL: "https://one.example", Cited: "Excerpt one"}}
	second := []Hit{{Title: "Two", URL: "https://two.example", Cited: "Excerpt two"}}
	for _, tt := range []struct {
		name string
		two  []Hit
		want int
	}{
		{"two_searches", second, 2},
		{"later_empty_search", []Hit{{Title: "No excerpt", URL: "https://none.example"}}, 1},
	} {
		t.Run(tt.name, func(t *testing.T) {
			parts := []Part{{Kind: Search, Text: "first", Hits: first}, {Kind: Search, Text: "second", Hits: tt.two}, {Kind: Text, Text: "Summary of searches"}}
			var msg map[string]any
			if err := json.Unmarshal(renderAnthropic(Result{Parts: parts}, "m"), &msg); err != nil {
				t.Fatal(err)
			}
			count := 0
			for _, raw := range msg["content"].([]any) {
				p := raw.(map[string]any)
				if p["type"] == "text" {
					if c, ok := p["citations"].([]any); ok {
						count += len(c)
					}
				}
			}
			if count != tt.want {
				t.Errorf("whole search citations=%d want=%d", count, tt.want)
			}
			rec := httptest.NewRecorder()
			enc := &anthropicEncoder{w: newSSEWriter(rec), model: "m"}
			enc.event(Event{Kind: KSearch, Text: "first", Hits: first})
			enc.event(Event{Kind: KSearch, Text: "second", Hits: tt.two})
			enc.event(Event{Kind: KText, Text: "Summary of searches"})
			enc.finish()
			got := strings.Count(rec.Body.String(), "\"type\":\"web_search_result_location\"")
			if got != tt.want {
				t.Errorf("stream search citations=%d want=%d", got, tt.want)
			}
		})
	}
}
