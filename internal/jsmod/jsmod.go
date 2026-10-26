// Package jsmod compiles a plugin's JavaScript module for moejs, the
// JavaScript engine written in Go that magpie runs plugin code in its own
// process with: gateway middleware (internal/middleware) and agents
// (internal/agentplug).
package jsmod

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Calcium-Ion/moejs"
)

// File is a module's file as it was read: its path and its time and size
// from just before, so a loader can tell it changed.
type File struct {
	Path            string
	ModTimeUnixNano int64
	Size            int64
}

// Compile compiles the module in file and the modules it imports, which
// are files under root: there is no node_modules here, so a package is
// bundled into the module. what names the kind of module in an error
// ("a middleware"). The files it read are given back besides the module,
// the entry included, so a loader can load the module again when one of
// them changed (#3723): the imports are otherwise invisible to it, and
// an edit to an imported file kept running until the entry changed.
func Compile(root, file, what string) (*moejs.Module, []File, error) {
	var read []File
	one := func(p string) (string, error) {
		fi, err := os.Stat(p)
		if err != nil {
			return "", err
		}
		read = append(read, File{Path: p, ModTimeUnixNano: fi.ModTime().UnixNano(), Size: fi.Size()})
		b, err := os.ReadFile(p)
		if err != nil {
			return "", err
		}
		return string(b), nil
	}
	src, err := one(file)
	if err != nil {
		return nil, nil, err
	}
	entry, err := moejs.Compile(file, src)
	if err != nil {
		return nil, nil, err
	}
	mods := map[string]*moejs.Module{file: entry}
	mod, err := moejs.Link(entry, func(ref moejs.Referrer, spec string) (*moejs.Module, error) {
		if !strings.HasPrefix(spec, "./") && !strings.HasPrefix(spec, "../") {
			return nil, fmt.Errorf("%q: %s imports only files beside it; bundle packages into it", spec, what)
		}
		from := file
		if ref != nil {
			from = ref.Name()
		}
		p := filepath.Clean(filepath.Join(filepath.Dir(from), filepath.FromSlash(spec)))
		if rel, err := filepath.Rel(root, p); err != nil || strings.HasPrefix(rel, "..") {
			return nil, fmt.Errorf("%q is outside the plugin", spec)
		}
		if m, ok := mods[p]; ok {
			return m, nil
		}
		src, err := one(p)
		if err != nil {
			return nil, err
		}
		m, err := moejs.Compile(p, src)
		if err != nil {
			return nil, err
		}
		mods[p] = m
		return m, nil
	})
	return mod, read, err
}
