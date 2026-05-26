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
	if provider == "codex" {
		source = trace.SourceCodexLog
	}
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
		if err != nil {
			ts = time.Now()
		}
		eventType := transcriptEventType(rec.Type)
		status := transcriptStatus(rec.Status)
		title := rec.Tool
		if title == "" {
			title = rec.Type
		}
		events = append(events, trace.Event{
			ID:             stableID(rec.Type, rec.ID, ts),
			SessionID:      sessionID,
			Type:           eventType,
			Title:          title,
			Timestamp:      ts,
			Status:         status,
			Source:         source,
			CorrelationIDs: []string{rec.ID},
			Confidence:     trace.ConfidenceExact,
			Summary:        map[string]any{"provider": provider, "recordType": rec.Type, "tool": rec.Tool},
			RawRef:         &trace.RawRef{File: file, Offset: offset},
		})
		offset += int64(len(line) + 1)
	}
	return events, scanner.Err()
}

func transcriptEventType(kind string) trace.EventType {
	switch kind {
	case "tool_call":
		return trace.EventToolCall
	case "tool_result":
		return trace.EventToolResult
	case "permission":
		return trace.EventPermission
	case "hook":
		return trace.EventHook
	case "skill":
		return trace.EventSkill
	case "subagent":
		return trace.EventSubagent
	case "user_message":
		return trace.EventUserMessage
	case "agent_turn":
		return trace.EventAgentTurn
	default:
		return trace.EventAgentTurn
	}
}

func transcriptStatus(status string) trace.Status {
	switch status {
	case "ok", "success":
		return trace.StatusOK
	case "error", "failed":
		return trace.StatusError
	case "blocked":
		return trace.StatusBlocked
	case "running":
		return trace.StatusRunning
	default:
		return trace.StatusUnknown
	}
}

func parseError(sessionID string, source trace.Source, file string, offset int64, err error) trace.Event {
	now := time.Now()
	return trace.Event{ID: stableID("parse_error", fmt.Sprint(offset), now), SessionID: sessionID, Type: trace.EventError, Title: "parse error", Timestamp: now, Status: trace.StatusError, Source: source, CorrelationIDs: []string{}, Confidence: trace.ConfidenceExact, Summary: map[string]any{"error": err.Error()}, RawRef: &trace.RawRef{File: file, Offset: offset}}
}

func stableID(parts ...any) string {
	items := make([]string, len(parts))
	for i, part := range parts {
		items[i] = fmt.Sprint(part)
	}
	return strings.NewReplacer(":", "_", ".", "_", " ", "_").Replace(strings.Join(items, "_"))
}
