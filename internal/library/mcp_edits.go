package library

import (
	"encoding/json"
	"log"
	"slices"
)

// takeMCPEdits takes into the library each entry magpie wrote into an
// agent's MCP file where the server was changed since: by the user, or by
// the agent's own MCP settings. The sync after gives the change to every
// agent, rather than writing magpie's old entry over it, as takeEdits does
// for a skill's copy (#3822). The user's own keys of an entry (a timeout,
// enabled, a tool list) aren't part of the server, and stay where they are.
// An entry given through the gateway, or one a WSL twin rewrites, is not the
// library's to take: it differs from the library's on purpose.
func (l *Library) takeMCPEdits(targets []*Target) {
	for _, t := range targets {
		if t.MCP == nil || t.MCP.WSL {
			continue
		}
		a := l.Applied[t.Agent.ID]
		if a == nil || len(a.MCPEntries) == 0 {
			continue
		}
		// a file that can't be read is unknown, never taken as unchanged
		es, err := t.MCP.entries()
		if err != nil {
			continue
		}
		for _, s := range l.MCP {
			if !slices.Contains(s.Agents, t.Agent.ID) || relayed(s) {
				continue
			}
			wrote := a.MCPEntries[s.Name]
			old := es[s.Name]
			if wrote == nil || old == nil {
				continue
			}
			cur, ok := t.MCP.decode(s.Name, old)
			if !ok {
				continue
			}
			var was map[string]any
			if json.Unmarshal(wrote, &was) != nil {
				continue
			}
			if prev, ok := t.MCP.decode(s.Name, was); !ok || prev.same(cur) {
				continue
			}
			c := *s
			c.Transport, c.Command, c.Args, c.Env, c.URL, c.Headers = cur.Transport, cur.Command, cur.Args, cur.Env, cur.URL, cur.Headers
			log.Printf("library: %s changed %s in its MCP file; the library takes it, and the other agents get it", t.Agent.ID, s.Name)
			*s = c
		}
	}
}
