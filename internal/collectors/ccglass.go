package collectors

import (
	"encoding/json"
	"io"
	"time"

	"github.com/agentz/cctrace/internal/trace"
)

type ccglassRecord struct {
	RequestID string         `json:"requestId"`
	StartedAt string         `json:"startedAt"`
	EndedAt   string         `json:"endedAt"`
	Model     string         `json:"model"`
	Usage     map[string]any `json:"usage"`
}

func ParseCcglassJSON(sessionID, file string, r io.Reader) ([]trace.Event, error) {
	var rec ccglassRecord
	if err := json.NewDecoder(r).Decode(&rec); err != nil {
		return nil, err
	}
	started, err := time.Parse(time.RFC3339, rec.StartedAt)
	if err != nil {
		started = time.Now()
	}
	ended, err := time.Parse(time.RFC3339, rec.EndedAt)
	if err != nil {
		ended = started
	}
	duration := ended.Sub(started).Milliseconds()
	reqID := rec.RequestID
	if reqID == "" {
		reqID = stableID("ccglass", started)
	}
	request := trace.Event{ID: stableID("api_request", reqID), SessionID: sessionID, Type: trace.EventAPIRequest, Title: rec.Model, Timestamp: started, Status: trace.StatusRunning, Source: trace.SourceCcglass, CorrelationIDs: []string{reqID}, Confidence: trace.ConfidenceExact, Summary: map[string]any{"model": rec.Model}, RawRef: &trace.RawRef{File: file, RequestID: reqID}}
	response := trace.Event{ID: stableID("api_response", reqID), SessionID: sessionID, Type: trace.EventAPIResponse, Title: rec.Model, Timestamp: ended, DurationMS: &duration, Status: trace.StatusOK, Source: trace.SourceCcglass, CorrelationIDs: []string{reqID}, Confidence: trace.ConfidenceExact, Summary: map[string]any{"model": rec.Model, "usage": rec.Usage}, RawRef: &trace.RawRef{File: file, RequestID: reqID}}
	return []trace.Event{request, response}, nil
}
