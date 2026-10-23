package redact

import (
	"strings"
	"testing"
)

func TestReviewV2LearningRebuildAndRestore(t *testing.T) {
	resetSent()
	forgetKnown(t)
	t.Cleanup(func() { resetSent(); forgetKnown(t) })
	o := Options{Secrets: true}
	const promoted = "reviewUpgradeQ9r7"
	const placeholder = "reviewPlaceholderR8s3"
	for _, v := range []string{promoted, placeholder} {
		original := "postgres://" + v + ":" + v + "@localhost/db"
		masked, n := Mask(original, o)
		if n != 1 || Restore(masked, false) != original {
			t.Fatalf("placeholder password did not mask and restore: %q", masked)
		}
	}
	oldText := "before upgrade, the ordinary text holds " + promoted
	if out, n := Mask(oldText, o); n != 0 || out != oldText {
		t.Fatalf("placeholder polluted text before upgrade: %q", out)
	}
	actual := "postgres://actualuser:" + promoted + "@localhost/db"
	if out, n := Mask(actual, o); n != 1 || Restore(out, false) != actual {
		t.Fatalf("real password did not mask and restore: %q", out)
	}
	mu.Lock()
	reindexKnown()
	mu.Unlock()
	newText := "after rebuilding, a new sentence holds " + promoted
	if out, n := Mask(newText, o); n != 1 || strings.Contains(out, promoted) || Restore(out, false) != newText {
		t.Fatalf("promoted value did not survive rebuild: %q", out)
	}
	if out, n := Mask(oldText, o); n != 0 || out != oldText {
		t.Fatalf("a previously sent text changed after upgrade: %q", out)
	}
	plainText := "after rebuilding, another new sentence holds " + placeholder
	if out, n := Mask(plainText, o); n != 0 || out != plainText {
		t.Fatalf("unlearned value polluted text after rebuild: %q", out)
	}
	if out := Scrub(newText); strings.Contains(out, promoted) {
		t.Fatalf("local scrub forgot promoted value: %q", out)
	}
	if out := Scrub(plainText); out != plainText {
		t.Fatalf("local scrub learned a placeholder: %q", out)
	}
}
