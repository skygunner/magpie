package agent

import "testing"

func TestLocalRewireInventory(t *testing.T) {
	home := syncHome(t)
	for _, a := range onMachine(home, home+"/.config") {
		if len(a.Fields) == 0 {
			continue
		}
		if a.joinRewire == nil && !a.reapplyRewire && (a.Native == nil || a.Native.connectRewire == nil) {
			t.Errorf("%s still uses the global rewire fallback", a.ID)
		}
	}
}
