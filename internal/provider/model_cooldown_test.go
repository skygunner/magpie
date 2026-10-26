package provider

import "testing"

func TestModelCooldownNormalizesAndClones(t *testing.T) {
	original := Provider{ModelCooldown: map[string]int{"m": 120, "n": 1, "off": 0, "negative": -1, " ": 60}}
	p := normalize(original)
	if len(p.ModelCooldown) != 2 || p.ModelCooldown["m"] != 60 || p.ModelCooldown["n"] != 1 {
		t.Fatalf("normalized cooldowns: %v", p.ModelCooldown)
	}
	copy := p.clone()
	copy.ModelCooldown["m"] = 2
	if p.ModelCooldown["m"] != 60 || original.ModelCooldown["m"] != 120 {
		t.Fatal("normalization or clone aliased the caller's map")
	}
}
