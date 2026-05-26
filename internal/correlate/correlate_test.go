package correlate

import (
	"testing"
	"time"

	"github.com/agentz/cctrace/internal/trace"
)

func TestCorrelateAPIRequestToNearestTool(t *testing.T) {
	base := time.UnixMilli(1000)
	events := []trace.Event{
		{ID: "tool_1", SessionID: "sess_1", Type: trace.EventToolCall, Title: "Bash", Timestamp: base, Status: trace.StatusOK, Source: trace.SourceClaudeTranscript, CorrelationIDs: []string{"tool_1"}, Confidence: trace.ConfidenceExact, Summary: map[string]any{}},
		{ID: "api_1", SessionID: "sess_1", Type: trace.EventAPIRequest, Title: "model", Timestamp: base.Add(500 * time.Millisecond), Status: trace.StatusOK, Source: trace.SourceCcglass, CorrelationIDs: []string{"req_1"}, Confidence: trace.ConfidenceExact, Summary: map[string]any{}},
	}
	got := Correlate(events, 2*time.Second)
	if got[1].ParentID != "tool_1" {
		t.Fatalf("ParentID = %q, want tool_1", got[1].ParentID)
	}
	if got[1].Confidence != trace.ConfidenceLikely {
		t.Fatalf("Confidence = %q, want likely", got[1].Confidence)
	}
}

func TestCorrelateLeavesFarAPIOrphan(t *testing.T) {
	base := time.UnixMilli(1000)
	events := []trace.Event{
		{ID: "tool_1", SessionID: "sess_1", Type: trace.EventToolCall, Timestamp: base, CorrelationIDs: []string{}, Confidence: trace.ConfidenceExact, Summary: map[string]any{}},
		{ID: "api_1", SessionID: "sess_1", Type: trace.EventAPIRequest, Timestamp: base.Add(10 * time.Second), CorrelationIDs: []string{}, Confidence: trace.ConfidenceExact, Summary: map[string]any{}},
	}
	got := Correlate(events, 2*time.Second)
	if got[1].ParentID != "" {
		t.Fatalf("ParentID = %q, want empty", got[1].ParentID)
	}
	if got[1].Confidence != trace.ConfidenceUnknown {
		t.Fatalf("Confidence = %q, want unknown", got[1].Confidence)
	}
}
