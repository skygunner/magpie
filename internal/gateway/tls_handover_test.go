package gateway

import (
	"bufio"
	"io"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"
)

// A handover with TLS on still answers a connection from this computer
// the TLS listener took, whose first byte hadn't come yet when it began:
// it reaches the server while the handed-over gateway drains, instead of
// being cut by the listener's end (#3647).
func TestHandedOffTLSGatewayAnswersAConnectionWaitingForItsFirstByte(t *testing.T) {
	fresh(t)
	cert, key := tlsFiles(t)
	writeCert(t, cert, key, "relay.example")
	t.Setenv("MAGPIE_ADDR", "127.0.0.1:0")
	old := New()
	stopOld, oldDone := listening(t, old)
	old.lnMu.Lock()
	addr := old.ln.Addr().String()
	old.lnMu.Unlock()

	// given: a connection from this computer the TLS listener took, silent
	silent, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}
	defer silent.Close()
	time.Sleep(50 * time.Millisecond) // it is accepted, waiting for its first byte

	// when: the gateway hands over, and only then the connection speaks
	old.HandOff()
	stopOld()
	req, _ := http.NewRequest("GET", "http://"+addr+"/v1/models", nil)
	if err := req.Write(silent); err != nil {
		t.Fatalf("the connection was cut at the handover: %v", err)
	}
	silent.SetReadDeadline(time.Now().Add(5 * time.Second))

	// then: the old gateway answers it before it quits
	res, err := http.ReadResponse(bufio.NewReader(silent), req)
	if err != nil {
		t.Fatalf("no answer to a request sent once the handover began: %v", err)
	}
	b, _ := io.ReadAll(res.Body)
	res.Body.Close()
	if res.StatusCode != http.StatusOK || !strings.Contains(string(b), `"data"`) {
		t.Fatalf("the silent connection's request: %d %s", res.StatusCode, b)
	}
	select {
	case err := <-oldDone:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("the old gateway never ended")
	}
}
