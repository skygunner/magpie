package gateway

import (
	"crypto/tls"
	"net/http"
	"time"

	"github.com/yetone/magpie/internal/netproxy"
)

// upstreamTLS is what the gateway's transports dial TLS with: nil, the
// system's roots, but for a test trusting its own server.
var upstreamTLS *tls.Config

// upstreamTransport is the gateway's transport to the vendors: a provider
// with a proxy of its own is sent through a transport kept for that proxy
// (#237), and the ChatGPT backend over HTTP/1.1 (netproxy.HTTP1), a
// streamed turn to it over its WebSocket (codexSocket). Every
// other vendor is spoken HTTP/2 to wherever it offers it. A request that
// finds its kept-alive connection dropped is sent again on a new one
// (netproxy.Dispatch). A vendor is given upstreamHeaderWait to begin
// answering, but for a drawing, which an images API answers only once it
// has drawn:
// it goes through a transport of its own that waits as long as Settings'
// Image generation timeout gives it (drawWithin), up to 60 minutes.
func upstreamTransport() http.RoundTripper {
	return drawsApart{
		any:   codexSocket{netproxy.Dispatch(vendorTransport(upstreamHeaderWait))},
		draws: netproxy.Dispatch(vendorTransport(0)),
	}
}

// upstreamHeaderWait is how long a vendor is given to begin answering a
// request that isn't a drawing.
var upstreamHeaderWait = 10 * time.Minute

// vendorTransport is a transport to the vendors that gives one wait to
// begin answering, none for 0: the request's context then bounds it.
func vendorTransport(wait time.Duration) *http.Transport {
	return &http.Transport{
		Proxy:                 netproxy.Func,
		ResponseHeaderTimeout: wait,
		MaxIdleConnsPerHost:   8,
		IdleConnTimeout:       90 * time.Second,
		ForceAttemptHTTP2:     true,
		TLSClientConfig:       upstreamTLS,
		// a silent HTTP/2 connection is pinged and dropped (#2843)
		HTTP2: netproxy.HTTP2Health(),
	}
}

// drawsApart sends a drawing's request (drawWaitOf) through draws, every
// other through any.
type drawsApart struct{ any, draws http.RoundTripper }

func (t drawsApart) RoundTrip(r *http.Request) (*http.Response, error) {
	if _, ok := drawWaitOf(r.Context()); ok {
		return t.draws.RoundTrip(r)
	}
	return t.any.RoundTrip(r)
}
