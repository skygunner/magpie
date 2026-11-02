package redact

import (
	"regexp"
	"strings"
	"testing"
)

// A value masked once goes as its placeholder wherever it is next, in
// whatever words are around it: what the vendor wrote about it, as the
// agent sends it back.
func TestKnownValueMaskedInOtherWords(t *testing.T) {
	o := Options{Secrets: true, Personal: true}
	const secret, email, phone = "Zq8hunterPass42", "rina.k@acme-labs.io", "13912345670"
	ps, n := Mask("DB_PASSWORD="+secret, o)
	pe, _ := Mask("I am "+email, o)
	pp, _ := Mask("call "+phone, o)
	if n != 1 || strings.Contains(ps+pe+pp, secret) || strings.Contains(pe, email) || strings.Contains(pp, phone) {
		t.Fatalf("not masked to begin with: %q %q %q", ps, pe, pp)
	}
	ps, pe, pp = strings.TrimPrefix(ps, "DB_PASSWORD="), strings.TrimPrefix(pe, "I am "), strings.TrimPrefix(pp, "call ")

	// the vendor's reply as the agent has it back: no rule matches these
	reply := "Your password is " + secret + ", and I will write to " + email + ". Or call " + phone + "!"
	out, n := Mask(reply, o)
	if strings.Contains(out, secret) || strings.Contains(out, email) || strings.Contains(out, phone) {
		t.Fatalf("a known value went out: %q", out)
	}
	if want := "Your password is " + ps + ", and I will write to " + pe + ". Or call " + pp + "!"; out != want || n != 3 {
		t.Fatalf("%d masked: %q, want the same placeholders as before: %q", n, out, want)
	}
	if back := Restore(out, false); back != reply {
		t.Fatalf("round trip: %q", back)
	}

	// in a request body, as the next turn's history has it
	body := `{"messages":[{"role":"assistant","content":"` + reply + `"},{"role":"user","content":"thanks"}]}`
	if got, n := MaskJSON([]byte(body), o); strings.Contains(string(got), secret) || strings.Contains(string(got), email) || strings.Contains(string(got), phone) || n != 3 {
		t.Fatalf("%d masked: %s", n, got)
	}

	// only what the settings mask: personal data off, secrets off
	if out, _ := Mask(reply, Options{Secrets: true}); !strings.Contains(out, email) || !strings.Contains(out, phone) || strings.Contains(out, secret) {
		t.Fatalf("personal data off: %q", out)
	}
	if out, _ := Mask(reply, Options{Personal: true}); !strings.Contains(out, secret) || strings.Contains(out, email) {
		t.Fatalf("secrets off: %q", out)
	}

	// a piece of a longer number or address is another one
	for _, s := range []string{"order 2" + phone, "order " + phone + "9", "mail " + email + ".cn now", "mail x" + email} {
		if out, _ := Mask(s, Options{Personal: true}); strings.Contains(out, pp) || strings.Contains(out, pe) {
			t.Errorf("%q: masked as the known value: %q", s, out)
		}
	}
	// a secret in other punctuation is the secret still; inside a longer
	// word it is another word
	if out, _ := Mask("`"+secret+"`/"+secret+"_v2", Options{Secrets: true}); strings.Contains(out, secret) {
		t.Errorf("secret left: %q", out)
	}
	if out, _ := Mask("pass"+secret+"x", Options{Secrets: true}); out != "pass"+secret+"x" {
		t.Errorf("a longer word masked: %q", out)
	}
}

// A value too short to be told from ordinary text is masked where a rule
// finds it, and nowhere else.
func TestShortValueNotMaskedEverywhere(t *testing.T) {
	o := Options{Secrets: true}
	if out, n := Mask("postgres://app:ab12cd@db/app", o); n != 1 || strings.Contains(out, "ab12cd") {
		t.Fatalf("the rule should mask it: %q", out)
	}
	if out, n := Mask("hex ab12cd0 and ab12cd", o); n != 0 || out != "hex ab12cd0 and ab12cd" {
		t.Fatalf("a short value masked everywhere: %q", out)
	}
}

