package gateway

import (
	"encoding/json"
	"strings"
	"testing"
)

// #3684: narration without calls must not ask the client to run a tool.
func TestChatPhantomToolFinish(t *testing.T) {
	for _, field := range []string{"", `,"tool_calls":[]`, `,"tool_calls":null`, `,"tool_calls":[ ]`} {
		for _, tools := range []string{"", "," + execChatTool} {
			t.Run(field+tools, func(t *testing.T) {
				// Given: a valid upstream reply with no calls and a phantom finish.
				ask := `{"model":"a/m","messages":[{"role":"user","content":"go"}]` + tools + `}`
				end := strings.Replace(stopChunk, `"stop"`, `"tool_calls"`, 1)
				end = strings.Replace(end, `"delta":{}`, `"delta":{`+strings.TrimPrefix(field, ",")+`}`, 1)
				up := `{"id":"c1","object":"chat.completion","model":"m","choices":[{"index":0,"message":{"role":"assistant","content":"Checking."` + field + `},"finish_reason":"tool_calls"}]}`
				for _, frame := range []string{strings.TrimPrefix(end, "data: "), up} {
					if !json.Valid([]byte(frame)) {
						t.Fatalf("invalid fixture: %s", frame)
					}
				}
				t.Run("stream", func(t *testing.T) {
					// When: the Chat HTTP surface relays the streamed reply.
					text, calls, finish, raw := relayedChat(t, strings.Replace(ask, `"model":`, `"stream":true,"model":`, 1), chunkOf("Checking."), end)
					// Then: narration survives, with no invented calls.
					if finish != "stop" || len(calls) != 0 || text != "Checking." {
						t.Fatalf("finish %q, want stop; text %q calls %v\n%s", finish, text, calls, raw)
					}
				})
				t.Run("whole", func(t *testing.T) {
					// When: the Chat HTTP surface relays the whole reply.
					text, calls, finish, raw := wholeChat(t, ask, "application/json", up)
					// Then: it makes the same correction without losing text.
					if finish != "stop" || len(calls) != 0 || strOr(text) != "Checking." {
						t.Fatalf("finish %q, want stop; text %q calls %+v\n%s", finish, strOr(text), calls, raw)
					}
				})
			})
		}
	}
}

func TestChatPhantomFinishKeepsRealCalls(t *testing.T) {
	for _, parsed := range []bool{false, true} {
		t.Run(map[bool]string{false: "native", true: "text parsed"}[parsed], func(t *testing.T) {
			// Given: a native call or a text call that can be parsed.
			ask := `{"model":"a/m","messages":[{"role":"user","content":"go"}],` + execChatTool + `}`
			end := strings.Replace(stopChunk, `"stop"`, `"tool_calls"`, 1)
			call := nativeExec
			up := `{"choices":[{"index":0,"message":{"role":"assistant","content":"Checking.","tool_calls":[{"id":"call_up","type":"function","function":{"name":"exec_command","arguments":"{\"cmd\":\"ls -la\"}"}}]},"finish_reason":"tool_calls"}]}`
			if parsed {
				content := `Checking.<tool_call>{"name":"exec_command","arguments":{"cmd":"ls -la"}}</tool_call>`
				call = chunkOf(content)
				up = strings.Replace(wholeChatUp(content), `"stop"`, `"tool_calls"`, 1)
			}
			t.Run("stream", func(t *testing.T) {
				// When: a call precedes a final chunk with no calls of its own.
				_, calls, finish, raw := relayedChat(t, strings.Replace(ask, `"model":`, `"stream":true,"model":`, 1), call, end)
				// Then: the client still receives the actual call and its finish.
				if finish != "tool_calls" || len(calls) != 1 || calls["exec_command"] != `{"cmd":"ls -la"}` {
					t.Fatalf("finish %q calls %v\n%s", finish, calls, raw)
				}
			})
			t.Run("whole", func(t *testing.T) {
				// When: the same call arrives in a whole response.
				_, calls, finish, raw := wholeChat(t, ask, "application/json", up)
				// Then: normalization keeps it, including its arguments.
				if finish != "tool_calls" || len(calls) != 1 || calls[0].Function.Name != "exec_command" || calls[0].Function.Arguments != `{"cmd":"ls -la"}` {
					t.Fatalf("finish %q calls %+v\n%s", finish, calls, raw)
				}
			})
		})
	}
}

func TestChatPhantomFinishPerChoice(t *testing.T) {
	// Given: only choice 1 delivered a real native call.
	var tidy chatTidy
	tidy.write([]byte(strings.Replace(nativeExec, `"index":0,"delta"`, `"index":1,"delta"`, 1) + "\n\n"))
	end := `data: {"choices":[{"index":1,"delta":{},"finish_reason":"tool_calls"},{"index":0,"delta":{},"finish_reason":"tool_calls"}]}` + "\n\n"
	// When: both choices finish in the same chunk, in reverse order.
	out := tidy.write([]byte(end))
	// Then: each choice's finish follows its own calls.
	var got struct {
		Choices []struct {
			Index  int    `json:"index"`
			Finish string `json:"finish_reason"`
		} `json:"choices"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(strings.TrimPrefix(string(out), "data: "))), &got); err != nil {
		t.Fatal(err)
	}
	if len(got.Choices) != 2 || got.Choices[0].Index != 1 || got.Choices[0].Finish != "tool_calls" || got.Choices[1].Index != 0 || got.Choices[1].Finish != "stop" {
		t.Fatalf("choice finishes: %s", out)
	}
}
