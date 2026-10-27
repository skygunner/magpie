package gui

import (
	"testing"
	"time"
)

// TestOnMainWaitsForTheMainThread holds the guarantee the autostart panic
// broke (#3108): an updater callback that runs on the app's main thread
// runs nothing until that thread is up, and then runs there — InvokeSync
// on an app whose main thread isn't up panics on the nil impl.
func TestOnMainWaitsForTheMainThread(t *testing.T) {
	oldUp, oldInvoke := mainUp, invokeSync
	t.Cleanup(func() { mainUp, invokeSync = oldUp, oldInvoke })
	up := make(chan struct{})
	mainUp = up
	ran := make(chan struct{})
	invokeSync = func(fn func()) { fn(); close(ran) }

	go onMain(func() {})
	select {
	case <-ran:
		t.Fatal("onMain ran fn before the main thread was up")
	case <-time.After(50 * time.Millisecond):
	}

	close(up)
	select {
	case <-ran:
	case <-time.After(2 * time.Second):
		t.Fatal("onMain did not run fn once the main thread was up")
	}
}
