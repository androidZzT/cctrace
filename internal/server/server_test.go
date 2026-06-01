package server

import (
	"bufio"
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/androidZzT/cctrace/internal/trace"
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

func TestIndexIncludesSessionExplorerLayout(t *testing.T) {
	body := indexBody(t)
	for _, want := range []string{
		"Event Stream",
		"Thread",
		"请求轮次",
		"Details",
		"Raw JSON",
		"class=\"explorer-shell\"",
		"data-filter=\"skill\"",
		"SKILL LOAD",
		"SKILL USE",
		"skill-available",
		"skill-invoked",
		"eventBadgeLabel(event)",
		"id=\"thread-nav\"",
		"id=\"turn-nav\"",
		"id=\"turn-groups\"",
		"renderEventExplorer(turnGroups)",
		"renderOperationRow(operation)",
		"focusThread(thread)",
		"findThreadFocusItem(thread.id)",
		"focusStreamItem(item)",
		"data-item-id",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("index HTML missing %q", want)
		}
	}
}

func TestIndexIncludesImportOverviewAndViewportControls(t *testing.T) {
	body := indexBody(t)
	for _, want := range []string{
		"id=\"import-form\"",
		"id=\"import-path\"",
		"id=\"import-file\"",
		"id=\"import-folder\"",
		"id=\"file-choose\"",
		"id=\"folder-choose\"",
		"id=\"import-dropzone\"",
		"Session path, rollout JSONL, or folder",
		"multiple",
		"webkitdirectory",
		"class=\"drop-overlay\"",
		"drop-overlay-panel",
		"松开导入 Trace 包",
		"async function importSession(path)",
		"async function importUploadedFiles(files)",
		"async function importUploadedFile(file)",
		"fetch('/api/import'",
		"fetch('/api/import/upload'",
		"filesFromDrop(event)",
		"readEntryFiles(entry, prefix)",
		"droppedRelativePaths",
		"document.addEventListener('dragenter'",
		"document.addEventListener('drop'",
		"event.dataTransfer.files",
		"overview-canvas",
		"overview-window",
		"overview-frame-handle left",
		"overview-frame-handle right",
		"function buildEventDensityBuckets(items, sessionStart, sessionEnd, width)",
		"function renderOverviewWindow(sessionStart, sessionEnd)",
		"function startOverviewRangeSelection(event)",
		"function updateOverviewRangeSelection(event)",
		"overviewShell.addEventListener('wheel', handleOverviewWheel, { passive: false })",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("index HTML missing %q", want)
		}
	}
	for _, forbidden := range []string{
		"id=\"zoom-out\"",
		"id=\"zoom-reset\"",
		"id=\"zoom-in\"",
		"class=\"zoom-controls\"",
		"class=\"drop-zone\"",
		">Drop files<",
		"import-dropzone').onclick",
	} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("index HTML still contains zoom control %q", forbidden)
		}
	}
}

func TestIndexIncludesExplorerViewModelBuilders(t *testing.T) {
	body := indexBody(t)
	for _, want := range []string{
		"function buildTurnGroups(items, ops)",
		"const groups = []",
		"Session Setup",
		"会话准备",
		"请求 ",
		"event.type === 'user_message'",
		"function buildOperations(items)",
		"const calls = new Map()",
		"const results = new Map()",
		"resultItem ? resultItem.start : callItem.end",
		"function buildThreadSummary(items)",
		"Agent: ",
		"function buildEventDensityBuckets(items, sessionStart, sessionEnd, width)",
		"bucket.skill",
		"summary.phase || 'skill'",
		"item.event.type === 'subagent'",
		"setEventTypeFilter('all')",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("index HTML missing %q", want)
		}
	}
}

func TestIndexIncludesOperationRowsAndRelatedEvents(t *testing.T) {
	body := indexBody(t)
	for _, want := range []string{
		"Operation",
		"operation-row",
		"duration-bar",
		"duration-fill",
		"Related events",
		"id=\"related-toggle\"",
		"relatedOnly",
		"hasRelatedCorrelation(item)",
		"showOperation(operation)",
		"showEvent(item)",
		"Arguments",
		"Output",
		"Correlation",
		"operation.durationMs",
		"function focusTurnGroup(group)",
		"setViewport(start, end)",
		"node.scrollIntoView({ block: 'start', behavior: 'smooth' })",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("index HTML missing %q", want)
		}
	}
}

