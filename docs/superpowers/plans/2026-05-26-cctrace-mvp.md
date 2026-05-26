# cctrace MVP Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the first working `cctrace` MVP: a Go CLI that wraps Claude Code/Codex commands, emits normalized trace events, serves a local waterfall UI, and can replay saved sessions.

**Architecture:** Use a small Go core with clear packages for trace models, persistence, collectors, correlation, server/UI, and CLI orchestration. The MVP starts with deterministic mock/file collectors and process events, then adds ccglass/transcript parsing hooks so the UI and event pipeline are testable before real integration becomes complete.

**Tech Stack:** Go 1.22+, standard library only for the MVP backend; vanilla HTML/CSS/JavaScript embedded in Go for the UI; `go test ./...` for tests.

---

## File Structure

Create these files:

- `go.mod` — Go module definition.
- `cmd/cctrace/main.go` — CLI entry point.
- `internal/trace/model.go` — shared event/session types and constants.
- `internal/trace/model_test.go` — model JSON and duration tests.
- `internal/store/store.go` — JSONL session persistence.
- `internal/store/store_test.go` — persistence tests.
- `internal/collectors/process.go` — child process wrapper collector.
- `internal/collectors/process_test.go` — process collector tests.
- `internal/collectors/transcript.go` — Claude/Codex JSONL transcript parser.
- `internal/collectors/transcript_test.go` — transcript parser tests.
- `internal/collectors/ccglass.go` — ccglass JSON/HAR-ish request parser for exported/session fragments.
- `internal/collectors/ccglass_test.go` — ccglass parser tests.
- `internal/correlate/correlate.go` — event correlation engine.
- `internal/correlate/correlate_test.go` — correlation tests.
- `internal/server/server.go` — HTTP server, SSE stream, static UI, replay endpoint.
- `internal/server/server_test.go` — server API tests.
- `internal/app/app.go` — high-level run/view orchestration.
- `internal/app/app_test.go` — orchestration tests using fake commands.
- `web/index.html` — embedded waterfall UI.
- `README.md` — MVP usage.
- Modify `CLAUDE.md` — add actual build/test/run commands after implementation.

Do not add Node, React, or frontend build tooling in the MVP. The UI should be simple and shippable with `go run ./cmd/cctrace`.

---

### Task 1: Scaffold Go Module and Trace Model

**Files:**
- Create: `go.mod`
- Create: `internal/trace/model.go`
- Create: `internal/trace/model_test.go`

- [ ] **Step 1: Write the failing model tests**

Create `internal/trace/model_test.go`:

```go
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
```

- [ ] **Step 2: Run test to verify it fails**

Run:

```bash
go test ./internal/trace
```

Expected: FAIL because `go.mod` and `internal/trace` do not exist yet.

- [ ] **Step 3: Add module and model implementation**

Create `go.mod`:

```go
module github.com/agentz/cctrace

go 1.22
```

Create `internal/trace/model.go`:

```go
package trace

import "time"

type EventType string

const (
	EventProcess     EventType = "process"
	EventUserMessage EventType = "user_message"
	EventAgentTurn   EventType = "agent_turn"
	EventAPIRequest  EventType = "api_request"
	EventAPIResponse EventType = "api_response"
	EventToolCall    EventType = "tool_call"
	EventToolResult  EventType = "tool_result"
	EventSkill       EventType = "skill"
	EventSubagent    EventType = "subagent"
	EventPermission  EventType = "permission"
	EventHook        EventType = "hook"
	EventError       EventType = "error"
)

type Status string

const (
	StatusRunning Status = "running"
	StatusOK      Status = "ok"
	StatusError   Status = "error"
	StatusBlocked Status = "blocked"
	StatusUnknown Status = "unknown"
)

type Source string

const (
	SourceCcglass          Source = "ccglass"
	SourceClaudeTranscript Source = "claude_transcript"
	SourceCodexLog         Source = "codex_log"
	SourceProcess          Source = "process"
	SourceDerived          Source = "derived"
)

type Confidence string

const (
	ConfidenceExact   Confidence = "exact"
	ConfidenceLikely  Confidence = "likely"
	ConfidencePossible Confidence = "possible"
	ConfidenceUnknown Confidence = "unknown"
)

type RawRef struct {
	File      string `json:"file,omitempty"`
	Offset    int64  `json:"offset,omitempty"`
	RequestID string `json:"requestId,omitempty"`
	BlobHash  string `json:"blobHash,omitempty"`
}

type Event struct {
	ID             string         `json:"id"`
	SessionID      string         `json:"sessionId"`
	Type           EventType      `json:"type"`
	Title          string         `json:"title"`
	Timestamp      time.Time      `json:"timestamp"`
	EndTimestamp   *time.Time     `json:"endTimestamp,omitempty"`
	DurationMS     *int64         `json:"durationMs,omitempty"`
	Status         Status         `json:"status"`
	Source         Source         `json:"source"`
	ParentID       string         `json:"parentId,omitempty"`
	CorrelationIDs []string       `json:"correlationIds"`
	Confidence     Confidence     `json:"confidence"`
	Summary        map[string]any `json:"summary"`
	RawRef         *RawRef        `json:"rawRef,omitempty"`
}

type Session struct {
	ID        string    `json:"id"`
	Command   []string  `json:"command"`
	StartedAt time.Time `json:"startedAt"`
	EndedAt   time.Time `json:"endedAt,omitempty"`
}

func (e Event) Duration() time.Duration {
	if e.EndTimestamp != nil {
		return e.EndTimestamp.Sub(e.Timestamp)
	}
	if e.DurationMS != nil {
		return time.Duration(*e.DurationMS) * time.Millisecond
	}
	return 0
}
```

