package collectors

import (
	"strings"
	"testing"

	"github.com/agentz/cctrace/internal/trace"
)

func TestParseTranscriptJSONL(t *testing.T) {
	input := strings.NewReader(`{"type":"tool_call","tool":"Bash","id":"tool_1","timestamp":"2026-05-26T10:00:00Z"}
{"type":"tool_result","tool":"Bash","id":"tool_1","status":"ok","timestamp":"2026-05-26T10:00:01Z"}
`)
	events, err := ParseTranscriptJSONL("sess_1", "claude", "session.jsonl", input)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 2 {
		t.Fatalf("len(events) = %d, want 2", len(events))
	}
	if events[0].Type != trace.EventToolCall || events[0].Title != "Bash" {
		t.Fatalf("event[0] = %#v", events[0])
	}
	if events[1].Type != trace.EventToolResult || events[1].Status != trace.StatusOK {
		t.Fatalf("event[1] = %#v", events[1])
	}
}

func TestParseClaudeMessageContentSummaries(t *testing.T) {
	input := strings.NewReader(`{"type":"user","uuid":"user_1","timestamp":"2026-05-27T01:43:39.055Z","message":{"role":"user","content":"没看到我们会话的内容"}}
{"type":"assistant","uuid":"assistant_1","timestamp":"2026-05-27T01:43:43.954Z","message":{"role":"assistant","content":[{"type":"text","text":"我会修复 transcript 内容解析。"}]}}
`)
	events, err := ParseTranscriptJSONL("sess_1", "claude", "session.jsonl", input)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 2 {
		t.Fatalf("len(events) = %d, want 2", len(events))
	}
	if events[0].Title != "没看到我们会话的内容" || events[0].Summary["text"] != "没看到我们会话的内容" {
		t.Fatalf("user event = %#v", events[0])
	}
	if events[1].Title != "我会修复 transcript 内容解析。" || events[1].Summary["text"] != "我会修复 transcript 内容解析。" {
		t.Fatalf("assistant event = %#v", events[1])
	}
}

func TestWatchTranscriptParsesLargeLine(t *testing.T) {
	largeText := strings.Repeat("x", 1024*1024) + " visible tail"
	line := `{"type":"user","uuid":"user_1","timestamp":"2026-05-27T01:43:39.055Z","message":{"role":"user","content":"` + largeText + `"}}` + "\n"
	events, err := ParseTranscriptJSONL("sess_1", "claude", "session.jsonl", strings.NewReader(line))
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 || !strings.Contains(events[0].Summary["text"].(string), "visible tail") {
		t.Fatalf("events = %#v", events)
	}
}
