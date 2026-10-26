package redact

import (
	"strings"
	"testing"
)

// These probes distinguish a rule match from propagation of a password
// already encountered. They do not exempt ordinary words used as passwords.
func TestVerifierMaskingSources(t *testing.T) {
	for _, seed := range []string{"", "preview", "scrub", "url", "cjk"} {
		t.Run(seed, func(t *testing.T) {
			// Given: isolated learning and sent-text caches, with a synthetic seed.
			Forget()
			t.Cleanup(Forget)
			var hits []Hit
			o := Options{Secrets: true, Seen: func(h Hit) { hits = append(hits, h) }}
			const credential = "postgres://service:verifier@localhost/db"
			switch seed {
			case "preview":
				Preview(credential, o)
			case "scrub":
				Scrub(credential)
			case "url":
				Mask(credential, o)
			case "cjk":
				Mask("密码：verifier", o)
			}
			learned := seed == "url" || seed == "cjk"
			if learned && (len(hits) != 1 || hits[0].By != ByRule || hits[0].Kind != "PASSWORD") {
				t.Fatalf("seed did not match password rule: %+v", hits)
			}

			for _, signed := range []bool{false, true} {
				for _, text := range []string{"verifier nodes", "verifier-node tagged droplets"} {
					signature := ""
					if signed {
						signature = `,"signature":"synthetic"`
					}
					body := `{"messages":[{"role":"assistant","content":[{"type":"thinking","thinking":"` + text + `"` + signature + `}]}]}`
					hits = nil

					// When: the real JSON engine masks the reported thinking text.
					out, n := MaskJSON([]byte(body), o)

					// Then: only a learned password propagates, including before '-'.
					if learned {
						if n != 1 || len(hits) != 1 || hits[0].By != ByKnown || hits[0].Kind != "PASSWORD" {
							t.Fatalf("known password not masked: %s %+v", out, hits)
						}
						want := strings.Replace(body, "verifier", hits[0].Placeholder, 1)
						if string(out) != want {
							t.Fatalf("got %s, want %s", out, want)
						}
					} else if n != 0 || len(hits) != 0 || string(out) != body {
						t.Fatalf("unlearned word changed: %s %+v", out, hits)
					}
					if got := Restore(string(out), false); got != body {
						t.Fatalf("round trip changed: %s", got)
					}
					t.Logf("seed=%q signed=%t text=%q masked=%d hits=%+v output=%s", seed, signed, text, n, hits, out)
				}
			}
		})
	}
}
