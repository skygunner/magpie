package agent

import (
	"encoding/json"
	"errors"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/yetone/magpie/internal/edit"
	"github.com/yetone/magpie/internal/filelock"
	"github.com/yetone/magpie/internal/provider"
)

// The stash remembers what an agent's config said before magpie pointed it
// at the gateway, so switching back restores it instead of guessing.
//
// stash.json holds every agent's originals, so a change to it is read,
// changed and written under stashMu, and written whole (edit.WriteAtomic):
// two changes at once, or one cut short, never leave it half there, which
// the next change read as no stash and wrote back over every agent's. A
// file that doesn't parse is left untouched; a change reports the error.
//
// The gateway and a magpie run from a terminal share the config folder, so
// the change is one at a time across them too: each takes the lock on
// stash.json.lock while it reads, changes and writes. A change that can't
// be made — another magpie holding the lock past the wait, or a folder
// that can't be written — says so instead of reporting a value kept: the
// caller goes on to write magpie's own value over the user's, and a stash
// that wasn't written has nothing to put back.
var stashMu sync.Mutex

// During rewire, writes that would change saved originals fail closed.
var rewireStash *bool
var rewireWiredAt map[string]string

type rewireGuard struct {
	blocked bool
	wiredAt map[string]string
}

// stashWait is how long a change waits for another magpie to let go of the
// stash; a test shortens it.
var stashWait = filelock.Wait

// errStashLock says a change went unwritten because another magpie kept the
// stash: what it saved would have been written over.
var errStashLock = errors.New("another magpie held the stash; nothing was saved")

func stashPath() string { return filepath.Join(filepath.Dir(provider.Path()), "stash.json") }

// stashLock takes the stash's lock for this magpie and the one on its lock
// file, and gives back the way to let both go. A file system that can't
// lock says so too: two magpies would otherwise write over each other's
// saved values, so the change fails rather than risk them.
func stashLock() (func(), error) {
	stashMu.Lock()
	unlock, held := filelock.Lock(stashPath()+".lock", stashWait)
	if !held {
		stashMu.Unlock()
		return nil, errStashLock
	}
	return func() { unlock(); stashMu.Unlock() }, nil
}

func stashLoad() map[string]string {
	out := map[string]string{}
	if b, err := os.ReadFile(stashPath()); err == nil {
		json.Unmarshal(b, &out)
	}
	return out
}

// stashWrite writes m as the stash. The caller holds the stash's lock.
func stashWrite(m map[string]string) error {
	return stashWriteFor(m, rewireStash)
}

// stashWriteFor applies a rewire guard only to the operation that owns it.
// The caller holds the stash's lock.
func stashWriteFor(m map[string]string, guard *bool) error {
	return stashWriteGuard(m, guard, rewireWiredAt)
}

func stashWriteGuard(m map[string]string, guard *bool, wired map[string]string) error {
	var before map[string]string
	if guard != nil {
		var err error
		before, err = stepcodeReadStash()
		if err != nil {
			*guard = true
			return err
		}
		for k, v := range before {
			next, exists := m[k]
			if strings.HasSuffix(k, wiredAt) && (!exists || next != v) && wired != nil {
				old, had := wired[k]
				if old != v || !had {
					*guard = true
					return errors.New("rewire found a concurrent wired address change")
				}
			}
			if !strings.HasSuffix(k, wiredAt) && m[k] != v {
				*guard = true
				return errors.New("rewire would change saved stash values")
			}
		}
		for k, v := range m {
			current, exists := before[k]
			if strings.HasSuffix(k, wiredAt) && (!exists || current != v) && wired != nil {
				old, had := wired[k]
				if old != current || had != exists {
					*guard = true
					return errors.New("rewire found a concurrent wired address change")
				}
			}
			if !strings.HasSuffix(k, wiredAt) && before[k] != v {
				*guard = true
				return errors.New("rewire would change saved stash values")
			}
		}
	}
	b, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	p := stashPath()
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	if _, err := stepcodeReadStash(); err != nil {
		return err
	}
	// made 0600 first: WriteAtomic keeps a file's mode, and the stash holds
	// keys
	if f, err := os.OpenFile(p, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600); err == nil {
		f.Close()
	} else if !errors.Is(err, fs.ErrExist) {
		return err
	}
	if err := edit.WriteAtomic(p, b); err != nil {
		return err
	}
	if guard != nil && wired != nil {
		for k := range before {
			if _, exists := m[k]; strings.HasSuffix(k, wiredAt) && !exists {
				delete(wired, k)
			}
		}
		for k, v := range m {
			if old, exists := before[k]; strings.HasSuffix(k, wiredAt) && (!exists || old != v) {
				wired[k] = v
			}
		}
	}
	return nil
}