func TestIndexAvoidsTimelineLanePresentation(t *testing.T) {
	body := indexBody(t)
	for _, forbidden := range []string{
		"flame-event",
		"buildFlameGraphLayout",
		"buildStackedTimelineLayout",
		"lane-track",
		"timeline-shell",
		"positionEventNode",
		"renderTrack",
		"flameRowHeight",
	} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("index HTML still contains timeline-lane wording %q", forbidden)
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

func TestImportEndpointLoadsEventsJSONL(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "events.jsonl")
	event := trace.Event{ID: "evt_imported", SessionID: "sess_imported", Type: trace.EventUserMessage, Title: "hello", Timestamp: time.UnixMilli(1000), Status: trace.StatusOK, Source: trace.SourceCodexLog, CorrelationIDs: []string{}, Confidence: trace.ConfidenceExact, Summary: map[string]any{"provider": "codex"}}
	writeJSONLines(t, source, event)

	hub := NewHub()
	req := httptest.NewRequest(http.MethodPost, "/api/import", strings.NewReader(`{"path":`+quoteJSON(dir)+`}`))
	res := httptest.NewRecorder()
	NewHandler(hub).ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", res.Code, res.Body.String())
	}
	var imported SessionImportResponse
	if err := json.Unmarshal(res.Body.Bytes(), &imported); err != nil {
		t.Fatal(err)
	}
	if imported.EventCount != 1 || imported.Source != "events_jsonl" || imported.Session.ID != "sess_imported" {
		t.Fatalf("imported = %#v", imported)
	}
	if events := hub.Events(); len(events) != 1 || events[0].ID != "evt_imported" || events[0].SessionID != "sess_imported" {
		t.Fatalf("events = %#v", events)
	}
	if session := hub.Session(); session.Mode != "history" || session.Provider != "codex" {
		t.Fatalf("session = %#v", session)
	}
}

func TestImportEndpointParsesCodexRolloutJSONL(t *testing.T) {
	path := filepath.Join(t.TempDir(), "rollout.jsonl")
	writeRawLines(t, path,
		`{"timestamp":"2026-05-28T03:06:27.362Z","type":"session_meta","payload":{"id":"019e6ca8-742f-7283-b139-593c75443b37","timestamp":"2026-05-28T03:06:20.393Z","cwd":"/tmp/project","originator":"Codex Desktop","cli_version":"0.133.0","source":"vscode","thread_source":"user","model_provider":"openai"}}`,
		`{"timestamp":"2026-05-28T03:06:28.000Z","type":"event_msg","payload":{"type":"user_message","message":"hello codex"}}`,
		`{"timestamp":"2026-05-28T03:06:29.000Z","type":"response_item","payload":{"type":"function_call","name":"spawn_agent","call_id":"call_spawn","arguments":"{\"message\":\"check\"}"}}`,
		`{"timestamp":"2026-05-28T03:06:30.000Z","type":"response_item","payload":{"type":"function_call_output","call_id":"call_spawn","output":"{\"agent_id\":\"agent_1\",\"nickname\":\"Archimedes\"}"}}`,
		`{"timestamp":"2026-05-28T03:06:31.000Z","type":"response_item","payload":{"type":"function_call","name":"wait_agent","call_id":"call_wait","arguments":"{\"targets\":[\"agent_1\"]}"}}`,
		`{"timestamp":"2026-05-28T03:06:32.000Z","type":"response_item","payload":{"type":"function_call_output","call_id":"call_wait","output":"{\"status\":{\"agent_1\":{\"completed\":\"done\"}},\"timed_out\":false}"}}`,
	)

	hub := NewHub()
	req := httptest.NewRequest(http.MethodPost, "/api/import", strings.NewReader(`{"path":`+quoteJSON(path)+`}`))
	res := httptest.NewRecorder()
	NewHandler(hub).ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", res.Code, res.Body.String())
	}
	events := hub.Events()
	if len(events) < 4 {
		t.Fatalf("events = %#v", events)
	}
	if hub.Session().ID != "codex_019e6ca8_742f_7283_b139_593c75443b37" || hub.Session().Provider != "codex" {
		t.Fatalf("session = %#v", hub.Session())
	}
	assertHasEvent(t, events, trace.EventUserMessage, "hello codex")
	assertHasEvent(t, events, trace.EventSubagent, "spawn Archimedes")
	assertHasEvent(t, events, trace.EventSubagent, "Archimedes completed")
}

