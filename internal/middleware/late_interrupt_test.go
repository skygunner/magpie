package middleware

import (
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"github.com/Calcium-Ion/moejs"
)

// A hook can end just as its time runs out: the timer's func has begun
// and not yet given its interrupt. Stop doesn't wait for it, so the
// interrupt landed after the next arm, on a call that had its own time
// left, and that call was kept as it was, reading "took too long"
// (#3742). Here the timer's func is held at that moment, so the next
// call, of the time it has of its own, succeeds.
func TestLateInterruptSparesTheNextCall(t *testing.T) {
	e := file(t, `export function onRequest(body) { body.slow = true; return body; }`)
	use(t, e)
	synctest.Test(t, func(t *testing.T) {
		// the timer's func is held while the call it was set for ends
		begun, release := make(chan struct{}), make(chan struct{})
		afterFunc = func(d time.Duration, f func()) *time.Timer {
			return time.AfterFunc(d, func() {
				close(begun)
				<-release
				f()
			})
		}
		t.Cleanup(func() { afterFunc = time.AfterFunc })
		was := requestLimit
		t.Cleanup(func() { requestLimit = was })
		requestLimit = 50 * time.Millisecond

		// one middleware, one runtime, as a request's hooks share
		s := current()
		m := s.list[0]
		in, err := m.get()
		if err != nil {
			t.Fatal(err)
		}
		defer m.put(in)

		// the call whose time runs out as it ends: the timer's func
		// begins as it ends and is held there
		got := make(chan error, 1)
		go func() {
			in.arm(10 * time.Millisecond)
			time.Sleep(20 * time.Millisecond)
			if in.disarm() {
				got <- errTimeout
				return
			}
			got <- nil
		}()
		<-begun
		synctest.Wait() // the first call has ended, the timer's func held
		// disarm must not have returned before the func has said
		// whether it fired: an interrupt it gave after the next arm
		// would stop a call that had its own time left
		if len(got) != 0 {
			t.Error("disarm returned while its timer's func hadn't given its interrupt")
		}
		close(release)
		synctest.Wait() // the timer's func has given its interrupt
		// the first call, whose timer ran out as the func was held,
		// reads took too long: its hook slept past its limit
		if err := <-got; err == nil {
			t.Error("the first call, that outstayed its limit, succeeded")
		}

		// the interrupt the held func gave lands between the calls:
		// the next call, of the time it has of its own, must not be
		// stopped by it
		requestLimit = was
		v, err := in.call(in.m.onReq, requestLimit, mustJSON(t, in, `{}`))
		if err != nil {
			t.Errorf("the second call: %v", err)
		} else if !strings.Contains(in.text(v), `"slow"`) {
			t.Errorf("the second call wasn't run: %s", in.text(v))
		}
	})
}

// mustJSON parses data as the call's argument.
func mustJSON(t *testing.T, in *inst, data string) moejs.Value {
	t.Helper()
	v, err := in.rt.ParseJSON([]byte(data))
	if err != nil {
		t.Fatal(err)
	}
	return v
}

// The everyday case stays: a hook that outstays its limit is stopped,
// reads "took too long" and the call after it runs.
func TestLimitStopsAHookStillRunning(t *testing.T) {
	e := file(t, `
let n = 0;
export function onRequest(body) { if (++n === 1) for (;;) {} body.n = n; return body; }`)
	use(t, e)
	was := requestLimit
	t.Cleanup(func() { requestLimit = was })
	requestLimit = 50 * time.Millisecond
	r := Begin(info())
	defer r.End()
	if body, _ := r.Request([]byte(`{}`)); strings.Contains(string(body), `"n"`) {
		t.Fatalf("the call that runs out: %s", body)
	}
	st := States()[e.Spec]
	if st.Failures != 1 || !strings.Contains(st.Last, "too long") {
		t.Fatalf("state %+v", st)
	}
	requestLimit = was
	if body, _ := r.Request([]byte(`{}`)); !strings.Contains(string(body), `"n":2`) {
		t.Fatalf("the call after: %s", body)
	}
}