// stash keeps kv in the stash, a value there being removed. An error says
// the change wasn't written: the file still reads as it did, so a caller
// keeping a value before writing over it stops instead of going on.
func stash(kv map[string]string) error {
	return stashFor(kv, nil, false)
}

// stashFor carries the guard belonging to a rewire operation when scoped is set.
// Unscoped callers keep the legacy guard until their operation is propagated.
func stashFor(kv map[string]string, guard *bool, scoped bool) error {
	return stashOwned(kv, guard, scoped, nil)
}

func stashOwned(kv map[string]string, guard *bool, scoped bool, wired map[string]string) error {
	unlock, err := stashLock()
	if err != nil {
		return err
	}
	defer unlock()
	m, err := stepcodeReadStash()
	if err != nil {
		return err
	}
	for k, v := range kv {
		if v == "" {
			delete(m, k)
		} else {
			m[k] = v
		}
	}
	if scoped {
		return stashWriteGuard(m, guard, wired)
	}
	return stashWrite(m)
}

func (g *rewireGuard) stash(kv map[string]string) error {
	if g == nil {
		return stash(kv)
	}
	return stashOwned(kv, &g.blocked, true, g.wiredAt)
}

func (g *rewireGuard) forget(keys ...string) {
	if g == nil {
		forget(keys...)
		return
	}
	unlock, err := stashLock()
	if err != nil {
		log.Printf("stash kept %v: %v", keys, err)
		return
	}
	defer unlock()
	m, err := stepcodeReadStash()
	if err != nil {
		log.Printf("stash kept %v: %v", keys, err)
		return
	}
	for _, key := range keys {
		delete(m, key)
	}
	if err := stashWriteGuard(m, &g.blocked, g.wiredAt); err != nil {
		log.Printf("stash kept %v: %v", keys, err)
	}
}

// forget drops stashed values without restoring them. A change that can't
// be written leaves them there, which keeps too much rather than too
// little, so it is logged and not returned.
func forget(keys ...string) {
	unlock, err := stashLock()
	if err != nil {
		log.Printf("stash kept %v: %v", keys, err)
		return
	}
	defer unlock()
	m, err := stepcodeReadStash()
	if err != nil {
		log.Printf("stash kept %v: %v", keys, err)
		return
	}
	for _, k := range keys {
		delete(m, k)
	}
	if err := stashWrite(m); err != nil {
		log.Printf("stash kept %v: %v", keys, err)
	}
}

// peekStash reads a saved value without consuming it before a config write.
func peekStash(key string) (string, error) {
	unlock, err := stashLock()
	if err != nil {
		return "", err
	}
	defer unlock()
	m, err := stepcodeReadStash()
	if err != nil {
		return "", err
	}
	return m[key], nil
}

// forgetIfUnchanged removes a restored value only if another writer has not replaced it.
func forgetIfUnchanged(key, value string) {
	(*rewireGuard)(nil).forgetIfUnchanged(key, value)
}

func (g *rewireGuard) forgetIfUnchanged(key, value string) {
	unlock, err := stashLock()
	if err != nil {
		log.Printf("stash kept %s: %v", key, err)
		return
	}
	defer unlock()
	m, err := stepcodeReadStash()
	if err != nil {
		log.Printf("stash kept %s: %v", key, err)
		return
	}
	if m[key] != value || value == "" {
		return
	}
	delete(m, key)
	if g == nil {
		err = stashWrite(m)
	} else {
		err = stashWriteGuard(m, &g.blocked, g.wiredAt)
	}
	if err != nil {
		log.Printf("stash kept %s: %v", key, err)
	}
}

// unstash takes key's value out of the stash. An error says the take wasn't
// written, the value still there to be put back again; the value itself is
// given back as well, since it is the user's.
func unstash(key string) (string, error) {
	return (*rewireGuard)(nil).unstash(key)
}

func (g *rewireGuard) unstash(key string) (string, error) {
	unlock, err := stashLock()
	if err != nil {
		return "", err
	}
	defer unlock()
	m, err := stepcodeReadStash()
	if err != nil {
		return "", err
	}
	v := m[key]
	if v == "" {
		return "", nil
	}
	delete(m, key)
	if g != nil {
		return v, stashWriteGuard(m, &g.blocked, g.wiredAt)
	}
	return v, stashWrite(m)
}
