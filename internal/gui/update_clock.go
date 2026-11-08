package gui

import (
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/yetone/magpie/internal/settings"
	"github.com/yetone/magpie/internal/steady"
)

func updateClockFile() string { return filepath.Join(settings.Dir(), "update-checked") }

func readUpdateClock(now time.Time) time.Time {
	b, err := os.ReadFile(updateClockFile())
	if err != nil {
		if !os.IsNotExist(err) {
			log.Printf("update: read check time: %v", err)
		}
		return time.Time{}
	}
	t, err := time.Parse(time.RFC3339Nano, string(b))
	if err != nil {
		log.Printf("update: parse check time: %v", err)
		return time.Time{}
	}
	if t.After(now) {
		return time.Time{}
	}
	return t
}

func writeUpdateClock(t time.Time) {
	if err := os.MkdirAll(settings.Dir(), 0o700); err != nil {
		log.Printf("update: save check time: %v", err)
		return
	}
	if err := steady.WriteFile(updateClockFile(), []byte(t.UTC().Format(time.RFC3339Nano)), 0o600); err != nil {
		log.Printf("update: save check time: %v", err)
	}
}