func TestImportEndpointLoadsCodexRolloutDirectoryBundle(t *testing.T) {
	dir := t.TempDir()
	writeRawLines(t, filepath.Join(dir, "rollout-a.jsonl"),
		`{"timestamp":"2026-05-28T03:06:20.000Z","type":"session_meta","payload":{"id":"codex-a","timestamp":"2026-05-28T03:06:20.000Z","cwd":"/tmp/project","model_provider":"openai"}}`,
		`{"timestamp":"2026-05-28T03:06:21.000Z","type":"event_msg","payload":{"type":"user_message","message":"first rollout"}}`,
	)
	writeRawLines(t, filepath.Join(dir, "rollout-b.jsonl"),
		`{"timestamp":"2026-05-28T03:07:20.000Z","type":"session_meta","payload":{"id":"codex-b","timestamp":"2026-05-28T03:07:20.000Z","cwd":"/tmp/project","model_provider":"openai"}}`,
		`{"timestamp":"2026-05-28T03:07:21.000Z","type":"event_msg","payload":{"type":"user_message","message":"second rollout"}}`,
	)

	hub := NewHub()
	req := httptest.NewRequest(http.MethodPost, "/api/import", strings.NewReader(`{"path":`+quoteJSON(dir)+`}`))
	res := httptest.NewRecorder()
	NewHandler(hub).ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", res.Code, res.Body.String())
	}
	var imported SessionImportResponse
	if err := json.Unmarshal(res.Body.Bytes(), &imported); err != nil {
		t.Fatal(err)
	}
	if imported.Source != "codex_bundle" || imported.Session.Provider != "codex" {
		t.Fatalf("imported = %#v", imported)
	}
	events := hub.Events()
	assertHasEvent(t, events, trace.EventUserMessage, "first rollout")
	assertHasEvent(t, events, trace.EventUserMessage, "second rollout")
	for _, event := range events {
		if event.SessionID != imported.Session.ID {
			t.Fatalf("event session id = %q, want %q", event.SessionID, imported.Session.ID)
		}
		if event.Summary["sourceSessionId"] == nil {
			t.Fatalf("event missing sourceSessionId: %#v", event)
		}
	}
}

func TestImportEndpointLoadsClaudeSessionBundle(t *testing.T) {
	root := t.TempDir()
	sessionID := "e2bbcb5c-985c-475e-a810-961d7100fe97"
	mainPath := filepath.Join(root, sessionID+".jsonl")
	sessionDir := filepath.Join(root, sessionID)
	subagentsDir := filepath.Join(sessionDir, "subagents")
	if err := os.MkdirAll(subagentsDir, 0o700); err != nil {
		t.Fatal(err)
	}
	writeRawLines(t, mainPath,
		`{"type":"user","uuid":"main_user","timestamp":"2026-05-27T06:39:10Z","sessionId":"`+sessionID+`","message":{"role":"user","content":"main request"}}`,
		`{"type":"assistant","uuid":"main_agent_start","timestamp":"2026-05-27T06:39:11Z","sessionId":"`+sessionID+`","message":{"role":"assistant","content":[{"type":"tool_use","id":"toolu_agent","name":"Agent","input":{"description":"Check docs","subagent_type":"coder","prompt":"go"}}]}}`,
		`{"type":"user","uuid":"main_agent_result","timestamp":"2026-05-27T06:39:20Z","sessionId":"`+sessionID+`","message":{"role":"user","content":[{"tool_use_id":"toolu_agent","type":"tool_result","content":[{"type":"text","text":"done"}]}]},"toolUseResult":{"status":"completed","agentId":"agent_a","agentType":"coder","totalDurationMs":9000,"totalTokens":10,"totalToolUseCount":1,"content":[{"type":"text","text":"done"}]}}`,
	)
	subagentPath := filepath.Join(subagentsDir, "agent-agent_a.jsonl")
	writeRawLines(t, subagentPath,
		`{"type":"user","uuid":"sub_user","timestamp":"2026-05-27T06:39:12Z","sessionId":"`+sessionID+`","agentId":"agent_a","isSidechain":true,"message":{"role":"user","content":"sub task"}}`,
		`{"type":"assistant","uuid":"sub_tool","timestamp":"2026-05-27T06:39:13Z","sessionId":"`+sessionID+`","agentId":"agent_a","isSidechain":true,"message":{"role":"assistant","content":[{"type":"tool_use","id":"toolu_read","name":"Read","input":{"file_path":"/tmp/a.md"}}]}}`,
		`{"type":"user","uuid":"sub_result","timestamp":"2026-05-27T06:39:14Z","sessionId":"`+sessionID+`","agentId":"agent_a","isSidechain":true,"message":{"role":"user","content":[{"tool_use_id":"toolu_read","type":"tool_result","content":[{"type":"text","text":"file contents"}]}]}}`,
	)
	writeRawLines(t, filepath.Join(subagentsDir, "agent-agent_a.meta.json"),
		`{"agentType":"coder","description":"Check docs","toolUseId":"toolu_agent"}`,
	)

	hub := NewHub()
	req := httptest.NewRequest(http.MethodPost, "/api/import", strings.NewReader(`{"path":`+quoteJSON(sessionDir)+`}`))
	res := httptest.NewRecorder()
	NewHandler(hub).ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", res.Code, res.Body.String())
	}
	var imported SessionImportResponse
	if err := json.Unmarshal(res.Body.Bytes(), &imported); err != nil {
		t.Fatal(err)
	}
	if imported.Source != "claude_bundle" || imported.Session.Provider != "claude" || imported.Session.ID != "claude_"+sanitizeSessionID(sessionID) {
		t.Fatalf("imported = %#v", imported)
	}
	events := hub.Events()
	assertHasEvent(t, events, trace.EventUserMessage, "main request")
	assertHasEvent(t, events, trace.EventUserMessage, "sub task")
	assertHasEvent(t, events, trace.EventSubagent, "coder: Check docs")
	assertHasEvent(t, events, trace.EventSubagent, "coder result")
	assertHasEvent(t, events, trace.EventToolCall, "Read")
	assertHasEvent(t, events, trace.EventToolResult, "tool result")
	subTask := findEvent(events, trace.EventUserMessage, "sub task")
	if subTask == nil || !containsString(subTask.CorrelationIDs, "agent_a") || !containsString(subTask.CorrelationIDs, "toolu_agent") {
		t.Fatalf("sub task correlations = %#v", subTask)
	}
}

