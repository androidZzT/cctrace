package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/agentz/cctrace/internal/trace"
)

func TestEventsEndpointReturnsPublishedEvents(t *testing.T) {
	hub := NewHub()
	hub.Publish(trace.Event{ID: "evt_1", SessionID: "sess_1", Type: trace.EventProcess, Title: "started", Timestamp: time.UnixMilli(1000), Status: trace.StatusOK, Source: trace.SourceProcess, CorrelationIDs: []string{}, Confidence: trace.ConfidenceExact, Summary: map[string]any{}})
	req := httptest.NewRequest(http.MethodGet, "/api/events", nil)
	res := httptest.NewRecorder()
	NewHandler(hub).ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("status = %d", res.Code)
	}
	var events []trace.Event
	if err := json.Unmarshal(res.Body.Bytes(), &events); err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 || events[0].ID != "evt_1" {
		t.Fatalf("events = %#v", events)
	}
}
