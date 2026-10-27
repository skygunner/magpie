package netproxy

import "net/http"

// replayable is req marked so Go's transport sends it again at once, on a
// new connection, when the kept-alive connection it went out on turns out
// to be dropped: written into, and closed before a byte of a reply ("EOF",
// "write: broken pipe"). Go does so by itself only for a request it may
// replay — a GET, or one marked idempotent — and only on a reused
// connection, so a POST to a vendor failed as a 502 and was asked again
// after a pause, each pause on a connection a proxy or the vendor's front
// had dropped while idle (#2136: Pi's Codex turn through Clash), and an
// editor's System One detection could not be answered at all for a vendor
// that answered curl (#3665: "Post …/systemone: EOF"). The mark is an
// X-Idempotency-Key with no value, which Go reads and never sends: the
// vendor sees the request as before.
func replayable(req *http.Request) *http.Request {
	_, marked := req.Header["Idempotency-Key"]
	if _, ok := req.Header["X-Idempotency-Key"]; ok {
		marked = true
	}
	if marked || req.Method == http.MethodGet || req.Method == http.MethodHead ||
		(req.Body != nil && req.Body != http.NoBody && req.GetBody == nil) {
		return req
	}
	// a copy, as a RoundTripper leaves its caller's request as it was
	r := req.Clone(req.Context())
	r.Header["X-Idempotency-Key"] = nil
	return r
}
