package cli

import (
	"io"
	"os"
	"strings"
	"testing"

	"github.com/yetone/magpie/internal/gui"
)

// captureStdout is what f printed, f being one of the printers that writes
// os.Stdout directly.
func captureStdout(t *testing.T, f func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	old := os.Stdout
	os.Stdout = w
	var out strings.Builder
	read := make(chan struct{})
	go func() {
		b, _ := io.ReadAll(r)
		out.Write(b)
		close(read)
	}()
	f()
	w.Close()
	os.Stdout = old
	<-read
	return out.String()
}

// A page bound to one address (`magpie serve --web=100.x.y.z:3430`,
// `magpie web --addr=` the same) answers there alone: the status and the
// serve log list that address, and none of the computer's others, where
// nothing listens (#4843). Bound to every interface, the links to them all
// are listed as before.
func TestWebPrintListsTheBindAlone(t *testing.T) {
	t.Setenv("MAGPIE_PUBLIC_URL", "https://magpie.example.com")
	const key = "0123456789abcdef"
	const addr = "100.100.100.100:3430"
	link := "http://" + addr + "/?k=" + key
	for _, c := range []struct {
		name string
		out  string
	}{
		{"daemon status", captureStdout(t, func() { printDaemonWeb(addr, key) })},
		{"magpie web", captureStdout(t, func() {
			printWeb(&gui.Web{Addr: addr, Link: link})
		})},
	} {
		if !strings.Contains(c.out, link) {
			t.Errorf("%s: the bound address's link: %q", c.name, c.out)
		}
		if strings.Contains(c.out, "on the network") {
			t.Errorf("%s: a page bound to one address listed links where nothing listens: %q", c.name, c.out)
		}
	}
	if out := captureStdout(t, func() { printDaemonWeb("0.0.0.0:3430", key) }); !strings.Contains(out, "on the network") {
		t.Errorf("a page on every interface listed no network links: %q", out)
	}
	if out := captureStdout(t, func() { printWeb(&gui.Web{Addr: "0.0.0.0:3430", Link: "http://127.0.0.1:3430/?k=" + key}) }); !strings.Contains(out, "on the network") {
		t.Errorf("a page on every interface listed no network links: %q", out)
	}
}