- [ ] **Step 4: Run test to verify it passes**

Run:

```bash
go test ./internal/trace
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add go.mod internal/trace/model.go internal/trace/model_test.go
git commit -m "feat: add trace event model"
```

---

### Task 2: Add JSONL Session Store

**Files:**
- Create: `internal/store/store.go`
- Create: `internal/store/store_test.go`

- [ ] **Step 1: Write failing store tests**

Create `internal/store/store_test.go`:

```go
package store

import (
	"testing"
	"time"

	"github.com/agentz/cctrace/internal/trace"
)

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
```

- [ ] **Step 2: Run test to verify it fails**

Run:

```bash
go test ./internal/store
```

Expected: FAIL because package `internal/store` does not exist.

- [ ] **Step 3: Implement JSONL store**

Create `internal/store/store.go`:

```go
package store

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/agentz/cctrace/internal/trace"
)

type Store struct{ root string }

func New(root string) *Store { return &Store{root: root} }

func (s *Store) sessionDir(id string) string { return filepath.Join(s.root, id) }

func (s *Store) CreateSession(session trace.Session) error {
	dir := s.sessionDir(session.ID)
	if err := os.MkdirAll(dir, 0o755); err != nil { return err }
	data, err := json.MarshalIndent(session, "", "  ")
	if err != nil { return err }
	return os.WriteFile(filepath.Join(dir, "session.json"), data, 0o644)
}

func (s *Store) AppendEvent(event trace.Event) error {
	dir := s.sessionDir(event.SessionID)
	if err := os.MkdirAll(dir, 0o755); err != nil { return err }
	file, err := os.OpenFile(filepath.Join(dir, "events.jsonl"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil { return err }
	defer file.Close()
	data, err := json.Marshal(event)
	if err != nil { return err }
	if _, err := file.Write(append(data, '\n')); err != nil { return err }
	return nil
}

func (s *Store) ReadEvents(sessionID string) ([]trace.Event, error) {
	file, err := os.Open(filepath.Join(s.sessionDir(sessionID), "events.jsonl"))
	if err != nil { return nil, err }
	defer file.Close()

	var events []trace.Event
	scanner := bufio.NewScanner(file)
	line := 0
	for scanner.Scan() {
		line++
		var event trace.Event
		if err := json.Unmarshal(scanner.Bytes(), &event); err != nil {
			return nil, fmt.Errorf("events.jsonl line %d: %w", line, err)
		}
		events = append(events, event)
	}
	return events, scanner.Err()
}
```

- [ ] **Step 4: Run test to verify it passes**

Run:

```bash
go test ./internal/store
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/store/store.go internal/store/store_test.go
git commit -m "feat: add trace session store"
```

---

### Task 3: Add Process Collector

**Files:**
- Create: `internal/collectors/process.go`
- Create: `internal/collectors/process_test.go`

- [ ] **Step 1: Write failing process collector test**

Create `internal/collectors/process_test.go`:

```go
package collectors

import (
	"context"
	"testing"

	"github.com/agentz/cctrace/internal/trace"
)

func TestRunProcessEmitsStartAndExit(t *testing.T) {
	collector := ProcessCollector{SessionID: "sess_1"}
	events, exitCode, err := collector.Run(context.Background(), []string{"sh", "-c", "printf hello"})
	if err != nil { t.Fatal(err) }
	if exitCode != 0 { t.Fatalf("exitCode = %d, want 0", exitCode) }
	if len(events) != 2 { t.Fatalf("len(events) = %d, want 2", len(events)) }
	if events[0].Type != trace.EventProcess || events[0].Status != trace.StatusRunning {
		t.Fatalf("start event = %#v", events[0])
	}
	if events[1].Type != trace.EventProcess || events[1].Status != trace.StatusOK {
		t.Fatalf("exit event = %#v", events[1])
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run:

```bash
go test ./internal/collectors -run TestRunProcessEmitsStartAndExit
```

Expected: FAIL because `ProcessCollector` is undefined.

- [ ] **Step 3: Implement process collector**

Create `internal/collectors/process.go`:

```go
package collectors

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/agentz/cctrace/internal/trace"
)

