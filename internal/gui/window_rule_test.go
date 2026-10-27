package gui

import (
	"slices"
	"testing"

	"github.com/yetone/magpie/internal/settings"
)

// The window opens at the size it was last given, the default before
// that, and never smaller than its least.
func TestOpenSize(t *testing.T) {
	for _, c := range []struct {
		name       string
		saved      []int
		minW, minH int
		w, h       int
	}{
		{"default", nil, 560, 420, windowW, windowH},
		{"kept", []int{1713, 1084}, 560, 420, 1713, 1084},
		{"under the least", []int{300, 200}, 560, 420, windowW, windowH},
		{"malformed", []int{900}, 560, 420, windowW, windowH},
		{"least at a larger text size", []int{600, 450}, 840, 630, 840, 630},
	} {
		if w, h := openSize(c.saved, c.minW, c.minH); w != c.w || h != c.h {
			t.Errorf("%s: openSize = %d×%d, want %d×%d", c.name, w, h, c.w, c.h)
		}
	}
}

// A window larger than its screen's work area is fitted to it, so its
// title bar is in sight; one that fits, or on a screen not known, is left.
func TestFitRoom(t *testing.T) {
	for _, c := range []struct {
		name         string
		w, h, sw, sh int
		fw, fh       int
		changed      bool
	}{
		{"fits", 1200, 800, 1920, 1040, 1200, 800, false},
		{"kept on a larger screen", 2400, 1400, 1920, 1040, 1920, 1040, true},
		{"too tall only", 1200, 1300, 1920, 1040, 1200, 1040, true},
		{"screen unknown", 2400, 1400, 0, 0, 2400, 1400, false},
	} {
		if fw, fh, changed := fitRoom(c.w, c.h, c.sw, c.sh); fw != c.fw || fh != c.fh || changed != c.changed {
			t.Errorf("%s: fitRoom = %d×%d %v, want %d×%d %v", c.name, fw, fh, changed, c.fw, c.fh, c.changed)
		}
	}
}

// The window opens where it was last left (#2785), not centred: the place it
// is moved to is kept as it settles, and used again. One left on a screen
// since unplugged opens centred, where its title bar is in reach, rather than
// off every screen; screens not looked at yet are unknown, not gone. A file
// from an older magpie, or a place short of x and y, is no place.
func TestMainWindowKeepsItsPlace(t *testing.T) {
	screens := [][4]int{{0, 0, 1728, 1084}, {-1280, 0, 1280, 1024}}
	var s settings.Settings
	if !settleAt(&s, -1280, 0) || !slices.Equal(s.WindowAt, []int{-1280, 0}) {
		t.Fatalf("settleAt kept %v", s.WindowAt)
	}
	if x, y, ok := openAt(s.WindowAt, screens); !ok || x != -1280 || y != 0 {
		t.Errorf("openAt = %d, %d, %v; want -1280, 0, true", x, y, ok)
	}
	if settleAt(&s, -1280, 0) {
		t.Errorf("settleAt changed a place it had kept: %v", s.WindowAt)
	}
	if !settleAt(&s, 100, 50) {
		t.Errorf("settleAt kept nothing of a place moved to: %v", s.WindowAt)
	}
	// that second screen unplugged
	if x, y, ok := openAt(s.WindowAt, screens[1:]); ok {
		t.Errorf("openAt = %d, %d, %v on a screen that has gone; want centred", x, y, ok)
	}
	// just past its right edge, and its last point
	if _, _, ok := openAt([]int{0, 0}, screens[1:]); ok {
		t.Error("openAt kept a place past the screen's right edge")
	}
	if _, _, ok := openAt([]int{-1, 1023}, screens[1:]); !ok {
		t.Error("openAt lost a place on the screen's last point")
	}
	if x, y, ok := openAt(s.WindowAt, nil); !ok || x != 100 || y != 50 {
		t.Errorf("openAt with no screens looked at = %d, %d, %v; want the place kept", x, y, ok)
	}
	for _, saved := range [][]int{nil, {5}} {
		if _, _, ok := openAt(saved, screens); ok {
			t.Errorf("openAt(%v) = a place, want centred", saved)
		}
	}
}

// The window is kept as it settles: maximised or not, and the size it is
// restored to. Maximised, the size kept stays the one the user gave it, so
// it opens maximised and restores to that size (on a MacBook: zoomed last,
// it opened at an older size a few points short of the screen). Fitted to a
// smaller screen, it keeps the larger size for the larger screen.
func TestSettle(t *testing.T) {
	type step struct {
		w, h, sw, sh int
		maximised    bool
		changed      bool
		window       []int
		max          bool
	}
	for _, c := range []struct {
		name  string
		steps []step
	}{
		{"resized", []step{{900, 700, 0, 0, false, true, []int{900, 700}, false}}},
		{"maximised keeps its size", []step{
			{900, 700, 0, 0, false, true, []int{900, 700}, false},
			{1728, 1084, 0, 0, true, true, []int{900, 700}, true},
		}},
		{"restored", []step{
			{900, 700, 0, 0, false, true, []int{900, 700}, false},
			{1728, 1084, 0, 0, true, true, []int{900, 700}, true},
			{899, 699, 0, 0, false, true, []int{900, 700}, false},
		}},
		{"maximised again, nothing new", []step{
			{1728, 1084, 0, 0, true, true, nil, true},
			{1728, 1084, 0, 0, true, false, nil, true},
		}},
		{"a point short", []step{
			{900, 700, 0, 0, false, true, []int{900, 700}, false},
			{899, 699, 0, 0, false, false, []int{900, 700}, false},
		}},
		{"under the least", []step{{300, 200, 0, 0, false, false, nil, false}}},
		{"fitted to a smaller screen", []step{
			{2400, 1400, 0, 0, false, true, []int{2400, 1400}, false},
			{1920, 1040, 1920, 1040, false, false, []int{2400, 1400}, false},
		}},
		{"fitted, then made narrower", []step{
			{2400, 1400, 0, 0, false, true, []int{2400, 1400}, false},
			{1500, 1040, 1920, 1040, false, true, []int{1500, 1400}, false},
		}},
		{"as large as the screen, nothing larger kept", []step{
			{1920, 1040, 1920, 1040, false, true, []int{1920, 1040}, false},
		}},
	} {
		t.Run(c.name, func(t *testing.T) {
			var s settings.Settings
			for i, st := range c.steps {
				changed := settle(&s, st.w, st.h, st.sw, st.sh, st.maximised)
				if changed != st.changed || !slices.Equal(s.Window, st.window) || s.WindowMaximised != st.max {
					t.Fatalf("step %d: changed %v, window %v, maximised %v; want %v, %v, %v",
						i, changed, s.Window, s.WindowMaximised, st.changed, st.window, st.max)
				}
			}
		})
	}
}