func TestImportUploadEndpointParsesCodexRolloutJSONL(t *testing.T) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", "rollout.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	lines := []string{
		`{"timestamp":"2026-05-28T03:06:27.362Z","type":"session_meta","payload":{"id":"019e6ca8-742f-7283-b139-593c75443b37","timestamp":"2026-05-28T03:06:20.393Z","cwd":"/tmp/project","model_provider":"openai"}}`,
		`{"timestamp":"2026-05-28T03:06:28.000Z","type":"event_msg","payload":{"type":"user_message","message":"uploaded rollout"}}`,
	}
	if _, err := part.Write([]byte(strings.Join(lines, "\n") + "\n")); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	hub := NewHub()
	req := httptest.NewRequest(http.MethodPost, "/api/import/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	res := httptest.NewRecorder()
	NewHandler(hub).ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", res.Code, res.Body.String())
	}
	var imported SessionImportResponse
	if err := json.Unmarshal(res.Body.Bytes(), &imported); err != nil {
		t.Fatal(err)
	}
	if imported.Source != "codex_rollout" || imported.Session.Command[0] != "uploaded:rollout.jsonl" {
		t.Fatalf("imported = %#v", imported)
	}
	assertHasEvent(t, hub.Events(), trace.EventUserMessage, "uploaded rollout")
}

