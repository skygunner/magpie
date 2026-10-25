package access

import "strings"

// RenameProvider moves every key's model and account lists from the id a
// provider had to the one it has now (#3630): a key held to some of its
// models or accounts keeps meaning them, where the editor showed them as
// not provided. An entry naming another provider, or a group, stays as it
// is, and no list gains one.
func RenameProvider(from, to string) (int, error) {
	if from == to {
		return 0, nil
	}
	mu.Lock()
	defer mu.Unlock()
	keys, err := load()
	if err != nil {
		return 0, err
	}
	moved := 0
	for i := range keys {
		for j, m := range keys[i].Models {
			if rest, ok := cutID(m, from); ok {
				keys[i].Models[j], moved = to+"/"+rest, moved+1
			}
		}
		for j, a := range keys[i].Accounts {
			if rest, ok := cutID(a, from); ok {
				keys[i].Accounts[j], moved = to+"/"+rest, moved+1
			}
		}
	}
	if moved == 0 {
		return 0, nil
	}
	return moved, save(keys)
}

// cutID is ref after the provider id it names, told apart in lower case as
// a key's lists are.
func cutID(ref, id string) (string, bool) {
	if len(ref) <= len(id) || !strings.EqualFold(ref[:len(id)], id) || ref[len(id)] != '/' {
		return "", false
	}
	return ref[len(id)+1:], true
}