type ProcessCollector struct{ SessionID string }

func (c ProcessCollector) Run(ctx context.Context, command []string) ([]trace.Event, int, error) {
	if len(command) == 0 { return nil, -1, fmt.Errorf("missing command") }
	started := time.Now()
	startEvent := trace.Event{
		ID: "process_start_" + started.Format("150405.000000000"), SessionID: c.SessionID,
		Type: trace.EventProcess, Title: "process started", Timestamp: started, Status: trace.StatusRunning,
		Source: trace.SourceProcess, CorrelationIDs: []string{}, Confidence: trace.ConfidenceExact,
		Summary: map[string]any{"command": command},
	}

	cmd := exec.CommandContext(ctx, command[0], command[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	err := cmd.Run()
	ended := time.Now()
	exitCode := 0
	status := trace.StatusOK
	if err != nil {
		status = trace.StatusError
		exitCode = 1
		if exitErr, ok := err.(*exec.ExitError); ok { exitCode = exitErr.ExitCode() }
	}
	duration := ended.Sub(started).Milliseconds()
	exitEvent := trace.Event{
		ID: "process_exit_" + ended.Format("150405.000000000"), SessionID: c.SessionID,
		Type: trace.EventProcess, Title: "process exited", Timestamp: ended, DurationMS: &duration, Status: status,
		Source: trace.SourceProcess, CorrelationIDs: []string{}, Confidence: trace.ConfidenceExact,
		Summary: map[string]any{"exitCode": float64(exitCode)},
	}
	return []trace.Event{startEvent, exitEvent}, exitCode, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run:

```bash
go test ./internal/collectors -run TestRunProcessEmitsStartAndExit
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/collectors/process.go internal/collectors/process_test.go
git commit -m "feat: collect wrapped process events"
```

---

### Task 4: Add Transcript and ccglass Parsers

**Files:**
- Create: `internal/collectors/transcript.go`
- Create: `internal/collectors/transcript_test.go`
- Create: `internal/collectors/ccglass.go`
- Create: `internal/collectors/ccglass_test.go`

- [ ] **Step 1: Write failing parser tests**

Create `internal/collectors/transcript_test.go`:

```go
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
	if err != nil { t.Fatal(err) }
	if len(events) != 2 { t.Fatalf("len(events) = %d, want 2", len(events)) }
	if events[0].Type != trace.EventToolCall || events[0].Title != "Bash" { t.Fatalf("event[0] = %#v", events[0]) }
	if events[1].Type != trace.EventToolResult || events[1].Status != trace.StatusOK { t.Fatalf("event[1] = %#v", events[1]) }
}
```

Create `internal/collectors/ccglass_test.go`:

```go
package collectors

import (
	"strings"
	"testing"

	"github.com/agentz/cctrace/internal/trace"
)

func TestParseCcglassRequests(t *testing.T) {
	input := strings.NewReader(`{"requestId":"req_1","startedAt":"2026-05-26T10:00:00Z","endedAt":"2026-05-26T10:00:02Z","model":"claude-opus-4-7","usage":{"input_tokens":10,"output_tokens":20}}`)
	events, err := ParseCcglassJSON("sess_1", "request.json", input)
	if err != nil { t.Fatal(err) }
	if len(events) != 2 { t.Fatalf("len(events) = %d, want 2", len(events)) }
	if events[0].Type != trace.EventAPIRequest || events[0].RawRef.RequestID != "req_1" { t.Fatalf("request event = %#v", events[0]) }
	if events[1].Type != trace.EventAPIResponse || events[1].Status != trace.StatusOK { t.Fatalf("response event = %#v", events[1]) }
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run:

```bash
go test ./internal/collectors -run 'TestParse(Transcript|Ccglass)'
```

Expected: FAIL because parser functions are undefined.

- [ ] **Step 3: Implement transcript parser**

Create `internal/collectors/transcript.go`:

```go
package collectors

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/agentz/cctrace/internal/trace"
)

type transcriptRecord struct {
	Type      string `json:"type"`
	Tool      string `json:"tool"`
	ID        string `json:"id"`
	Status    string `json:"status"`
	Timestamp string `json:"timestamp"`
}

func ParseTranscriptJSONL(sessionID, provider, file string, r io.Reader) ([]trace.Event, error) {
	source := trace.SourceClaudeTranscript
	if provider == "codex" { source = trace.SourceCodexLog }
	var events []trace.Event
	scanner := bufio.NewScanner(r)
	var offset int64
	for scanner.Scan() {
		line := scanner.Text()
		var rec transcriptRecord
		if err := json.Unmarshal([]byte(line), &rec); err != nil {
			events = append(events, parseError(sessionID, source, file, offset, err))
			offset += int64(len(line) + 1)
			continue
		}
		ts, err := time.Parse(time.RFC3339, rec.Timestamp)
		if err != nil { ts = time.Now() }
		eventType := transcriptEventType(rec.Type)
		status := transcriptStatus(rec.Status)
		title := rec.Tool
		if title == "" { title = rec.Type }
		events = append(events, trace.Event{
			ID: stableID(rec.Type, rec.ID, ts), SessionID: sessionID, Type: eventType, Title: title, Timestamp: ts,
			Status: status, Source: source, CorrelationIDs: []string{rec.ID}, Confidence: trace.ConfidenceExact,
			Summary: map[string]any{"provider": provider, "recordType": rec.Type, "tool": rec.Tool},
			RawRef: &trace.RawRef{File: file, Offset: offset},
		})
		offset += int64(len(line) + 1)
	}
	return events, scanner.Err()
}

func transcriptEventType(kind string) trace.EventType {
	switch kind {
	case "tool_call": return trace.EventToolCall
	case "tool_result": return trace.EventToolResult
	case "permission": return trace.EventPermission
	case "hook": return trace.EventHook
	case "skill": return trace.EventSkill
	case "subagent": return trace.EventSubagent
	case "user_message": return trace.EventUserMessage
	case "agent_turn": return trace.EventAgentTurn
	default: return trace.EventAgentTurn
	}
}

func transcriptStatus(status string) trace.Status {
	switch status {
	case "ok", "success": return trace.StatusOK
	case "error", "failed": return trace.StatusError
	case "blocked": return trace.StatusBlocked
	case "running": return trace.StatusRunning
	default: return trace.StatusUnknown
	}
}

func parseError(sessionID string, source trace.Source, file string, offset int64, err error) trace.Event {
	now := time.Now()
	return trace.Event{ID: stableID("parse_error", fmt.Sprint(offset), now), SessionID: sessionID, Type: trace.EventError, Title: "parse error", Timestamp: now, Status: trace.StatusError, Source: source, CorrelationIDs: []string{}, Confidence: trace.ConfidenceExact, Summary: map[string]any{"error": err.Error()}, RawRef: &trace.RawRef{File: file, Offset: offset}}
}

func stableID(parts ...any) string {
	items := make([]string, len(parts))
	for i, part := range parts { items[i] = fmt.Sprint(part) }
	return strings.NewReplacer(":", "_", ".", "_", " ", "_").Replace(strings.Join(items, "_"))
}
```

- [ ] **Step 4: Implement ccglass parser**

Create `internal/collectors/ccglass.go`:

```go
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
	if err := json.NewDecoder(r).Decode(&rec); err != nil { return nil, err }
	started, err := time.Parse(time.RFC3339, rec.StartedAt)
	if err != nil { started = time.Now() }
	ended, err := time.Parse(time.RFC3339, rec.EndedAt)
	if err != nil { ended = started }
	duration := ended.Sub(started).Milliseconds()
	reqID := rec.RequestID
	if reqID == "" { reqID = stableID("ccglass", started) }
	request := trace.Event{ID: stableID("api_request", reqID), SessionID: sessionID, Type: trace.EventAPIRequest, Title: rec.Model, Timestamp: started, Status: trace.StatusRunning, Source: trace.SourceCcglass, CorrelationIDs: []string{reqID}, Confidence: trace.ConfidenceExact, Summary: map[string]any{"model": rec.Model}, RawRef: &trace.RawRef{File: file, RequestID: reqID}}
	response := trace.Event{ID: stableID("api_response", reqID), SessionID: sessionID, Type: trace.EventAPIResponse, Title: rec.Model, Timestamp: ended, DurationMS: &duration, Status: trace.StatusOK, Source: trace.SourceCcglass, CorrelationIDs: []string{reqID}, Confidence: trace.ConfidenceExact, Summary: map[string]any{"model": rec.Model, "usage": rec.Usage}, RawRef: &trace.RawRef{File: file, RequestID: reqID}}
	return []trace.Event{request, response}, nil
}
```

- [ ] **Step 5: Run tests to verify they pass**

Run:

```bash
go test ./internal/collectors
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/collectors/transcript.go internal/collectors/transcript_test.go internal/collectors/ccglass.go internal/collectors/ccglass_test.go
git commit -m "feat: parse transcript and ccglass events"
```

---

### Task 5: Add Correlation Engine

**Files:**
- Create: `internal/correlate/correlate.go`
- Create: `internal/correlate/correlate_test.go`

- [ ] **Step 1: Write failing correlation tests**

Create `internal/correlate/correlate_test.go`:

```go
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
	if got[1].ParentID != "tool_1" { t.Fatalf("ParentID = %q, want tool_1", got[1].ParentID) }
	if got[1].Confidence != trace.ConfidenceLikely { t.Fatalf("Confidence = %q, want likely", got[1].Confidence) }
}

