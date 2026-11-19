package gui

import (
	"strings"

	"github.com/yetone/magpie/internal/settings"
)

// The main window opens as it was last left: at the size it was last given
// (settings.Window), where it was last put (settings.WindowAt) and maximised
// again if it was (settings.WindowMaximised).

// windowW, windowH is the main window's size, in points, before it is first
// given one.
const windowW, windowH = 660, 600

// openSize is the size, in points, the main window opens at: the one it was
// last given (saved), else windowW×windowH; no smaller than its least (minW,
// minH).
func openSize(saved []int, minW, minH int) (int, int) {
	w, h := windowW, windowH
	if len(saved) == 2 && saved[0] >= windowMinW && saved[1] >= windowMinH {
		w, h = saved[0], saved[1]
	}
	return max(w, minW), max(h, minH)
}

// fitRoom is w×h no larger than the work area of the screen it is on (sw,
// sh; 0 when unknown), and whether that changed it: a size kept on a larger
// screen, or before the display was scaled up, still fits, title bar in
// sight.
func fitRoom(w, h, sw, sh int) (int, int, bool) {
	fw, fh := w, h
	if sw > 0 {
		fw = min(fw, sw)
	}
	if sh > 0 {
		fh = min(fh, sh)
	}
	return fw, fh, fw != w || fh != h
}

// settle keeps in s the main window as it has settled, w×h and maximised or
// not, on a screen whose work area is sw×sh (0s when unknown), and says
// whether s changed. A maximised window is the screen's size, not one the
// user gave it: the size it is restored to stays as it was. So does one
// under the least, or within 2 points of the size kept — macOS opens a
// window a point short of the size it was given, and kept as it is the
// window would shrink a point at every start. A side as long as the screen's
// keeps a longer one kept: the window was fitted to a smaller screen
// (fitRoom), and opens at its own size again on the larger one.
func settle(s *settings.Settings, w, h, sw, sh int, maximised bool) bool {
	changed := s.WindowMaximised != maximised
	s.WindowMaximised = maximised
	if maximised || w < windowMinW || h < windowMinH {
		return changed
	}
	if len(s.Window) == 2 {
		w, h = fitted(s.Window[0], w, sw), fitted(s.Window[1], h, sh)
		if abs(s.Window[0]-w) <= 2 && abs(s.Window[1]-h) <= 2 {
			return changed
		}
	}
	s.Window = []int{w, h}
	return true
}

// fitted is the side kept for one now as long as the screen's, room: the
// longer one kept before, if it was.
func fitted(kept, now, room int) int {
	if room > 0 && kept > now && abs(now-room) <= 2 {
		return kept
	}
	return now
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

// area is a screen's work area, in the units and coordinates of the main
// window's Position: x, y of its top left corner, width and height.
type area struct{ x, y, w, h int }

// desk is a screen the main window can be left on: the platform's id for it
// (Wails' Screen.ID) and its work area.
type desk struct {
	id string
	at area
}

// openAt is where the main window, w×h, opens: at the place it was last
// left (saved: x, y), moved as little as it takes to lie whole on the work
// area of the screen holding most of it now (areas). Not ok — left where the
// system puts it, the middle of the screen — when no place was kept, or
// no screen there now holds any of it: the screen it was on is gone, the
// screens were laid out again, or none is known. A window larger than the
// area keeps its top left corner in it, title bar in reach.
func openAt(saved []int, w, h int, desks []desk) (x, y int, ok bool) {
	if len(saved) != 2 || w <= 0 || h <= 0 {
		return 0, 0, false
	}
	x, y = saved[0], saved[1]
	best, most := desk{}, 0
	for _, d := range desks {
		if n := overlap(x, w, d.at.x, d.at.w) * overlap(y, h, d.at.y, d.at.h); n > most {
			best, most = d, n
		}
	}
	if most == 0 {
		return 0, 0, false
	}
	return within(x, w, best.at.x, best.at.w), within(y, h, best.at.y, best.at.h), true
}

// screenFor is the id of the screen whose work area holds the point x, y,
// "" when none does: the screen the main window is on as it settles.
func screenFor(desks []desk, x, y int) string {
	for _, d := range desks {
		if holds(x, y, d.at) {
			return d.id
		}
	}
	return ""
}

// holds is whether a screen's work area holds the point x, y.
func holds(x, y int, a area) bool {
	return x >= a.x && x < a.x+a.w && y >= a.y && y < a.y+a.h
}

// putWhere is where a window w×h, last at x, y, opens to sit on the screen
// named on: clamped into that screen's work area, as openAt clamps. Not ok
// when the name names no screen there now (the screen gone, or an id the
// platform does not carry across restarts), and the caller leaves the
// window where it was put.
func putWhere(on string, desks []desk, x, y, w, h int) (int, int, bool) {
	for _, d := range desks {
		if d.id == on {
			return within(x, w, d.at.x, d.at.w), within(y, h, d.at.y, d.at.h), true
		}
	}
	return 0, 0, false
}

// overlap is how much of the span from p, n long, lies in the one from
// ap, an long.
func overlap(p, n, ap, an int) int {
	return max(0, min(p+n, ap+an)-max(p, ap))
}

// within is p moved as little as it takes for the span from it, n long, to
// lie in the one from ap, an long; ap when it is the longer.
func within(p, n, ap, an int) int {
	return max(ap, min(p, ap+an-n))
}

// settleAt keeps in s the place the main window was left at, x, y, and
// says whether s changed.
func settleAt(s *settings.Settings, x, y int) bool {
	if len(s.WindowAt) == 2 && s.WindowAt[0] == x && s.WindowAt[1] == y {
		return false
	}
	s.WindowAt = []int{x, y}
	return true
}

// settleScreen keeps in s the id of the screen the main window was left on,
// and says whether s changed. An empty id — no screen holds it — keeps what
// was kept: no screen is no news.
func settleScreen(s *settings.Settings, on string) bool {
	if on == "" || s.WindowScreen == on {
		return false
	}
	s.WindowScreen = on
	return true
}

// keepsPlace says whether the main window's place can be read and set on
// goos: not in a Wayland session (unless GTK is made to speak X11 there,
// GDK_BACKEND=x11), where a window has no place it can know and the
// compositor puts it where it likes; Wails reads 0, 0 there.
func keepsPlace(goos string, getenv func(string) string) bool {
	return !waylandSession(goos, getenv) || strings.HasPrefix(getenv("GDK_BACKEND"), "x11")
}
