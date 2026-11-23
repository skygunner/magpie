//go:build !nogui

package gui

import (
	"testing"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// The first show on the Mac can come before Wails' screen cache, and a
// place kept there must still be restored (#2785, pluo: a corner became the
// centre after every update restart).
func TestOpenAtBeforeScreenCache(t *testing.T) {
	native := func() []area { return []area{{x: 0, y: 44, w: 3072, h: 1650}} }
	for _, c := range []struct {
		name    string
		screens []*application.Screen
	}{
		{"no cache", nil},
		{"nothing usable in the cache", []*application.Screen{nil, {}}},
	} {
		t.Run(c.name, func(t *testing.T) {
			areas := workAreas(c.screens, native)
			x, y, ok := openAt([]int{2113, 44}, 942, 904, areas)
			if x != 2113 || y != 44 || !ok {
				t.Errorf("openAt = %d, %d, %v; want 2113, 44, true", x, y, ok)
			}
			// A place no display holds is still left centred.
			if _, _, ok := openAt([]int{-2500, 44}, 942, 904, areas); ok {
				t.Error("restored a place no display holds")
			}
		})
	}

	cached := func() []area {
		t.Fatal("read the native work areas with a cache to read")
		return nil
	}
	screens := []*application.Screen{{WorkArea: application.Rect{X: -1920, Y: 32, Width: 1920, Height: 1040}}}
	if x, y, ok := openAt([]int{-1500, 100}, 942, 904, workAreas(screens, cached)); x != -1500 || y != 100 || !ok {
		t.Errorf("from the cache: openAt = %d, %d, %v; want -1500, 100, true", x, y, ok)
	}
}