func TestCorrelateLeavesFarAPIOrphan(t *testing.T) {
	base := time.UnixMilli(1000)
	events := []trace.Event{
		{ID: "tool_1", SessionID: "sess_1", Type: trace.EventToolCall, Timestamp: base, CorrelationIDs: []string{}, Confidence: trace.ConfidenceExact, Summary: map[string]any{}},
		{ID: "api_1", SessionID: "sess_1", Type: trace.EventAPIRequest, Timestamp: base.Add(10 * time.Second), CorrelationIDs: []string{}, Confidence: trace.ConfidenceExact, Summary: map[string]any{}},
	}
	got := Correlate(events, 2*time.Second)
	if got[1].ParentID != "" { t.Fatalf("ParentID = %q, want empty", got[1].ParentID) }
	if got[1].Confidence != trace.ConfidenceUnknown { t.Fatalf("Confidence = %q, want unknown", got[1].Confidence) }
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run:

```bash
go test ./internal/correlate
```

Expected: FAIL because `Correlate` is undefined.

- [ ] **Step 3: Implement correlation**

Create `internal/correlate/correlate.go`:

```go
package correlate

import (
	"slices"
	"time"

	"github.com/agentz/cctrace/internal/trace"
)

func Correlate(events []trace.Event, window time.Duration) []trace.Event {
	out := append([]trace.Event(nil), events...)
	slices.SortFunc(out, func(a, b trace.Event) int { return a.Timestamp.Compare(b.Timestamp) })
	for i := range out {
		if out[i].Type != trace.EventAPIRequest && out[i].Type != trace.EventAPIResponse { continue }
		parent := nearestParent(out, i, window)
		if parent == "" {
			out[i].Confidence = trace.ConfidenceUnknown
			continue
		}
		out[i].ParentID = parent
		out[i].Confidence = trace.ConfidenceLikely
	}
	return out
}

func nearestParent(events []trace.Event, index int, window time.Duration) string {
	candidateTypes := map[trace.EventType]bool{trace.EventToolCall: true, trace.EventAgentTurn: true, trace.EventSkill: true, trace.EventSubagent: true}
	var bestID string
	bestDelta := window + time.Nanosecond
	for i := range events {
		if i == index || !candidateTypes[events[i].Type] { continue }
		delta := events[index].Timestamp.Sub(events[i].Timestamp)
		if delta < 0 { delta = -delta }
		if delta <= window && delta < bestDelta {
			bestDelta = delta
			bestID = events[i].ID
		}
	}
	return bestID
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run:

```bash
go test ./internal/correlate
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/correlate/correlate.go internal/correlate/correlate_test.go
git commit -m "feat: correlate trace events"
```

---

### Task 6: Add HTTP Server and Waterfall UI

**Files:**
- Create: `web/index.html`
- Create: `internal/server/server.go`
- Create: `internal/server/server_test.go`

- [ ] **Step 1: Write failing server test**

Create `internal/server/server_test.go`:

```go
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
	if res.Code != http.StatusOK { t.Fatalf("status = %d", res.Code) }
	var events []trace.Event
	if err := json.Unmarshal(res.Body.Bytes(), &events); err != nil { t.Fatal(err) }
	if len(events) != 1 || events[0].ID != "evt_1" { t.Fatalf("events = %#v", events) }
}
```

- [ ] **Step 2: Run test to verify it fails**

Run:

```bash
go test ./internal/server
```

Expected: FAIL because server package does not exist.

- [ ] **Step 3: Implement server**

Create `internal/server/server.go`:

```go
package server

