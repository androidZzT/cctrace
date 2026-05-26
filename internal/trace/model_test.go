package trace

import (
	"encoding/json"
	"testing"
	"time"
)

func TestEventDurationFromEndTimestamp(t *testing.T) {
	start := time.UnixMilli(1000)
	end := time.UnixMilli(1750)
	event := Event{Timestamp: start, EndTimestamp: &end}

	if got := event.Duration(); got != 750*time.Millisecond {
		t.Fatalf("Duration() = %s, want 750ms", got)
	}
}

func TestEventDurationUsesDurationMSWhenEndMissing(t *testing.T) {
	duration := int64(425)
	event := Event{Timestamp: time.UnixMilli(1000), DurationMS: &duration}

	if got := event.Duration(); got != 425*time.Millisecond {
		t.Fatalf("Duration() = %s, want 425ms", got)
	}
}

func TestEventJSONShape(t *testing.T) {
	event := Event{
		ID:             "evt_1",
		SessionID:      "sess_1",
		Type:           EventToolCall,
		Title:          "Bash",
		Timestamp:      time.UnixMilli(1000),
		Status:         StatusOK,
		Source:         SourceClaudeTranscript,
		CorrelationIDs: []string{"evt_parent"},
		Confidence:     ConfidenceExact,
		Summary:        map[string]any{"tool": "Bash"},
		RawRef:         &RawRef{File: "session.jsonl", Offset: 42},
	}

	data, err := json.Marshal(event)
	if err != nil {
		t.Fatal(err)
	}

	var decoded Event
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.ID != "evt_1" || decoded.Type != EventToolCall || decoded.Source != SourceClaudeTranscript {
		t.Fatalf("decoded event mismatch: %#v", decoded)
	}
	if decoded.RawRef == nil || decoded.RawRef.Offset != 42 {
		t.Fatalf("decoded raw ref mismatch: %#v", decoded.RawRef)
	}
}
