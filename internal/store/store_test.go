package store

import (
	"strings"
	"testing"
	"time"

	"github.com/agentz/cctrace/internal/trace"
)

func TestStoreReadsLargeEventLines(t *testing.T) {
	dir := t.TempDir()
	store := New(dir)
	largeText := strings.Repeat("x", 1024*1024) + " visible tail"
	event := trace.Event{
		ID:        "evt_large",
		SessionID: "sess_large",
		Type:      trace.EventUserMessage,
		Title:     largeText,
		Timestamp: time.UnixMilli(1000),
		Status:    trace.StatusOK,
		Source:    trace.SourceClaudeTranscript,
		Summary:   map[string]any{"text": largeText},
	}
	if err := store.AppendEvent(event); err != nil {
		t.Fatal(err)
	}
	events, err := store.ReadEvents("sess_large")
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 || events[0].Title != largeText || events[0].Summary["text"] != largeText {
		t.Fatalf("events = %#v", events)
	}
}

func TestStoreAppendsAndReadsEvents(t *testing.T) {
	dir := t.TempDir()
	store := New(dir)
	session := trace.Session{ID: "sess_1", Command: []string{"claude"}, StartedAt: time.UnixMilli(1000)}
	if err := store.CreateSession(session); err != nil {
		t.Fatal(err)
	}
	event := trace.Event{
		ID: "evt_1", SessionID: "sess_1", Type: trace.EventProcess, Title: "started",
		Timestamp: time.UnixMilli(1000), Status: trace.StatusRunning, Source: trace.SourceProcess,
		CorrelationIDs: []string{}, Confidence: trace.ConfidenceExact, Summary: map[string]any{"pid": float64(123)},
	}
	if err := store.AppendEvent(event); err != nil {
		t.Fatal(err)
	}
	events, err := store.ReadEvents("sess_1")
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 || events[0].ID != "evt_1" || events[0].Summary["pid"] != float64(123) {
		t.Fatalf("events = %#v", events)
	}
}
