package maskrec

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// What an answer put back is kept with its record, in the record's day,
// not only in the memory of the magpie that sent the request: a window
// that doesn't serve the gateway, and one after a restart, show the count
// (#3540). Its line counts no request in the summary, and an answer that
// comes back after midnight is still read with its record.
func TestPutBackKeptWithItsRecord(t *testing.T) {
	now := time.Date(2026, 10, 20, 23, 59, 30, 0, time.Local)
	dir := sandbox(t, now)
	p := "{{API_KEY_ccccdddd}}"
	r := Add(Record{Time: now, What: "chat", N: 2, Values: []Value{{Placeholder: p, Kind: "API_KEY", By: "rule", Len: 40, N: 2}}})
	// the answer comes back after midnight, and the put-back is told
	clock = func() time.Time { return now.Add(time.Minute) }
	Restored(r.Key, map[string]int{p: 2})
	// memory is gone: a restart, or a window that didn't serve the gateway
	store.mu.Lock()
	store.snips, store.days = nil, nil
	store.mu.Unlock()
	if got, ok := Get(r.Key); !ok || got.Back[p] != 2 {
		t.Fatalf("after a restart: %+v %v", got, ok)
	}
	rs, _ := List(0, 10, Filter{})
	if len(rs) != 1 || rs[0].Key != r.Key || rs[0].Back[p] != 2 {
		t.Fatalf("listed: %+v", rs)
	}
	if _, err := os.Stat(filepath.Join(dir, "2026-10-20.jsonl")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "2026-10-21.jsonl")); err == nil {
		t.Fatal("the put-back went into the next day")
	}
	if s := Sum(3); s.Requests != 1 || s.N != 2 {
		t.Fatalf("a put-back counted as a request: %+v", s)
	}
}
