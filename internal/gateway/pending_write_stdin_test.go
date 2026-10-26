package gateway

import (
	"encoding/json"
	"fmt"
	"testing"
)

// TestPendingWriteStdinLoopGuard distinguishes a pending wait from repeated work.
// All handles and outputs are synthetic; no model, process, or network is used.
func TestPendingWriteStdinLoopGuard(t *testing.T) {
	for _, tc := range []struct {
		name string
		tool string
		args string
		text string
		stop bool
	}{
		{
			name: "pending_wait_must_continue",
			tool: "write_stdin",
			args: `{"chars":"","max_output_tokens":10000,"session_id":4242,"yield_time_ms":30000}`,
			text: "Process running with session ID 4242\nOriginal token count: 0\nOutput:\n",
			stop: false,
		},
		{
			name: "repeated_read_must_still_stop",
			tool: "exec_command",
			args: `{"cmd":"cat example.txt"}`,
			text: "Process exited with code 0\nOriginal token count: 1\nOutput:\nunchanged\n",
			stop: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// Eight exchanges use unique call IDs and normal execution metadata.
			var msgs []Message
			for i := range 8 {
				id := fmt.Sprintf("synthetic_call_%d", i)
				result := fmt.Sprintf("Chunk ID: %06x\nWall time: 30.%04d seconds\n%s", i+1, i, tc.text)
				msgs = append(msgs,
					Message{Role: "assistant", Parts: []Part{{
						Kind: ToolCall, ID: id, Name: tc.tool, Args: json.RawMessage(tc.args),
					}}},
					Message{Role: "user", Parts: []Part{{
						Kind: ToolResult, CallID: id, Text: result,
					}}},
				)
			}
			if _, stopped := toolLoopOf(msgs); stopped != tc.stop {
				t.Fatalf("stopped=%t, want %t for %s", stopped, tc.stop, tc.tool)
			}
		})
	}
}
