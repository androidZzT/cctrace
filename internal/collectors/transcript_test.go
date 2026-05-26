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
