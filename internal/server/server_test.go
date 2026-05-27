package server

import (
	"bufio"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/agentz/cctrace/internal/trace"
)

func TestSessionEndpointReturnsMetadata(t *testing.T) {
	hub := NewHub()
	hub.SetSession(SessionMetadata{ID: "sess_1", Provider: "claude", Mode: "live", Command: []string{"claude"}})
	req := httptest.NewRequest(http.MethodGet, "/api/session", nil)
	res := httptest.NewRecorder()
	NewHandler(hub).ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("status = %d", res.Code)
	}
	var metadata SessionMetadata
	if err := json.Unmarshal(res.Body.Bytes(), &metadata); err != nil {
		t.Fatal(err)
	}
	if metadata.ID != "sess_1" || metadata.Provider != "claude" || metadata.Mode != "live" {
		t.Fatalf("metadata = %#v", metadata)
	}
}

func TestIndexIncludesPerfettoTimelineAffordances(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	res := httptest.NewRecorder()
	NewHandler(NewHub()).ServeHTTP(res, req)
	body := res.Body.String()
	for _, want := range []string{
		"timeline-shell",
		"time-ruler",
		"lane-labels",
		"lane-track",
		"event-block",
		"event-marker",
		"agentLaneForEvent(event)",
		"agentLaneKey(event)",
		"discoverAgentLanes(events)",
		"positionEventNode",
		"Main Session",
		"API / Model",
		"System / Hooks",
		"session-mode",
		"/api/session",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("index HTML missing %q", want)
		}
	}
}

func TestIndexIncludesTimelineOverviewAndZoomControls(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	res := httptest.NewRecorder()
	NewHandler(NewHub()).ServeHTTP(res, req)
	body := res.Body.String()
	for _, want := range []string{
		"overview-shell",
		"overview-canvas",
		"overview-window",
		"drawOverviewCanvas(items, sessionStart, sessionEnd)",
		"renderOverviewWindow(sessionStart, sessionEnd)",
		"zoom-out",
		"zoom-reset",
		"zoom-in",
		"function zoomViewportAt(clientX, nextSpanMs)",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("index HTML missing %q", want)
		}
	}
}

func TestIndexIncludesWallClockGestureZoomAndOverviewSeek(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	res := httptest.NewRecorder()
	NewHandler(NewHub()).ServeHTTP(res, req)
	body := res.Body.String()
	for _, want := range []string{
		"formatClockTime(sessionStart + ms)",
		"function formatClockTime(timestampMs)",
		"toLocaleTimeString",
		"function zoomViewportAt(clientX, nextSpanMs)",
		"timelineShell.addEventListener('wheel'",
		"event.ctrlKey || event.metaKey",
		"gesturestart",
		"gesturechange",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("index HTML missing %q", want)
		}
	}
}

func TestIndexIncludesOverviewRangeSelectionAndVirtualizedRendering(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	res := httptest.NewRecorder()
	NewHandler(NewHub()).ServeHTTP(res, req)
	body := res.Body.String()
	for _, want := range []string{
		"let viewportStart = null",
		"let viewportEnd = null",
		"function initializeViewport(renderItems)",
		"function setViewport(startMs, endMs)",
		"function overviewMsFromClientX(clientX)",
		"function startOverviewRangeSelection(event)",
		"function updateOverviewRangeSelection(event)",
		"function finishOverviewRangeSelection()",
		"overviewCanvas.addEventListener('pointerdown', startOverviewRangeSelection)",
		"const visibleItems = filteredItems.filter((item) => item.end >= viewportStart && item.start <= viewportEnd)",
		"drawOverviewCanvas(filteredItems, traceStart, traceEnd)",
		"visibleItems.forEach((item) => positionEventNode(item, width))",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("index HTML missing %q", want)
		}
	}
}

func TestIndexIncludesInitialFiveMinuteFocusProgressiveLoadAndRangeDrag(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	res := httptest.NewRecorder()
	NewHandler(NewHub()).ServeHTTP(res, req)
	body := res.Body.String()
	for _, want := range []string{
		"const maxInitialViewportMs = 5 * 60 * 1000",
		"let renderItems = []",
		"function buildRenderModel(events)",
		"renderItems = buildRenderModel(allEvents)",
		"viewportStart = Math.max(traceStart, traceEnd - maxInitialViewportMs)",
		"let overviewDragMode = ''",
		"overviewDragMode = overviewDragHitMode(event)",
		"previewViewport(nextStart, nextEnd)",
		"function panViewport(deltaMs)",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("index HTML missing %q", want)
		}
	}
}