func TestImportUploadEndpointParsesMultipleFilesAsBundle(t *testing.T) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	files := map[string][]string{
		"rollout-a.jsonl": {
			`{"timestamp":"2026-05-28T03:06:20.000Z","type":"session_meta","payload":{"id":"upload-a","timestamp":"2026-05-28T03:06:20.000Z","cwd":"/tmp/project","model_provider":"openai"}}`,
			`{"timestamp":"2026-05-28T03:06:21.000Z","type":"event_msg","payload":{"type":"user_message","message":"uploaded first"}}`,
		},
		"rollout-b.jsonl": {
			`{"timestamp":"2026-05-28T03:07:20.000Z","type":"session_meta","payload":{"id":"upload-b","timestamp":"2026-05-28T03:07:20.000Z","cwd":"/tmp/project","model_provider":"openai"}}`,
			`{"timestamp":"2026-05-28T03:07:21.000Z","type":"event_msg","payload":{"type":"user_message","message":"uploaded second"}}`,
		},
	}
	for name, lines := range files {
		part, err := writer.CreateFormFile("file", name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := part.Write([]byte(strings.Join(lines, "\n") + "\n")); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	hub := NewHub()
	req := httptest.NewRequest(http.MethodPost, "/api/import/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	res := httptest.NewRecorder()
	NewHandler(hub).ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", res.Code, res.Body.String())
	}
	var imported SessionImportResponse
	if err := json.Unmarshal(res.Body.Bytes(), &imported); err != nil {
		t.Fatal(err)
	}
	if imported.Source != "codex_bundle" || len(imported.Session.Command) != 1 || imported.Session.Command[0] != "uploaded:2 files" {
		t.Fatalf("imported = %#v", imported)
	}
	assertHasEvent(t, hub.Events(), trace.EventUserMessage, "uploaded first")
	assertHasEvent(t, hub.Events(), trace.EventUserMessage, "uploaded second")
}

func TestImportEndpointParsesCodexSkillEvents(t *testing.T) {
	path := filepath.Join(t.TempDir(), "rollout.jsonl")
	skillPath := "/tmp/codex/skills/browser/SKILL.md"
	writeJSONLines(t, path,
		map[string]any{
			"timestamp": "2026-05-28T03:06:27.362Z",
			"type":      "session_meta",
			"payload": map[string]any{
				"id":             "019e6ca8-742f-7283-b139-593c75443b37",
				"timestamp":      "2026-05-28T03:06:20.393Z",
				"cwd":            "/tmp/project",
				"model_provider": "openai",
			},
		},
		map[string]any{
			"timestamp": "2026-05-28T03:06:28.000Z",
			"type":      "response_item",
			"payload": map[string]any{
				"type": "message",
				"role": "developer",
				"content": []map[string]any{{
					"type": "input_text",
					"text": "<skills_instructions>\n### Available skills\n- browser:browser: Browser automation for local app testing. (file: " + skillPath + ")\n- skill-creator: Guide for creating skills. (file: /tmp/codex/skills/skill-creator/SKILL.md)\n</skills_instructions>",
				}},
			},
		},
		map[string]any{
			"timestamp": "2026-05-28T03:06:29.000Z",
			"type":      "response_item",
			"payload": map[string]any{
				"type":      "function_call",
				"name":      "exec_command",
				"call_id":   "call_skill",
				"arguments": `{"cmd":"cat ` + skillPath + `"}`,
			},
		},
	)

	hub := NewHub()
	req := httptest.NewRequest(http.MethodPost, "/api/import", strings.NewReader(`{"path":`+quoteJSON(path)+`}`))
	res := httptest.NewRecorder()
	NewHandler(hub).ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", res.Code, res.Body.String())
	}
	events := hub.Events()
	available := findSkillEvent(events, "browser:browser", "available")
	if available == nil {
		t.Fatalf("missing available skill event in %#v", events)
	}
	if available.Summary["description"] != "Browser automation for local app testing." || available.Summary["path"] != skillPath {
		t.Fatalf("available summary = %#v", available.Summary)
	}
	invoked := findSkillEvent(events, "browser:browser invoked", "invoked")
	if invoked == nil {
		t.Fatalf("missing invoked skill event in %#v", events)
	}
	if invoked.Summary["trigger"] != "skill_document_read" {
		t.Fatalf("invoked summary = %#v", invoked.Summary)
	}
	if !containsString(invoked.CorrelationIDs, "skill:browser:browser") {
		t.Fatalf("invoked correlation ids = %#v", invoked.CorrelationIDs)
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

func indexBody(t *testing.T) string {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	res := httptest.NewRecorder()
	NewHandler(NewHub()).ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("status = %d", res.Code)
	}
	return res.Body.String()
}

func writeJSONLines(t *testing.T, path string, values ...any) {
	t.Helper()
	var lines []string
	for _, value := range values {
		data, err := json.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}
		lines = append(lines, string(data))
	}
	writeRawLines(t, path, lines...)
}

func writeRawLines(t *testing.T, path string, lines ...string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
}

func quoteJSON(value string) string {
	data, _ := json.Marshal(value)
	return string(data)
}

func assertHasEvent(t *testing.T, events []trace.Event, eventType trace.EventType, title string) {
	t.Helper()
	for _, event := range events {
		if event.Type == eventType && event.Title == title {
			return
		}
	}
	t.Fatalf("missing %s %q in %#v", eventType, title, events)
}

func findEvent(events []trace.Event, eventType trace.EventType, title string) *trace.Event {
	for i := range events {
		if events[i].Type == eventType && events[i].Title == title {
			return &events[i]
		}
	}
	return nil
}

func findSkillEvent(events []trace.Event, title, phase string) *trace.Event {
	for i := range events {
		event := &events[i]
		if event.Type == trace.EventSkill && event.Title == title && event.Summary["phase"] == phase {
			return event
		}
	}
	return nil
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
