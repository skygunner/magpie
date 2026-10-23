package provider

import (
	"context"
	"strings"
	"testing"
	"time"
)

// Antigravity lists a model a tier holds only switched off under an id
// ending -disabled (gemini-4-argon-disabled, "Gemini 4 Argon"): the id
// itself answers nothing — every request to it NOT_FOUND — so it is left
// out of the account's models instead of offered to pick (#3027,
// #3339, dreamfarer-space).
func TestAntigravityDisabledSuffixNotOffered(t *testing.T) {
	withStub := func(map[string]any) (int, string) {
		return 200, `{"models":{
		"gemini-3.8-flash-high":{"displayName":"Gemini 3.8 Flash (High)","maxTokens":1048576,"quotaInfo":{"remainingFraction":1}},
		"gemini-4-argon-disabled":{"displayName":"Gemini 4 Argon","maxTokens":1048576,"quotaInfo":{"remainingFraction":1}}}}`
	}
	f := &fakeGoogle{load: proLoad, modelsFor: withStub}
	googleSandbox(t, f)
	auth := googleAuth{AccessToken: "tok", RefreshToken: "rt-disabled", Expiry: time.Now().Add(time.Hour).UnixMilli()}
	if err := addGoogleLogin("antigravity", "disabled@example.com", "", auth); err != nil {
		t.Fatal(err)
	}
	got, err := googleLogins("antigravity")[0].acct.models(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	var ids []string
	for _, m := range got {
		ids = append(ids, m.ID)
	}
	if strings.Contains(strings.Join(ids, ","), "disabled") {
		t.Fatalf("a -disabled id was offered: %v", ids)
	}
	if len(ids) == 0 {
		t.Fatalf("no models at all: %v", ids)
	}
}
