package davsync

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

// A successful write without If-Match can have overwritten another writer.
func TestDAVPutReportsBlindAndUsesOnlySafePutVersion(t *testing.T) {
	for _, tc := range []struct {
		name, match, lastModified, date, responseETag string
		blind                                         bool
		modified                                      string
	}{
		{"unconditional", "", "", "", "", true, ""},
		{"conditional", `"old"`, "", "", `"new"`, false, ""},
		{"old modification date", "", "Wed, 01 Jan 2020 00:00:00 GMT", "Wed, 01 Jan 2020 00:00:02 GMT", "", true, "Wed, 01 Jan 2020 00:00:00 GMT"},
		{"same second", "", "Wed, 01 Jan 2020 00:00:00 GMT", "Wed, 01 Jan 2020 00:00:00 GMT", "", true, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodHead {
					w.Header().Set("Content-Length", "3")
					return
				}
				if got := r.Header.Get("If-Match"); got != tc.match {
					t.Errorf("If-Match = %q, want %q", got, tc.match)
				}
				w.Header().Set("Date", tc.date)
				w.Header().Set("Last-Modified", tc.lastModified)
				w.Header().Set("ETag", tc.responseETag)
				w.WriteHeader(http.StatusCreated)
			}))
			defer srv.Close()
			d, err := newDAV(Config{URL: srv.URL})
			if err != nil {
				t.Fatal(err)
			}
			v, err := d.put(context.Background(), []byte("abc"), tc.match)
			if err != nil {
				t.Fatal(err)
			}
			if v.Blind != tc.blind || v.Modified != tc.modified || v.ETag != tc.responseETag {
				t.Fatalf("version = %+v, want blind=%v modified=%q etag=%q", v, tc.blind, tc.modified, tc.responseETag)
			}
		})
	}
}
