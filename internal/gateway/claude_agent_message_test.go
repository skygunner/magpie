package gateway

import (
	"strconv"
	"strings"
	"testing"
)

// What the client's agent sends when a sub-agent hands its report back with
// SubagentHandback while its parent is mid-turn (#2868, Claude Desktop on a
// Claude subscription, Claude Code 2.1.302): the call's result says the
// report went as a message, and the report itself follows the results as a
// {role: "system"} message, or as text of their message, wrapped as the
// agent builds one: `<agent-message from="…">`, the report, `</agent-message>`.
const (
	handbackNotice  = `This agent's report was delivered to you as a message from "a1c74891359cffa5e" (its SubagentHandback call). Read it there; it is not repeated here.`
	reportText      = "seq 1 30 跑完了，输出就是 1 到 30。\n\n结论：共 30 行，从 1 开始，没有缺行。"
	handbackMessage = "<agent-message from=\"a1c74891359cffa5e\">\n" + reportText + "\n</agent-message>"
)

var handbackStart = `{"model":"claude-opus-5-5","max_tokens":64,"messages":[` +
	`{"role":"user","content":"派一个子代理跑 seq 1 30，把结果列出来"},` +
	`{"role":"assistant","content":[{"type":"tool_use","id":"call-1","name":"Agent","input":{"prompt":"run seq 1 30"}}]},` +
	`{"role":"user","content":[{"tool_use_id":"call-1","type":"tool_result","content":` + strconv.Quote(handbackNotice) + `,"is_error":false}`

func handbackBody(tail string) string { return handbackStart + tail + `]}` }

// A sub-agent's report that comes after a tool's results reaches the run
// waiting on them, with its <agent-message> wrapper, and is not written to
// the run's Claude Code as a message of the user's. Before, the report was
// left out as the client's agent's notice, and the run's model was told to
// read a message it never got (#2868).
func TestClaudeAgentMessageGoesOnWithTheResults(t *testing.T) {
	for _, tc := range []struct {
		name string
		body string
	}{{
		"as a system message",
		handbackBody(`]},{"role":"system","content":[{"type":"text","text":` + strconv.Quote(handbackMessage) + `,"cache_control":{"type":"ephemeral"}}]}`),
	}, {
		"in a reminder after the results",
		handbackBody(`,{"type":"text","text":` + strconv.Quote("<system-reminder>\n"+handbackMessage+"\n</system-reminder>") + `}]}`),
	}, {
		"bare after the results",
		handbackBody(`,{"type":"text","text":` + strconv.Quote(handbackMessage) + `}]}`),
	}, {
		"in a user message after the results",
		handbackBody(`]},{"role":"user","content":[{"type":"text","text":` + strconv.Quote(handbackMessage) + `}]}`),
	}} {
		t.Run(tc.name, func(t *testing.T) {
			got, said, _ := continuing(t, tc.body)
			text := resultText(got)
			if !strings.HasPrefix(text, handbackNotice) {
				t.Fatalf("call-1 was answered %q, want the tool's result first", text)
			}
			if !strings.Contains(text, `<agent-message from="a1c74891359cffa5e">`) || !strings.Contains(text, "结论：共 30 行") {
				t.Fatalf("the report never reached the run: call-1 was answered %q", text)
			}
			if len(said) != 0 {
				t.Fatalf("a sub-agent's report was written to Claude Code as the user's: %q", said)
			}
		})
	}
}

// A message the user sent mid-turn beside the report is still theirs, alone,
// and the report goes with the results.
func TestClaudeAgentMessageBesideTheUsersMessage(t *testing.T) {
	const user = "The user sent a new message while you were working:\n顺便告诉我现在是几点"
	body := handbackBody(`]},{"role":"system","content":[{"type":"text","text":` + strconv.Quote(user+"\n\n"+handbackMessage) + `}]}`)
	got, said, _ := continuing(t, body)
	if len(said) != 1 || said[0] != "顺便告诉我现在是几点" {
		t.Fatalf("Claude Code was written %q, want the user's words alone", said)
	}
	text := resultText(got)
	if !strings.Contains(text, "结论：共 30 行") {
		t.Fatalf("the report didn't go with the results: %q", text)
	}
	if strings.Contains(text, "顺便告诉我现在是几点") {
		t.Fatalf("the user's words went with the results: %q", text)
	}
}

// A report that comes as the next turn, after a reply, is told to the run as
// it is.
func TestClaudeTurnToldTheAgentMessage(t *testing.T) {
	msgs := []Message{{Role: "user", Parts: []Part{{Kind: Text, Text: handbackMessage}}}}
	var turn strings.Builder
	for _, b := range renderClaudeTurn(msgs, nil) {
		if s, ok := b["text"].(string); ok {
			turn.WriteString(s)
		}
	}
	if !strings.Contains(turn.String(), "结论：共 30 行") {
		t.Fatalf("the turn wasn't told the report:\n%s", turn.String())
	}
}
