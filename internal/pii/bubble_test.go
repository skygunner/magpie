package pii

import (
	"errors"
	"sync/atomic"
	"testing"
	"testing/synctest"
)

// A request in a synctest bubble leaves nothing in srv that a Stop
// outside it closes. srv.stop was made by the first request and closed
// by Stop: made in the bubble, closed outside it, synctest kills the
// whole test binary ("close of synctest channel from outside bubble"),
// and every test of the package after it with it (#3722).
func TestStopAfterARequestInABubbleClosesNothing(t *testing.T) {
	was := dir
	dir = func() string { return t.TempDir() } // no pf-serve: the request ends before a load
	t.Cleanup(func() { dir = was })
	synctest.Test(t, func(t *testing.T) {
		if _, err := classify([]string{"hi Alice"}); !errors.Is(err, ErrNotInstalled) {
			t.Fatalf("err = %v, want ErrNotInstalled", err)
		}
	})
	Stop()
	if st := Now(); st.Running || st.Loading {
		t.Fatalf("Now = %+v after Stop", st)
	}
}

// The idle timer an answered request arms in a synctest bubble is not
// stopped by a Stop outside it: detachLocked stopped it, and synctest
// kills the test binary for that too ("stop of synctest timer from
// outside bubble") (#3722).
func TestStopAfterAnAnsweredRequestInABubbleStopsNoTimer(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		idleLocked(&helper{}) // as an answered request arms it
	})
	Stop()
}

// AstrLink Guard's idle timer is the same, armed by a request in
// guardClassifyTurn and stopped by guardUnload, which Stop calls from
// outside the bubble (#3722).
func TestGuardStopAfterARequestInABubbleStopsNoTimer(t *testing.T) {
	guardSandbox(t)
	fakeGuardFiles(t)
	var reads atomic.Int32
	guardLoad = func(string) (spanFinder, error) {
		return fakeFinder{nil, &reads}, nil
	}
	synctest.Test(t, func(t *testing.T) {
		if _, err := guardClassifyTurn([]string{"one"}); err != nil {
			t.Fatal(err)
		}
	})
	guardModel{}.Stop()
	if st := (guardModel{}).Now(); st.Running {
		t.Fatalf("Now = %+v after Stop", st)
	}
}
