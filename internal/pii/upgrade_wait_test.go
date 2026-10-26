package pii

import (
	"io"
	"net/http"
	"testing"
	"time"
)

// The issue's oracle is a race on the tests' own var writes, so the
// regression runs it through the helpers the tests put the vars back with:
// the first start's upgrade is held at its fetch (an event, not a delay), a
// second start returns at once with the upgrading CAS taken, and the tests'
// cleanups put the vars back — on a tree that joins only the last upgrade
// the held one is still reading when they do, and the race detector reports
// the pair (#3628).
func TestUpgradeAStartBeganIsWaitedForEvenWhenALaterOneCouldNotRun(t *testing.T) {
	gone := make(chan struct{})
	release := make(chan struct{})
	client = &http.Client{Transport: heldTransport{gone, release}}

	oldInstall(t, map[string][]byte{})

	// The release closes before the helpers' cleanups wait on the
	// upgrades: cleanups run last registered first, so this one — put in
	// after oldInstall's — lets the held fetch go before the joins.
	t.Cleanup(func() {
		t2 := http.DefaultTransport.(*http.Transport).Clone()
		t2.ResponseHeaderTimeout = 60 * time.Second
		client = &http.Client{Transport: t2}
		close(release)
	})

	if _, err := Find([]string{"hi Alice"}); err != nil {
		t.Fatal(err)
	}
	select {
	case <-gone:
	case <-time.After(30 * time.Second):
		t.Fatal("the first start's upgrade never fetched")
	}

	// The second start can only return at once — the upgrading CAS is
	// taken — and on a tree that keeps only the last upgrade's join it is
	// the one the cleanups' wait then finds.
	Stop()
	if _, err := Find([]string{"Bob Alice"}); err != nil {
		t.Fatal(err)
	}
}

// heldTransport parks the first request it is given at gone, and goes on
// once release closes; it answers nothing, so the upgrade fails its fetch
// and leaves the old pf-serve as it found it.
type heldTransport struct {
	gone    chan struct{}
	release chan struct{}
}

func (h heldTransport) RoundTrip(*http.Request) (*http.Response, error) {
	close(h.gone)
	<-h.release
	return nil, io.EOF
}
