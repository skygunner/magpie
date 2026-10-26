package gateway

import "strings"

func zcodeTrialLimited(c candidate, body []byte) bool {
	if c.p.PluginProvider() != "zcode" || !strings.HasSuffix(strings.ToLower(c.model), "-trial") {
		return false
	}
	// Match the plugin's own message, including its translated provider prefix.
	said := strings.TrimPrefix(vendorMessage(body), c.p.Name+": ")
	return said == "ZCode has no active limited quota for "+c.model
}
