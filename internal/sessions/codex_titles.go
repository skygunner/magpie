package sessions

import (
	"bufio"
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/yetone/magpie/internal/appdir"
)

// TitleApplication is an actual name write, not just a chat's current name.
// Keeping old writes lets an association survive a later manual rename.
type TitleApplication struct {
	Digest  string
	Updated time.Time
}
type titleRecipient struct {
	ID      string
	Updated time.Time
}

// codexTitleState is one folder's index as it was last read: kept per
// folder, so a second Codex folder's names are read beside the main
// folder's, whose cache they used to crowd out (#3530).
type codexTitleState struct {
	path         string
	size, offset int64
	modified     time.Time
	file         os.FileInfo
	names        map[string]string
	last         map[string]TitleApplication
	recipients   map[string][]titleRecipient
	writes       int
	overflow     bool
}

var codexTitles struct {
	sync.Mutex
	byPath map[string]*codexTitleState
}

const titleApplicationLimit = 100000

func codexTitleIndexPath() string { return filepath.Join(CodexDir(), "session_index.jsonl") }

// codexTitleIndexPaths are every Codex folder's name index, the main
// folder's first (#3530).
func codexTitleIndexPaths() []string {
	var out []string
	for _, home := range appdir.CodexHomes() {
		out = append(out, filepath.Join(home, "session_index.jsonl"))
	}
	return out
}

// stateOf is the folder's state, made if this is its first read. Caller
// holds codexTitles.
func stateOf(path string) *codexTitleState {
	if codexTitles.byPath == nil {
		codexTitles.byPath = map[string]*codexTitleState{}
	}
	st := codexTitles.byPath[path]
	if st == nil {
		st = &codexTitleState{}
		codexTitles.byPath[path] = st
	}
	return st
}

func CodexTitleDigest(name string) string {
	name = strings.Join(strings.Fields(name), " ")
	if name == "" {
		return ""
	}
	return codexFingerprint("title", name)
}

// codexTitleOf is CodexTitles of one id at a time, for a pass that
// assembles every session (pricer): the indexes are looked at on the first
// ask, and after (an unchanged file costs a stat, the incremental read
// standing). A stat of it for each Codex session was ~60ms of a start's
// stats (#3493). The main folder's name stands when two folders name one
// id (#3530).
func codexTitleOf() func(id string) string {
	var asked bool
	return func(id string) string {
		codexTitles.Lock()
		defer codexTitles.Unlock()
		// every folder's index on the pass's first ask, then frozen for the
		// pass: one pass reads a session's file and its name as they were
		// together (#3493). Per-folder caches made the old re-read rule -
		// reload when another index was read in between - moot: no other
		// folder's read crowds this one out any more (#3530).
		if !asked {
			asked = true
			for _, path := range codexTitleIndexPaths() {
				loadCodexTitles(path)
			}
		}
		for _, path := range codexTitleIndexPaths() {
			if st := codexTitles.byPath[path]; st != nil {
				if name := st.names[id]; name != "" {
					return name
				}
			}
		}
		return ""
	}
}

// CodexTitles reads only the append-only name indexes, of every Codex
// folder (#3530). They cache unchanged files and read newly appended
// complete lines from the previous offset.
func CodexTitles(ids []string) map[string]string {
	out := make(map[string]string)
	if len(ids) == 0 {
		return out
	}
	codexTitles.Lock()
	defer codexTitles.Unlock()
	for _, path := range codexTitleIndexPaths() {
		if !loadCodexTitles(path) {
			continue
		}
		st := codexTitles.byPath[path]
		for _, id := range ids {
			if out[id] == "" {
				out[id] = st.names[id]
			}
		}
	}
	return out
}

func codexTitlesAt(path string, ids []string) map[string]string {
	out := make(map[string]string)
	if len(ids) == 0 {
		return out
	}
	codexTitles.Lock()
	defer codexTitles.Unlock()
	if !loadCodexTitles(path) {
		return out
	}
	st := codexTitles.byPath[path]
	for _, id := range ids {
		if name := st.names[id]; name != "" {
			out[id] = name
		}
	}
	return out
}

// CodexTitleRecipients includes chats whose requests never crossed Magpie.
// They still count as conflicting recipients of a generated title.
func CodexTitleRecipients(digests []string) map[string][]TitleApplication {
	out := map[string][]TitleApplication{}
	if len(digests) == 0 {
		return out
	}
	codexTitles.Lock()
	defer codexTitles.Unlock()
	for _, path := range codexTitleIndexPaths() {
		if !loadCodexTitles(path) {
			continue
		}
		st := codexTitles.byPath[path]
		if st.overflow {
			continue
		}
		for _, digest := range digests {
			for _, recipient := range st.recipients[digest] {
				out[recipient.ID] = append(out[recipient.ID], TitleApplication{digest, recipient.Updated})
			}
		}
	}
	return out
}

// Caller holds codexTitles. Truncation, replacement or same-size rewriting
// rebuilds the folder's cache; an incomplete last line waits for the next
// append.
func loadCodexTitles(path string) bool {
	info, err := os.Stat(path)
	if err != nil || !info.Mode().IsRegular() {
		return false
	}
	st := stateOf(path)
	if st.path == path && st.size == info.Size() && st.modified.Equal(info.ModTime()) {
		return true
	}
	file, err := os.Open(path)
	if err != nil {
		return false
	}
	defer file.Close()
	appendOnly := st.path == path && st.file != nil && os.SameFile(st.file, info) && info.Size() > st.size
	if !appendOnly {
		st.offset = 0
		st.names = map[string]string{}
		st.last = map[string]TitleApplication{}
		st.recipients = map[string][]titleRecipient{}
		st.writes, st.overflow = 0, false
	}
	if _, err := file.Seek(st.offset, 0); err != nil {
		return false
	}
	scan := bufio.NewScanner(file)
	scan.Buffer(make([]byte, 4096), 1<<20)
	offset := st.offset
	scan.Split(func(data []byte, atEOF bool) (int, []byte, error) {
		if i := bytes.IndexByte(data, '\n'); i >= 0 {
			offset += int64(i + 1)
			return i + 1, data[:i], nil
		}
		return 0, nil, nil // never commit an unterminated row
	})
	cutoff := time.Now().AddDate(0, 0, -31)
	for scan.Scan() {
		var row struct {
			ID      string    `json:"id"`
			Name    string    `json:"thread_name"`
			Updated time.Time `json:"updated_at"`
		}
		if json.Unmarshal(scan.Bytes(), &row) != nil || row.ID == "" {
			continue
		}
		name := strings.Join(strings.Fields(row.Name), " ")
		if name == "" {
			continue
		}
		chars := []rune(name)
		st.names[row.ID] = string(chars[:min(len(chars), 200)])
		if row.Updated.Before(cutoff) || st.overflow {
			continue
		}
		a := TitleApplication{CodexTitleDigest(name), row.Updated}
		prev, seen := st.last[row.ID]
		if seen && prev == a {
			continue
		}
		if st.writes >= titleApplicationLimit {
			st.overflow = true
			st.last = nil
			st.recipients = nil
			continue
		}
		st.last[row.ID] = a
		st.recipients[a.Digest] = append(st.recipients[a.Digest], titleRecipient{row.ID, a.Updated})
		st.writes++
	}
	if scan.Err() != nil {
		st.path = ""
		return false
	}
	st.path, st.size, st.modified, st.file, st.offset = path, info.Size(), info.ModTime(), info, offset
	return true
}
