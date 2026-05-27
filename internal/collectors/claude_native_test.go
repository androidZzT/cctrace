package collectors

import (
	"strings"
	"testing"

	"github.com/agentz/cctrace/internal/trace"
)

func TestParseClaudeNativeAgentToolUseAsSubagent(t *testing.T) {
	input := strings.NewReader(`{"type":"assistant","uuid":"assistant_1","timestamp":"2026-05-27T07:13:15.406Z","message":{"role":"assistant","content":[{"type":"tool_use","id":"call_1","name":"Agent","input":{"description":"Subagent smoke test","subagent_type":"Explore","prompt":"Inspect files"}}]}}
`)
	events, err := ParseTranscriptJSONL("sess_1", "claude", "session.jsonl", input)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 {
		t.Fatalf("len(events) = %d, want 1", len(events))
	}
	event := events[0]
	if event.Type != trace.EventSubagent || event.Title != "Explore: Subagent smoke test" {
		t.Fatalf("event = %#v", event)
	}
	if event.Summary["prompt"] != "Inspect files" {
		t.Fatalf("prompt = %#v", event.Summary["prompt"])
	}
}

func TestParseClaudeNativeAgentToolResultAsSubagentResult(t *testing.T) {
	input := strings.NewReader(`{"type":"user","uuid":"result_1","timestamp":"2026-05-27T07:13:34.008Z","message":{"role":"user","content":[{"tool_use_id":"call_1","type":"tool_result","content":[{"type":"text","text":"Branch clean"}]}]},"toolUseResult":{"status":"completed","agentId":"agent_1","agentType":"Explore","totalDurationMs":15061,"totalTokens":27036,"totalToolUseCount":2,"content":[{"type":"text","text":"Branch clean"}]}}
`)
	events, err := ParseTranscriptJSONL("sess_1", "claude", "session.jsonl", input)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 {
		t.Fatalf("len(events) = %d, want 1", len(events))
	}
	event := events[0]
	if event.Type != trace.EventSubagent || event.Title != "Explore result" || event.Status != trace.StatusOK {
		t.Fatalf("event = %#v", event)
	}
	if event.Summary["result"] != "Branch clean" || event.Summary["totalTokens"] != float64(27036) {
		t.Fatalf("summary = %#v", event.Summary)
	}
	if event.DurationMS == nil || *event.DurationMS != 15061 {
		t.Fatalf("DurationMS = %#v", event.DurationMS)
	}
}

func TestParseClaudeNativeTranscript(t *testing.T) {
	input := strings.NewReader(`{"type":"user","uuid":"user_1","timestamp":"2026-05-27T01:43:39.055Z","message":{"role":"user","content":"hi"}}
{"type":"assistant","uuid":"assistant_1","timestamp":"2026-05-27T01:43:43.954Z","message":{"role":"assistant","content":[{"type":"text","text":"Hello"}],"usage":{"input_tokens":1,"output_tokens":2}}}
`)
	events, err := ParseTranscriptJSONL("sess_1", "claude", "session.jsonl", input)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 2 {
		t.Fatalf("len(events) = %d, want 2", len(events))
	}
	if events[0].Type != trace.EventUserMessage || events[0].Title != "hi" {
		t.Fatalf("user event = %#v", events[0])
	}
	if events[1].Type != trace.EventAgentTurn || events[1].Title != "Hello" {
		t.Fatalf("assistant event = %#v", events[1])
	}
}
