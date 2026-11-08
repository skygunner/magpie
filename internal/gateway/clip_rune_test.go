package gateway

import (
	"context"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestOneOffErrRuneBoundary(t *testing.T) {
	for _, prefix := range []int{297, 298, 299, 300} {
		stderr := strings.Repeat("a", prefix) + "❯more"
		err := oneOffErr(context.Background(), stderr, nil)
		want := strings.Repeat("a", prefix)
		if prefix == 297 {
			want += "❯"
		}
		want = "Claude Code: " + want + "…"
		if err == nil || !utf8.ValidString(err.Error()) || err.Error() != want {
			t.Fatalf("prefix %d: got %v, want %q", prefix, err, want)
		}
	}
}
