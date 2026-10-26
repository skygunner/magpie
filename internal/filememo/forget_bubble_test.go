package filememo

import (
	"testing"
	"testing/synctest"
)

// The Written channel a loop waits on is made and closed by the same
// filememo: one that Forget made in a synctest bubble can't be closed by a
// later Forget from another bubble (provider's logins tests, one bubble per
// subtest, since #3185), so a channel is made only while someone waits.
func TestForgetsAcrossSynctestBubbles(t *testing.T) {
	synctest.Test(t, func(t *testing.T) { Forget() })
	synctest.Test(t, func(t *testing.T) { Forget() })
}
