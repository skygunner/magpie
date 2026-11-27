package gateway

import (
	"slices"

	"github.com/tidwall/gjson"
	"github.com/yetone/magpie/internal/provider"
)

// Raccoon's GLM Flash accepts json_object but refuses even a single-property
// json_schema. Keep the schema intact for another member before applying affinity.
func raccoonSchemaGroup(from provider.Protocol, body []byte, cs []candidate, pl planned) ([]candidate, planned, bool) {
	var path string
	switch from {
	case provider.Chat:
		path = "response_format.type"
	case provider.Responses:
		path = "text.format.type"
	default:
		return cs, pl, true
	}
	if gjson.GetBytes(body, path).Str != "json_schema" {
		return cs, pl, true
	}
	incompatible := func(c candidate) bool {
		return c.p.PluginProvider() == "raccoon" && c.p.OwnModelID(c.model) == "sn-glm-5-3-flash-p"
	}
	if !slices.ContainsFunc(cs, incompatible) && !slices.ContainsFunc(pl.held, incompatible) && !slices.ContainsFunc(pl.capped, incompatible) {
		return cs, pl, true
	}
	return keepCandidates(cs, pl, func(c candidate) bool { return !incompatible(c) })
}
