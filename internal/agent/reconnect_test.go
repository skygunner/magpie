package agent

import (
	"encoding/json"
	"testing"
)

// #3868: reconnect should fail if the main connection (Joined/Beside)
// is not restored, even if auxiliary fields are present.
func TestReconnectRequiresMainConnection(t *testing.T) {
	home := syncHome(t)
	a := codex(home)
	rec := reconnection{
		Magpie: map[string]string{
			"sub_agent_model": magpieID,
			"memory_model":    magpieID,
			"model":           magpieID,
		},
		Routed: []string{"sub_agent_model", "memory_model", "model"},
		Left:   map[string]string{},
	}
	b, _ := json.Marshal(rec)
	stash(map[string]string{
		a.ID + ".reconnect": string(b),
	})
	defer forget(a.ID + ".reconnect")

	now := map[string]string{
		"sub_agent_model": magpieID,
		"memory_model":    magpieID,
		"model":           "other",
		"openai_base_url": "https://api.openai.com/v1",
	}

	ok, err := a.reconnect(now)
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("reconnect should fail when main connection (openai_base_url) is not magpie's")
	}
}
