package gui

import "github.com/yetone/magpie/internal/settings"

// The main window opens as it was last left: at the size it was last given
// (settings.Window), where it was last left (settings.WindowAt) and on the
// screen it was on (settings.WindowScreen), and maximised again if it was
// (settings.WindowMaximised).

// windowW, windowH is the main window's size, in points, before it is first
// given one.
const windowW, windowH = 660, 600

// desk is a screen the main window can be left on: its id as the platform
// names it (Wails' Screen.ID) and its work area as x, y, width and height, in
// the units of the window's Position.
type desk struct {
	id   string
	room [4]int
}

// openAt is the place the main window opens at: the one it was last left at
// (saved), on the screen it was on (on) when that screen is still attached
// and its work area still holds the place. Else the place, on any screen that
// still holds it (the displays rearranged, or that one renamed); else no
// place, and the window opens centred, where at least its title bar is in
// reach. Screens not looked at yet are unknown, not gone, and the place saved
// is used. at is the screen it opens on, 0 with no screens looked at.
func openAt(saved []int, on string, screens []desk) (at desk, x, y int, ok bool) {
	if len(saved) != 2 {
		return desk{}, 0, 0, false
	}
	if len(screens) == 0 {
		return desk{}, saved[0], saved[1], true
	}
	for _, d := range screens {
		if on != "" && d.id == on && holds(d.room, saved[0], saved[1]) {
			return d, saved[0], saved[1], true
		}
	}
	for _, d := range screens {
		if holds(d.room, saved[0], saved[1]) {
			return d, saved[0], saved[1], true
		}
	}
	return desk{}, 0, 0, false
}

// holds is whether a screen's work area, x, y, width and height, holds the
// point.
func holds(room [4]int, x, y int) bool {
	return x >= room[0] && x < room[0]+room[2] && y >= room[1] && y < room[1]+room[3]
}

// deskOf is the screen holding x, y, if any: the one the main window is on as
// it settles. "" when none does.
func deskOf(screens []desk, x, y int) string {
	for _, d := range screens {
		if holds(d.room, x, y) {
			return d.id
		}
	}
	return ""
}

// settleAt keeps in s the main window's place, x and y, and the id of the
// screen it is on (on, "" when no screen holds it), as it settled, and says
// whether s changed. The caller leaves a maximised window out: its place is
// the screen's, not one the user gave it, as its size is (settle).
func settleAt(s *settings.Settings, x, y int, on string) bool {
	changed := false
	if len(s.WindowAt) != 2 || s.WindowAt[0] != x || s.WindowAt[1] != y {
		s.WindowAt = []int{x, y}
		changed = true
	}
	if on != "" && s.WindowScreen != on {
		s.WindowScreen = on
		changed = true
	}
	return changed
}

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
