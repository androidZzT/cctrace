package collectors

import (
	"strings"
	"testing"

	"github.com/agentz/cctrace/internal/trace"
)

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
	if events[1].Type != trace.EventAgentTurn || events[1].Title != "assistant" {
		t.Fatalf("assistant event = %#v", events[1])
	}
}
