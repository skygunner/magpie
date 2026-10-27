package gateway

import (
	"net/http"
	"testing"
)

func TestGLMPlanWindowSpentIsQuota(t *testing.T) {
	anthropicBody := `{"type":"rate_limit_error","message":"Zhipu GLM: [1310][您已达到每周/每月使用上限，您的限额将在 2026-10-24 20:44:45 重置。][20261020091843ac61fce173a347a5]"}`
	if got := failure(http.StatusTooManyRequests, []byte(anthropicBody)); got != failQuota {
		t.Fatalf("the Anthropic endpoint's refusal read as %s, want quota", got)
	}

	openAIBody := `{"error":{"message":"Zhipu GLM: [1310][您已达到每周/每月使用上限，您的限额将在 2026-10-24 20:44:45 重置。]"}}`
	if got := failure(http.StatusTooManyRequests, []byte(openAIBody)); got != failQuota {
		t.Fatalf("the OpenAI-shaped endpoint's refusal read as %s, want quota", got)
	}

	perMinute := `{"type":"rate_limit_error","message":"Zhipu GLM: [1302] 请求太频繁"}`
	if got := failure(http.StatusTooManyRequests, []byte(perMinute)); got != failRate {
		t.Fatalf("per-minute 1302 read as %s, want rate", got)
	}
}
