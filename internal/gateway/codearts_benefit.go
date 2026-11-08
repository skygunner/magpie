package gateway

import (
	"net/http"
	"strings"

	"github.com/tidwall/gjson"
	"github.com/yetone/magpie/internal/provider"
)

func codeartsBenefitMissing(p provider.Provider, status int, body []byte) bool {
	if status != http.StatusBadRequest || p.PluginProvider() != "codearts" || !gjson.ValidBytes(body) {
		return false
	}
	code, message := gjson.GetBytes(body, "code"), gjson.GetBytes(body, "message")
	if e := gjson.GetBytes(body, "error"); e.IsObject() {
		code, message = e.Get("code"), e.Get("message")
	}
	if code.Type != gjson.String || code.String() != "InferHub.4004.200" || message.Type != gjson.String {
		return false
	}
	text := strings.TrimSpace(message.String())
	return strings.EqualFold(text, "benefit not found") || strings.EqualFold(text, "benefit not found (InferHub.4004.200)") || strings.EqualFold(text, "CodeArts: benefit not found (InferHub.4004.200)")
}
