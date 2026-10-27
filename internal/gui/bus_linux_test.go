package gui

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"testing"

	"github.com/godbus/dbus/v5"
)

// trayReg is one item a watcher was handed: the name it is filed under and
// the entry it shows as.
type trayReg struct{ name, entry string }

// fakeTrayWatcher is a StatusNotifierWatcher on a private session bus, with
// the two shapes magpie meets: one that files the item under the caller's own
// name and keeps it when the well-known name goes (COSMIC, #2804), and one
// that files it under the name it was handed and drops it with that name
// (noctalia, #2421).
type fakeTrayWatcher struct {
	conn   *dbus.Conn // asks the bus which names are still owned
	mu     sync.Mutex
	byName bool // file the item under the caller's own name
	filed  []trayReg
}

func (w *fakeTrayWatcher) reset(byName bool) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.byName, w.filed = byName, nil
}

// RegisterStatusNotifierItem files a registration the way the watcher it
// models does: COSMIC keys it on the caller's own name, so a registration by
// a well-known name shows as "<sender>/StatusNotifierItem" and one by a
// unique name as that name (the two entries in #2804).
func (w *fakeTrayWatcher) RegisterStatusNotifierItem(service string, sender dbus.Sender) *dbus.Error {
	w.mu.Lock()
	defer w.mu.Unlock()
	reg := trayReg{name: service, entry: service + "/StatusNotifierItem"}
	if w.byName {
		reg.name = string(sender)
		if strings.HasPrefix(service, ":") {
			reg.entry = service
		} else {
			reg.entry = string(sender) + "/StatusNotifierItem"
		}
	}
	w.filed = append(w.filed, reg)
	return nil
}

// Get answers RegisteredStatusNotifierItems. A watcher that filed an item
// under a name the bus no longer owns dropped it with the name.
func (w *fakeTrayWatcher) Get(iface, property string) (dbus.Variant, *dbus.Error) {
	if iface != "org.kde.StatusNotifierWatcher" || property != "RegisteredStatusNotifierItems" {
		return dbus.Variant{}, dbus.NewError("org.freedesktop.DBus.Error.UnknownProperty", nil)
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	items := []string{}
	for _, r := range w.filed {
		if strings.HasPrefix(r.name, ":") || w.owned(r.name) {
			items = append(items, r.entry)
		}
	}
	return dbus.MakeVariant(items), nil
}

func (w *fakeTrayWatcher) owned(name string) bool {
	var owner string
	return w.conn.BusObject().Call("org.freedesktop.DBus.GetNameOwner", 0, name).Store(&owner) == nil
}

// newTrayWatcherBus starts a session bus of the test's own and owns
// org.kde.StatusNotifierWatcher on it.
func newTrayWatcherBus(t *testing.T) *fakeTrayWatcher {
	t.Helper()
	bin, err := exec.LookPath("dbus-daemon")
	if err != nil {
		t.Skip("dbus-daemon is required for a session bus of the test's own")
	}
	cmd := exec.Command(bin, "--session", "--nofork", "--print-address=1")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { cmd.Process.Kill(); cmd.Wait() })
	sc := bufio.NewScanner(stdout)
	if !sc.Scan() {
		t.Fatalf("no private bus address: %v", sc.Err())
	}
	t.Setenv("DBUS_SESSION_BUS_ADDRESS", sc.Text())

	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	w := &fakeTrayWatcher{conn: conn}
	if err := conn.Export(w, "/StatusNotifierWatcher", "org.kde.StatusNotifierWatcher"); err != nil {
		t.Fatal(err)
	}
	if err := conn.Export(w, "/StatusNotifierWatcher", "org.freedesktop.DBus.Properties"); err != nil {
		t.Fatal(err)
	}
	if reply, err := conn.RequestName("org.kde.StatusNotifierWatcher", dbus.NameFlagDoNotQueue); err != nil || reply != dbus.RequestNameReplyPrimaryOwner {
		t.Fatalf("claim the watcher name: %v %v", reply, err)
	}
	return w
}

// dropTrayName gives up the name Wails' tray took and registers the item
// again under the connection's own name. A watcher that already lists the
// item under that name keeps it through the release: registering again is a
// second magpie in the panel (#2804), while a watcher that dropped it with
// the name still needs the registration (#2421).
func TestTrayReleasedNameKeepsOneItem(t *testing.T) {
	w := newTrayWatcherBus(t)
	conn, err := dbus.SessionBus()
	if err != nil {
		t.Fatal(err)
	}
	trayName := fmt.Sprintf("org.kde.StatusNotifierItem-%d-1", os.Getpid())
	obj := conn.Object("org.kde.StatusNotifierWatcher", "/StatusNotifierWatcher")
	for _, c := range []struct {
		name   string
		byName bool
	}{
		{"a watcher that keeps the item under our own name", true},
		{"a watcher that drops it with the name", false},
	} {
		t.Run(c.name, func(t *testing.T) {
			w.reset(c.byName)
			if reply, err := conn.RequestName(trayName, 0); err != nil || reply != dbus.RequestNameReplyPrimaryOwner {
				t.Fatalf("take the tray name: %v %v", reply, err)
			}
			// the tray registers as Wails' does, by the name it took
			if call := obj.Call("org.kde.StatusNotifierWatcher.RegisterStatusNotifierItem", 0, trayName); call.Err != nil {
				t.Fatal("register:", call.Err)
			}
			dropTrayName()
			v, err := obj.GetProperty("org.kde.StatusNotifierWatcher.RegisteredStatusNotifierItems")
			if err != nil {
				t.Fatal("read items:", err)
			}
			var items []string
			if err := v.Store(&items); err != nil {
				t.Fatal("read items:", err)
			}
			if len(items) != 1 {
				t.Fatalf("the panel lists %d magpie items %q after the name was released, want 1", len(items), items)
			}
		})
	}
}
