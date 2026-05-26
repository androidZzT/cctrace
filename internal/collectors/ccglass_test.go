package collectors

import (
	"strings"
	"testing"

	"github.com/agentz/cctrace/internal/trace"
)

func TestParseCcglassRequests(t *testing.T) {
	input := strings.NewReader(`{"requestId":"req_1","startedAt":"2026-05-26T10:00:00Z","endedAt":"2026-05-26T10:00:02Z","model":"claude-opus-4-7","usage":{"input_tokens":10,"output_tokens":20}}`)
	events, err := ParseCcglassJSON("sess_1", "request.json", input)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 2 {
		t.Fatalf("len(events) = %d, want 2", len(events))
	}
	if events[0].Type != trace.EventAPIRequest || events[0].RawRef.RequestID != "req_1" {
		t.Fatalf("request event = %#v", events[0])
	}
	if events[1].Type != trace.EventAPIResponse || events[1].Status != trace.StatusOK {
		t.Fatalf("response event = %#v", events[1])
	}
}