import (
	"embed"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"github.com/agentz/cctrace/internal/trace"
)

//go:embed ../../web/index.html
var embedded embed.FS

type Hub struct {
	mu     sync.Mutex
	events []trace.Event
}

func NewHub() *Hub { return &Hub{} }

func (h *Hub) Publish(event trace.Event) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.events = append(h.events, event)
}

func (h *Hub) Events() []trace.Event {
	h.mu.Lock()
	defer h.mu.Unlock()
	return append([]trace.Event(nil), h.events...)
}

func NewHandler(hub *Hub) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		data, err := embedded.ReadFile("../../web/index.html")
		if err != nil { http.Error(w, err.Error(), http.StatusInternalServerError); return }
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(data)
	})
	mux.HandleFunc("/api/events", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(hub.Events())
	})
	return mux
}

func Listen(addr string, hub *Hub) error {
	fmt.Printf("cctrace UI: http://%s\n", addr)
	return http.ListenAndServe(addr, NewHandler(hub))
}
```

If `go test` reports that `//go:embed ../../web/index.html` is invalid because embed patterns cannot contain `..`, replace the embed approach in this task with a `uiHTML` string constant in `server.go`, and keep `web/index.html` as the source file copied into that string. The passing state for MVP is a served `/` page and `/api/events` JSON.