func TestIndexIncludesEventTypeUIAffordances(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	res := httptest.NewRecorder()
	NewHandler(NewHub()).ServeHTTP(res, req)
	body := res.Body.String()
	for _, want := range []string{
		"data-filter=\"user_message\"",
		"data-filter=\"agent_turn\"",
		"data-filter=\"tool_call\"",
		"data-filter=\"skill\"",
		"data-filter=\"process\"",
		"typeBadge(event.type)",
		"summary.description",
		"Raw JSON",
		"event-block",
		".skill",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("index HTML missing %q", want)
		}
	}
}

func TestIndexUsesPerfettoViewportCanvasOverviewAndBoundedLabels(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	res := httptest.NewRecorder()
	NewHandler(NewHub()).ServeHTTP(res, req)
	body := res.Body.String()
	for _, want := range []string{
		"<canvas class=\"overview-canvas\" id=\"overview-canvas\"></canvas>",
		"let viewportStart = null",
		"let viewportEnd = null",
		"function buildRenderModel(events)",
		"function shortText(value, maxLength)",
		"function initializeViewport(renderItems)",
		"function panViewport(deltaMs)",
		"function zoomViewportAt(clientX, nextSpanMs)",
		"function drawOverviewCanvas(items, sessionStart, sessionEnd)",
		"function renderOverviewWindow(sessionStart, sessionEnd)",
		"function startTimelinePan(event)",
		"timelineShell.addEventListener('pointerdown', startTimelinePan)",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("index HTML missing %q", want)
		}
	}
	for _, forbidden := range []string{
		"function renderOverviewEvent(event, sessionStart, sessionEnd)",
		"events.forEach((event) => renderOverviewEvent(event, sessionStart, sessionEnd))",
		"overview-mini-event",
		"const width = Math.max(1400, (rangeEnd - rangeStart + 5000) * pxPerMs())",
	} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("index HTML still contains slow path %q", forbidden)
		}
	}
}

func TestIndexIncludesPersistentOverviewFrameHandlesAndDraftViewport(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	res := httptest.NewRecorder()
	NewHandler(NewHub()).ServeHTTP(res, req)
	body := res.Body.String()
	for _, want := range []string{
		"overview-frame-handle left",
		"overview-frame-handle right",
		"let draftViewportStart = null",
		"let draftViewportEnd = null",
		"let overviewDragFrameStart = 0",
		"let overviewDragFrameEnd = 0",
		"function previewViewport(startMs, endMs)",
		"function commitPreviewViewport()",
		"function overviewDragHitMode(event)",
		"overviewDragMode = overviewDragHitMode(event)",
		"if (overviewDragMode === 'resize-left')",
		"if (overviewDragMode === 'resize-right')",
		"previewViewport(nextStart, nextEnd)",
		"commitPreviewViewport()",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("index HTML missing %q", want)
		}
	}
}

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

func TestEventStreamBroadcastsPublishedEvents(t *testing.T) {
	hub := NewHub()
	srv := httptest.NewServer(NewHandler(hub))
	defer srv.Close()

	res, err := http.Get(srv.URL + "/api/events/stream")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", res.StatusCode)
	}
	if got := res.Header.Get("Content-Type"); !strings.HasPrefix(got, "text/event-stream") {
		t.Fatalf("Content-Type = %q", got)
	}

	hub.Publish(trace.Event{ID: "evt_stream", SessionID: "sess_1", Type: trace.EventUserMessage, Title: "hi", Timestamp: time.UnixMilli(1000), Status: trace.StatusOK, Source: trace.SourceClaudeTranscript, CorrelationIDs: []string{}, Confidence: trace.ConfidenceExact, Summary: map[string]any{}})

	lines := make(chan string, 4)
	go func() {
		scanner := bufio.NewScanner(res.Body)
		for scanner.Scan() {
			lines <- scanner.Text()
		}
	}()

	deadline := time.After(2 * time.Second)
	for {
		select {
		case line := <-lines:
			if !strings.HasPrefix(line, "data: ") {
				continue
			}
			if !strings.Contains(line, `"id":"evt_stream"`) {
				t.Fatalf("stream line = %q", line)
			}
			return
		case <-deadline:
			t.Fatal("timed out waiting for streamed event")
		}
	}
}
