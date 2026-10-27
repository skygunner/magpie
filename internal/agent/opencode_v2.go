package agent

import (
	"github.com/yetone/magpie/internal/edit"
)

// openCodeV2ProviderJSON is the v2 provider block for OpenCode 2+.
// It uses the "providers" top-level key instead of "provider".
func openCodeV2ProviderJSON(id, gw string) any {
	models := magpieModels("opencode")
	ms := map[string]any{}
	for _, m := range models {
		e := map[string]any{"name": m.Name}
		if m.Images {
			e["capabilities"] = map[string]any{"input": []string{"text", "image"}, "output": []string{"text"}}
		}
		if m.Context > 0 {
			e["limit"] = map[string]any{"context": m.Context, "output": maxTokens(m)}
		}
		if openCodeReasons(m) {
			e["reasoning"] = true
		}
		e["variants"] = openCodeVariants(m.Efforts, m.CodexFast)
		ms[m.ID] = e
	}
	return map[string]any{
		"package": "aisdk:@ai-sdk/openai-compatible",
		"name":    "magpie",
		"settings": map[string]any{
			"baseURL": gw + "/v1",
			"apiKey":  openCodeKeyAt("opencode", gw),
		},
		"models": ms,
	}
}

// openCodeV2Sync updates both the v1 (provider.magpie) and v2 (providers.magpie) blocks.
// It only writes the v2 block if the providers key already exists (OpenCode 2 format).
func openCodeV2Sync(path string, onMagpie func() bool, id string, at place) error {
	if !onMagpie() {
		return nil
	}
	// v1 block
	_, ok := edit.GetJSON(path, "provider."+magpieID)
	if !ok {
		// provider doesn't exist: create fresh
		v1 := magpieProviderJSONAt("opencode", id, at.gw())
		if err := edit.SetJSON(path, edit.KV{Path: "provider." + magpieID, Value: v1}); err != nil {
			return err
		}
	} else {
		// provider exists: preserve user's variants
		kept := openCodeKept(path, func() any { return magpieProviderJSONAt("opencode", id, at.gw()) })()
		if err := edit.SetJSON(path, edit.KV{Path: "provider." + magpieID, Value: kept}); err != nil {
			return err
		}
	}
	// v2 block - only write if providers key already exists (OpenCode 2 format)
	if _, ok := edit.GetJSON(path, "providers"); ok {
		v2 := openCodeV2ProviderJSON(id, at.gw())
		return edit.SetJSON(path, edit.KV{Path: "providers." + magpieID, Value: v2})
	}
	return nil
}

// openCodeV2SyncInOrder updates the v1 and v2 blocks in order, only if they already exist.
func openCodeV2SyncInOrder(path string, id string, at place) error {
	// v1 block - use openCodeKept to preserve user's variants
	kept := openCodeKept(path, func() any { return magpieProviderJSONAt("opencode", id, at.gw()) })()
	if err := syncJSONInOrder(path, "provider."+magpieID, func() any { return kept }); err != nil {
		return err
	}
	// v2 block - only update if it already exists
	if _, ok := edit.GetJSON(path, "providers"); ok {
		v2 := openCodeV2ProviderJSON(id, at.gw())
		if err := syncJSONInOrder(path, "providers."+magpieID, func() any { return v2 }); err != nil {
			return err
		}
	}
	return nil
}
