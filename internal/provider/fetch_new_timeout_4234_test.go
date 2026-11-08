package provider

import (
	"context"
	"errors"
	"net/http"
	"path/filepath"
	"testing"
	"testing/synctest"
	"time"
)

type deadlineTransport struct {
	result chan error
}

func (d deadlineTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	select {
	case <-r.Context().Done():
		d.result <- r.Context().Err()
		return nil, r.Context().Err()
	case <-time.After(2 * time.Second):
		err := errors.New("request context did not expire")
		d.result <- err
		return nil, err
	}
}

func TestFetchNewStopsAtDeadline(t *testing.T) {
	isolate(t)
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	t.Setenv("XDG_CACHE_HOME", filepath.Join(home, ".cache"))
	newFetches.Lock()
	newFetches.m = map[string]time.Time{}
	newFetches.Unlock()
	codexListedAt(t, "http://codex.test")
	codexSignedIn(t, home)
	result := make(chan error, 1)
	old := http.DefaultClient.Transport
	http.DefaultClient.Transport = deadlineTransport{result: result}
	t.Cleanup(func() { http.DefaultClient.Transport = old })
	synctest.Test(t, func(t *testing.T) {
		if !FetchNew(time.Second) {
			t.Fatal("timed-out vendor list was not reported missing")
		}
		select {
		case err := <-result:
			if !errors.Is(err, context.DeadlineExceeded) {
				t.Fatalf("request error = %v, want deadline exceeded", err)
			}
		default:
			t.Fatal("vendor request did not reach transport")
		}
	})
}
