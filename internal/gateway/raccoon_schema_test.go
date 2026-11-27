package gateway

import (
	"github.com/yetone/magpie/internal/provider"
	"testing"
)

func TestRaccoonSchemaGroup(t *testing.T) {
	const model = "sn-glm-5-3-flash-p"
	raccoon := provider.PluginForTest("renamed", "raccoon")
	other := provider.PluginForTest("elsewhere", "other")
	bad := candidate{p: raccoon, model: model}
	good := candidate{p: other, model: model}
	for _, row := range []struct {
		proto provider.Protocol
		body  string
		skip  bool
	}{
		{provider.Chat, `{"response_format":{"type":"json_schema","json_schema":{"schema":{"type":"object"}}}}`, true},
		{provider.Responses, `{"text":{"format":{"type":"json_schema","schema":{"type":"object"}}}}`, true},
		{provider.Chat, `{"response_format":{"type":"json_object"}}`, false},
		{provider.Chat, `{"messages":[{"role":"user","content":"json_schema"}]}`, false},
	} {
		cs := []candidate{bad, good, {p: raccoon, model: "different"}}
		pl := planned{held: []candidate{bad, good}, capped: []candidate{bad, good}, order: []Weighed{{ID: "bad"}, {ID: "good"}, {ID: "other-model"}}}
		kept, plan, ok := raccoonSchemaGroup(row.proto, []byte(row.body), cs, pl)
		if !ok || len(kept) != len(plan.order) {
			t.Fatal("invalid group plan")
		}
		if row.skip {
			if len(kept) != 2 || kept[0].p.ID != good.p.ID || kept[1].model != "different" || plan.order[0].ID != "good" || len(plan.held) != 1 || len(plan.capped) != 1 {
				t.Fatal("incompatible member was not filtered correctly")
			}
			if _, _, ok := raccoonSchemaGroup(row.proto, []byte(row.body), []candidate{bad}, planned{}); ok {
				t.Fatal("incompatible group accepted")
			}
		} else if len(kept) != 3 {
			t.Fatal("ordinary request lost a member")
		}
	}
}