- [ ] **Step 4: Add simple UI**

Create `web/index.html`:

```html
<!doctype html>
<html>
<head>
  <meta charset="utf-8">
  <title>cctrace</title>
  <style>
    body { margin: 0; font-family: ui-sans-serif, system-ui; background: #0f172a; color: #e2e8f0; }
    header { padding: 16px 20px; border-bottom: 1px solid #334155; }
    main { display: grid; grid-template-columns: 260px 1fr 320px; height: calc(100vh - 57px); }
    aside, section { padding: 16px; overflow: auto; border-right: 1px solid #334155; }
    .event { margin: 8px 0; padding: 8px; border-radius: 8px; background: #1e293b; cursor: pointer; }
    .bar { margin: 8px 0; padding: 8px; border-radius: 8px; background: #2563eb; min-width: 120px; }
    .api_request, .api_response { background: #16a34a; }
    .tool_call, .tool_result { background: #ca8a04; }
    .error { background: #dc2626; }
    pre { white-space: pre-wrap; word-break: break-word; }
  </style>
</head>
<body>
<header><strong>cctrace</strong> real-time agent profiler</header>
<main>
  <aside><h3>Trace Tree</h3><div id="tree"></div></aside>
  <section><h3>Waterfall Timeline</h3><div id="timeline"></div></section>
  <section><h3>Details</h3><pre id="details">Select an event</pre></section>
</main>
<script>
async function loadEvents() {
  const res = await fetch('/api/events');
  const events = await res.json();
  const tree = document.getElementById('tree');
  const timeline = document.getElementById('timeline');
  tree.innerHTML = '';
  timeline.innerHTML = '';
  events.forEach((event) => {
    const item = document.createElement('div');
    item.className = 'event';
    item.textContent = `${event.type}: ${event.title}`;
    item.onclick = () => show(event);
    tree.appendChild(item);
    const bar = document.createElement('div');
    bar.className = `bar ${event.type}`;
    const width = Math.max(120, Number(event.durationMs || 100));
    bar.style.width = Math.min(width, 900) + 'px';
    bar.textContent = `${event.title} · ${event.status} · ${event.confidence}`;
    bar.onclick = () => show(event);
    timeline.appendChild(bar);
  });
}
function show(event) { document.getElementById('details').textContent = JSON.stringify(event, null, 2); }
loadEvents();
setInterval(loadEvents, 1000);
</script>
</body>
</html>
```

- [ ] **Step 5: Run tests to verify server passes**

Run:

```bash
go test ./internal/server
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add web/index.html internal/server/server.go internal/server/server_test.go
git commit -m "feat: serve profiler timeline UI"
```

---

### Task 7: Add App Orchestration and CLI

**Files:**
- Create: `internal/app/app.go`
- Create: `internal/app/app_test.go`
- Create: `cmd/cctrace/main.go`

- [ ] **Step 1: Write failing app test**

Create `internal/app/app_test.go`:

```go
package app

import (
	"context"
	"testing"
)

func TestParseArgsForWrappedCommand(t *testing.T) {
	cfg, err := ParseArgs([]string{"claude", "--", "sh", "-c", "true"})
	if err != nil { t.Fatal(err) }
	if cfg.Provider != "claude" { t.Fatalf("Provider = %q", cfg.Provider) }
	if len(cfg.Command) != 3 || cfg.Command[0] != "sh" { t.Fatalf("Command = %#v", cfg.Command) }
}

func TestRunWrappedCommandCompletes(t *testing.T) {
	dir := t.TempDir()
	cfg := Config{Provider: "claude", Command: []string{"sh", "-c", "true"}, StoreDir: dir, Addr: "127.0.0.1:0"}
	if err := Run(context.Background(), cfg); err != nil { t.Fatal(err) }
}
```

- [ ] **Step 2: Run test to verify it fails**

Run:

```bash
go test ./internal/app
```

Expected: FAIL because app package does not exist.

- [ ] **Step 3: Implement app orchestration**

Create `internal/app/app.go`:

```go
package app

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/agentz/cctrace/internal/collectors"
	"github.com/agentz/cctrace/internal/server"
	"github.com/agentz/cctrace/internal/store"
	"github.com/agentz/cctrace/internal/trace"
)

type Config struct {
	Provider string
	Command  []string
	StoreDir string
	Addr     string
}

func ParseArgs(args []string) (Config, error) {
	if len(args) < 3 { return Config{}, fmt.Errorf("usage: cctrace claude|codex -- <command>") }
	provider := args[0]
	if provider != "claude" && provider != "codex" { return Config{}, fmt.Errorf("unsupported provider %q", provider) }
	if args[1] != "--" { return Config{}, fmt.Errorf("expected -- before command") }
	return Config{Provider: provider, Command: args[2:]}, nil
}

func Run(ctx context.Context, cfg Config) error {
	if cfg.StoreDir == "" { cfg.StoreDir = filepath.Join(os.TempDir(), "cctrace-sessions") }
	if cfg.Addr == "" { cfg.Addr = "127.0.0.1:43177" }
	sessionID := "sess_" + strings.NewReplacer(".", "", "-", "").Replace(time.Now().Format("20060102_150405.000000000"))
	hub := server.NewHub()
	st := store.New(cfg.StoreDir)
	session := trace.Session{ID: sessionID, Command: cfg.Command, StartedAt: time.Now()}
	if err := st.CreateSession(session); err != nil { return err }

	addr := cfg.Addr
	if strings.HasSuffix(addr, ":0") {
		ln, err := net.Listen("tcp", addr)
		if err != nil { return err }
		addr = ln.Addr().String()
		_ = ln.Close()
	}
	go func() { _ = server.Listen(addr, hub) }()

	collector := collectors.ProcessCollector{SessionID: sessionID}
	events, _, err := collector.Run(ctx, cfg.Command)
	for _, event := range events {
		hub.Publish(event)
		_ = st.AppendEvent(event)
	}
	return err
}
```

Create `cmd/cctrace/main.go`:

```go
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/agentz/cctrace/internal/app"
)

func main() {
	cfg, err := app.ParseArgs(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	if err := app.Run(context.Background(), cfg); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
```

- [ ] **Step 4: Run app tests**

Run:

```bash
go test ./internal/app
```

Expected: PASS.

- [ ] **Step 5: Run CLI smoke command**

Run:

```bash
go run ./cmd/cctrace claude -- sh -c 'printf cctrace-smoke'
```

Expected: command prints `cctrace-smoke`, prints a `cctrace UI:` URL, and exits 0.

- [ ] **Step 6: Commit**

```bash
git add internal/app/app.go internal/app/app_test.go cmd/cctrace/main.go
git commit -m "feat: add cctrace wrapper CLI"
```

---

### Task 8: Add Replay Command and Documentation

**Files:**
- Modify: `internal/app/app.go`
- Modify: `cmd/cctrace/main.go`
- Create: `README.md`
- Modify: `CLAUDE.md`

- [ ] **Step 1: Write failing parse test for replay**

Append to `internal/app/app_test.go`:

```go
func TestParseArgsForView(t *testing.T) {
	cfg, err := ParseArgs([]string{"view", "sess_1"})
	if err != nil { t.Fatal(err) }
	if cfg.Provider != "view" { t.Fatalf("Provider = %q", cfg.Provider) }
	if len(cfg.Command) != 1 || cfg.Command[0] != "sess_1" { t.Fatalf("Command = %#v", cfg.Command) }
}
```

- [ ] **Step 2: Run test to verify it fails**

Run:

```bash
go test ./internal/app -run TestParseArgsForView
```

Expected: FAIL because `view` is unsupported.

- [ ] **Step 3: Implement view parsing and minimal replay**

Modify `ParseArgs` in `internal/app/app.go` so it starts with:

```go
func ParseArgs(args []string) (Config, error) {
	if len(args) >= 2 && args[0] == "view" {
		return Config{Provider: "view", Command: []string{args[1]}}, nil
	}
	if len(args) < 3 { return Config{}, fmt.Errorf("usage: cctrace claude|codex -- <command>") }
	provider := args[0]
	if provider != "claude" && provider != "codex" { return Config{}, fmt.Errorf("unsupported provider %q", provider) }
	if args[1] != "--" { return Config{}, fmt.Errorf("expected -- before command") }
	return Config{Provider: provider, Command: args[2:]}, nil
}
```

Modify `Run` in `internal/app/app.go` to route view mode before creating a new session:

```go
func Run(ctx context.Context, cfg Config) error {
	if cfg.StoreDir == "" { cfg.StoreDir = filepath.Join(os.TempDir(), "cctrace-sessions") }
	if cfg.Addr == "" { cfg.Addr = "127.0.0.1:43177" }
	if cfg.Provider == "view" {
		hub := server.NewHub()
		st := store.New(cfg.StoreDir)
		events, err := st.ReadEvents(cfg.Command[0])
		if err != nil { return err }
		for _, event := range events { hub.Publish(event) }
		return server.Listen(cfg.Addr, hub)
	}
	// keep the existing wrapper implementation below this line unchanged
```

- [ ] **Step 4: Run tests**

Run:

```bash
go test ./internal/app
```

Expected: PASS.

- [ ] **Step 5: Add README**

Create `README.md`:

```markdown
# cctrace

`cctrace` is a real-time profiler for AI coding agents. The MVP wraps Claude Code or Codex commands and shows process and trace events in a local waterfall UI.

## Commands

Run all tests:

```sh
go test ./...
```

Run the CLI in development:

```sh
go run ./cmd/cctrace claude -- claude
```

Run a smoke command:

```sh
go run ./cmd/cctrace claude -- sh -c 'printf cctrace-smoke'
```

Replay a saved session:

```sh
go run ./cmd/cctrace view <session-id>
```

## MVP status

The first implementation establishes the trace model, store, process collection, parsers, correlation, and a local waterfall UI. ccglass and transcript parsing are represented by tested parser entry points; deeper live tailing integration is the next iteration after the MVP pipeline is working end to end.
```

- [ ] **Step 6: Update CLAUDE.md commands**

Replace the current `## Commands` section in `CLAUDE.md` with:

```markdown
## Commands

- Run all tests: `go test ./...`
- Run trace model tests only: `go test ./internal/trace`
- Run a single test: `go test ./internal/app -run TestParseArgsForWrappedCommand`
- Run the CLI in development: `go run ./cmd/cctrace claude -- <command>`
- Run a smoke trace: `go run ./cmd/cctrace claude -- sh -c 'printf cctrace-smoke'`
- Replay a saved session: `go run ./cmd/cctrace view <session-id>`
```

Replace the current `## Architecture` section in `CLAUDE.md` with:

```markdown
## Architecture

`cctrace` is a Go CLI and local web UI for profiling AI coding agent sessions. The CLI wraps Claude Code or Codex commands, records process and trace events, stores sessions as JSONL, and serves a local timeline UI.

Core packages:

- `internal/trace` defines the shared `Event` and `Session` model.
- `internal/store` persists sessions and events under a session directory.
- `internal/collectors` converts process, transcript, and ccglass records into trace events.
- `internal/correlate` links events heuristically and marks confidence.
- `internal/server` serves the local web UI and event JSON API.
- `internal/app` wires CLI parsing, collection, persistence, and server startup.
```

- [ ] **Step 7: Run full verification**

Run:

```bash
go test ./...
go run ./cmd/cctrace claude -- sh -c 'printf cctrace-smoke'
```

Expected: all tests PASS; smoke command exits 0 and prints `cctrace-smoke`.

- [ ] **Step 8: Commit**

```bash
git add README.md CLAUDE.md internal/app/app.go internal/app/app_test.go cmd/cctrace/main.go
git commit -m "feat: add replay command and docs"
```

---

## Self-Review

Spec coverage:

- Wrapper CLI: Task 7 implements `cctrace claude -- <command>` and `cctrace codex -- <command>` parsing and execution.
- Trace model: Task 1 implements event/session fields from the spec.
- Persistence/replay: Tasks 2 and 8 implement JSONL session storage and `view` replay.
- Collectors: Tasks 3 and 4 implement process events and parser entry points for transcript/ccglass data.
- Correlation: Task 5 implements a first heuristic correlation engine with confidence changes.
- UI: Task 6 implements the local waterfall-style UI and event endpoint.
- Tests: Every task starts with failing tests and ends with passing tests and a commit.

Known deliberate MVP limits:

- Live tailing of ccglass and transcript files is not fully implemented in this first plan; parser entry points are implemented and tested so live watchers can be added as the next focused iteration.
- The UI polls `/api/events` once per second instead of using SSE/WebSocket. This keeps the MVP dependency-free and validates the waterfall interaction before adding push streaming.
- ccglass proxy startup is not automated in the first working slice. The wrapper and event pipeline are established first, then ccglass lifecycle integration can be added safely.

Placeholder scan: no task contains TBD/TODO/FIXME placeholders. Each code-writing step includes concrete code and exact commands.

Type consistency: package names, function names, and constants match across tasks: `trace.Event`, `store.New`, `collectors.ProcessCollector`, `collectors.ParseTranscriptJSONL`, `collectors.ParseCcglassJSON`, `correlate.Correlate`, `server.NewHub`, `app.ParseArgs`, and `app.Run`.