// What a vendor sealed has a secret masked before as its placeholder, the
// way the vendor wrote it, though nothing else in the request has it; a
// number or address of its own it keeps, though one masked before.
func TestKnownInSigned(t *testing.T) {
	o := Options{Secrets: true, Personal: true}
	const secret, phone = "Wv7otterPass93", "13612345674"
	Mask("DB_PASSWORD="+secret, o)
	Mask("call "+phone, o)
	body := `{"messages":[{"role":"assistant","content":[{"type":"thinking","thinking":"the password is ` + secret + `; order ` + phone + `","signature":"Eqk1"}]}]}`
	out, _ := MaskJSON([]byte(body), o)
	if strings.Contains(string(out), secret) || !strings.Contains(string(out), "order "+phone) {
		t.Fatalf("%s", out)
	}
}

// After values are let go, the ones still held are matched as before.
func TestKnownReindexed(t *testing.T) {
	o := Options{Secrets: true}
	const secret = "Kp3heronPass71"
	p, _ := Mask("API_TOKEN="+secret, o)
	mu.Lock()
	reindexKnown()
	mu.Unlock()
	if out, _ := Mask("it is "+secret, o); out != "it is "+strings.TrimPrefix(p, "API_TOKEN=") {
		t.Fatalf("%q", out)
	}
}

// The word an example credential is given that names the credential
// itself — postgresql://example:password@localhost/test — is masked in
// the URL's password slot, as ever, and not learned: the ordinary
// sentences that use the word went as {{PASSWORD_…}} afterwards (#3365,
// Phantomthedog, 4,888 request records).
func TestCredentialOwnNameNotLearned(t *testing.T) {
	Forget()
	t.Cleanup(Forget)
	o := Options{Secrets: true, Personal: true}
	seed := "postgresql://example:password@localhost/test"
	if out, n := Mask(seed, o); n != 1 || !regexp.MustCompile(`^postgresql://example:\{\{PASSWORD_[a-z2-7]{8}\}\}@localhost/test$`).MatchString(out) {
		t.Fatalf("%d masked: %q", n, out)
	}
	// the word alone is no credential: a sentence and an assignment alike
	// go as written
	for _, s := range []string{
		"Review SSH password authentication now.",
		"DB_PASSWORD=password",
	} {
		if out, n := Mask(s, o); out != s || n != 0 {
			t.Errorf("%d masked: %q went as %q", n, s, out)
		}
	}

	// a real password still learns from the same slot, and is masked in
	// prose after
	if _, n := Mask("postgresql://example:Tr0ub4dor3@localhost/test", o); n != 1 {
		t.Fatalf("the rule should mask it")
	}
	if out, n := Mask("the password is Tr0ub4dor3, changed yesterday", o); n != 1 || strings.Contains(out, "Tr0ub4dor3") {
		t.Errorf("%d masked: %q", n, out)
	}

	// a bare word the list does not name is a password like any other:
	// learned, masked wherever it is next. The list is closed; whether a
	// dictionary of ordinary words should keep their look from learning is
	// a wider choice than this fix makes.
	Mask("postgresql://example:strawberry@localhost/test", o)
	if out, n := Mask("cd ~/src/strawberry && ls", o); n != 1 || strings.Contains(out, "strawberry") {
		t.Errorf("%d masked: %q", n, out)
	}
	// a capital in it is enough: Password is someone's password too,
	// learned as before, and masked in prose that spells it so
	Mask("postgresql://example:Password@localhost/test", o)
	if out, n := Mask("Review SSH Password authentication now.", o); n != 1 || strings.Contains(out, "Password") {
		t.Errorf("%d masked: %q", n, out)
	}
}
