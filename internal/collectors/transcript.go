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
	Type          string          `json:"type"`
	Tool          string          `json:"tool"`
	ID            string          `json:"id"`
	UUID          string          `json:"uuid"`
	Status        string          `json:"status"`
	Timestamp     string          `json:"timestamp"`
	Message       json.RawMessage `json:"message"`
	ToolUseResult json.RawMessage `json:"toolUseResult"`
}

type claudeMessage struct {
	Role    string          `json:"role"`
	Content json.RawMessage `json:"content"`
}

type claudeContentBlock struct {
	Type      string          `json:"type"`
	Text      string          `json:"text"`
	Name      string          `json:"name"`
	ID        string          `json:"id"`
	Input     json.RawMessage `json:"input"`
	Content   json.RawMessage `json:"content"`
	ToolUseID string          `json:"tool_use_id"`
	IsError   bool            `json:"is_error"`
}

type claudeAgentInput struct {
	Description  string `json:"description"`
	Isolation    string `json:"isolation"`
	Model        string `json:"model"`
	Prompt       string `json:"prompt"`
	SubagentType string `json:"subagent_type"`
}

type claudeToolUseResult struct {
	Status            string          `json:"status"`
	AgentID           string          `json:"agentId"`
	AgentType         string          `json:"agentType"`
	TotalDurationMS   int64           `json:"totalDurationMs"`
	TotalTokens       float64         `json:"totalTokens"`
	TotalToolUseCount float64         `json:"totalToolUseCount"`
	ToolStats         map[string]any  `json:"toolStats"`
	Content           json.RawMessage `json:"content"`
}

func ParseTranscriptJSONL(sessionID, provider, file string, r io.Reader) ([]trace.Event, error) {
	source := trace.SourceClaudeTranscript
	if provider == "codex" {
		source = trace.SourceCodexLog
	}
	var events []trace.Event
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 1024*1024), 64*1024*1024)
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
		if event, ok := claudeAgentToolUseEvent(sessionID, provider, source, file, offset, rec, ts); ok {
			events = append(events, event)
			offset += int64(len(line) + 1)
			continue
		}
		if event, ok := claudeAgentToolResultEvent(sessionID, provider, source, file, offset, rec, ts); ok {
			events = append(events, event)
			offset += int64(len(line) + 1)
			continue
		}
		if parsed, ok := claudeNativeToolEvents(sessionID, provider, source, file, offset, rec, ts); ok {
			events = append(events, parsed...)
			offset += int64(len(line) + 1)
			continue
		}
		eventType := transcriptEventType(rec.Type)
		status := transcriptStatus(rec.Status)
		title := rec.Tool
		if title == "" {
			title = transcriptTitle(rec)
		}
		correlationID := rec.ID
		if correlationID == "" {
			correlationID = rec.UUID
		}
		summary := map[string]any{"provider": provider, "recordType": rec.Type, "tool": rec.Tool}
		if text := transcriptText(rec); text != "" {
			summary["text"] = text
		}
		events = append(events, trace.Event{
			ID:             stableID(rec.Type, correlationID, ts),
			SessionID:      sessionID,
			Type:           eventType,
			Title:          title,
			Timestamp:      ts,
			Status:         status,
			Source:         source,
			CorrelationIDs: []string{correlationID},
			Confidence:     trace.ConfidenceExact,
			Summary:        summary,
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
	case "user_message", "user":
		return trace.EventUserMessage
	case "agent_turn", "assistant":
		return trace.EventAgentTurn
	default:
		return trace.EventAgentTurn
	}
}

func transcriptStatus(status string) trace.Status {
	switch status {
	case "ok", "success", "completed":
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

func transcriptTitle(rec transcriptRecord) string {
	if text := transcriptText(rec); text != "" {
		return text
	}
	if len(rec.Message) > 0 {
		var message claudeMessage
		if err := json.Unmarshal(rec.Message, &message); err == nil {
			if rec.Type == "assistant" {
				return "assistant"
			}
		}
	}
	return rec.Type
}

func transcriptText(rec transcriptRecord) string {
	if len(rec.Message) == 0 {
		return ""
	}
	var message claudeMessage
	if err := json.Unmarshal(rec.Message, &message); err != nil {
		return ""
	}
	var text string
	if err := json.Unmarshal(message.Content, &text); err == nil {
		return text
	}
	return claudeTextFromRawContent(message.Content)
}

func claudeAgentToolUseEvent(sessionID, provider string, source trace.Source, file string, offset int64, rec transcriptRecord, ts time.Time) (trace.Event, bool) {
	if rec.Type != "assistant" || len(rec.Message) == 0 {
		return trace.Event{}, false
	}
	var message claudeMessage
	if err := json.Unmarshal(rec.Message, &message); err != nil {
		return trace.Event{}, false
	}
	var blocks []claudeContentBlock
	if err := json.Unmarshal(message.Content, &blocks); err != nil {
		return trace.Event{}, false
	}
	for _, block := range blocks {
		if block.Type != "tool_use" || block.Name != "Agent" {
			continue
		}
		var input claudeAgentInput
		if err := json.Unmarshal(block.Input, &input); err != nil {
			return trace.Event{}, false
		}
		title := "Agent"
		if input.SubagentType != "" && input.Description != "" {
			title = input.SubagentType + ": " + input.Description
		} else if input.Description != "" {
			title = input.Description
		} else if input.SubagentType != "" {
			title = input.SubagentType
		}
		return trace.Event{
			ID:             stableID("subagent", block.ID, ts),
			SessionID:      sessionID,
			Type:           trace.EventSubagent,
			Title:          title,
			Timestamp:      ts,
			Status:         trace.StatusRunning,
			Source:         source,
			CorrelationIDs: []string{block.ID},
			Confidence:     trace.ConfidenceExact,
			Summary: map[string]any{
				"provider":     provider,
				"recordType":   rec.Type,
				"phase":        "start",
				"tool":         block.Name,
				"description":  input.Description,
				"subagentType": input.SubagentType,
				"prompt":       input.Prompt,
				"model":        input.Model,
				"isolation":    input.Isolation,
			},
			RawRef: &trace.RawRef{File: file, Offset: offset},
		}, true
	}
	return trace.Event{}, false
}

func claudeAgentToolResultEvent(sessionID, provider string, source trace.Source, file string, offset int64, rec transcriptRecord, ts time.Time) (trace.Event, bool) {
	if len(rec.ToolUseResult) == 0 {
		return trace.Event{}, false
	}
	var result claudeToolUseResult
	if err := json.Unmarshal(rec.ToolUseResult, &result); err != nil || result.AgentType == "" {
		return trace.Event{}, false
	}
	var toolUseID string
	if len(rec.Message) > 0 {
		var message claudeMessage
		if err := json.Unmarshal(rec.Message, &message); err == nil {
			var blocks []claudeContentBlock
			if err := json.Unmarshal(message.Content, &blocks); err == nil {
				for _, block := range blocks {
					if block.ToolUseID != "" {
						toolUseID = block.ToolUseID
						break
					}
				}
			}
		}
	}
	correlationIDs := []string{}
	if toolUseID != "" {
		correlationIDs = append(correlationIDs, toolUseID)
	}
	if result.AgentID != "" {
		correlationIDs = append(correlationIDs, result.AgentID)
	}
	if len(correlationIDs) == 0 && rec.UUID != "" {
		correlationIDs = append(correlationIDs, rec.UUID)
	}
	var duration *int64
	if result.TotalDurationMS > 0 {
		duration = &result.TotalDurationMS
	}
	return trace.Event{
		ID:             stableID("subagent_result", strings.Join(correlationIDs, "_"), ts),
		SessionID:      sessionID,
		Type:           trace.EventSubagent,
		Title:          result.AgentType + " result",
		Timestamp:      ts,
		DurationMS:     duration,
		Status:         transcriptStatus(result.Status),
		Source:         source,
		CorrelationIDs: correlationIDs,
		Confidence:     trace.ConfidenceExact,
		Summary: map[string]any{
			"provider":          provider,
			"recordType":        rec.Type,
			"phase":             "result",
			"agentId":           result.AgentID,
			"agentType":         result.AgentType,
			"result":            claudeTextFromRawContent(result.Content),
			"totalTokens":       result.TotalTokens,
			"totalToolUseCount": result.TotalToolUseCount,
			"toolStats":         result.ToolStats,
		},
		RawRef: &trace.RawRef{File: file, Offset: offset},
	}, true
}

func claudeNativeToolEvents(sessionID, provider string, source trace.Source, file string, offset int64, rec transcriptRecord, ts time.Time) ([]trace.Event, bool) {
	if len(rec.Message) == 0 {
		return nil, false
	}
	var message claudeMessage
	if err := json.Unmarshal(rec.Message, &message); err != nil {
		return nil, false
	}
	var blocks []claudeContentBlock
	if err := json.Unmarshal(message.Content, &blocks); err != nil {
		return nil, false
	}
	var events []trace.Event
	if text := claudeTextBlocks(blocks); text != "" {
		correlationID := rec.ID
		if correlationID == "" {
			correlationID = rec.UUID
		}
		events = append(events, trace.Event{
			ID:             stableID(rec.Type, correlationID, ts),
			SessionID:      sessionID,
			Type:           transcriptEventType(rec.Type),
			Title:          truncateText(text, 96),
			Timestamp:      ts,
			Status:         transcriptStatus(rec.Status),
			Source:         source,
			CorrelationIDs: []string{correlationID},
			Confidence:     trace.ConfidenceExact,
			Summary:        map[string]any{"provider": provider, "recordType": rec.Type, "tool": rec.Tool, "text": text},
			RawRef:         &trace.RawRef{File: file, Offset: offset},
		})
	}
	for _, block := range blocks {
		switch block.Type {
		case "tool_use":
			if block.Name == "Agent" {
				continue
			}
			toolID := block.ID
			if toolID == "" {
				toolID = rec.UUID
			}
			title := block.Name
			if title == "" {
				title = "tool"
			}
			events = append(events, trace.Event{
				ID:             stableID("tool_call", toolID, ts),
				SessionID:      sessionID,
				Type:           trace.EventToolCall,
				Title:          title,
				Timestamp:      ts,
				Status:         trace.StatusRunning,
				Source:         source,
				CorrelationIDs: []string{toolID},
				Confidence:     trace.ConfidenceExact,
				Summary:        map[string]any{"provider": provider, "recordType": rec.Type, "tool": title, "arguments": decodeRawJSON(block.Input)},
				RawRef:         &trace.RawRef{File: file, Offset: offset},
			})
		case "tool_result":
			toolID := block.ToolUseID
			if toolID == "" {
				toolID = rec.ID
			}
			if toolID == "" {
				toolID = rec.UUID
			}
			status := transcriptStatus(rec.Status)
			if block.IsError {
				status = trace.StatusError
			} else if status == trace.StatusUnknown {
				status = trace.StatusOK
			}
			events = append(events, trace.Event{
				ID:             stableID("tool_result", toolID, ts),
				SessionID:      sessionID,
				Type:           trace.EventToolResult,
				Title:          "tool result",
				Timestamp:      ts,
				Status:         status,
				Source:         source,
				CorrelationIDs: []string{toolID},
				Confidence:     trace.ConfidenceExact,
				Summary:        map[string]any{"provider": provider, "recordType": rec.Type, "tool": rec.Tool, "output": truncateText(claudeBlockOutput(block.Content), 2000)},
				RawRef:         &trace.RawRef{File: file, Offset: offset},
			})
		}
	}
	if len(events) == 0 {
		return nil, false
	}
	return events, true
}

func claudeTextBlocks(blocks []claudeContentBlock) string {
	var texts []string
	for _, block := range blocks {
		if block.Type == "text" && block.Text != "" {
			texts = append(texts, block.Text)
		}
	}
	return strings.Join(texts, "\n")
}

func claudeBlockOutput(raw json.RawMessage) string {
	var text string
	if err := json.Unmarshal(raw, &text); err == nil {
		return text
	}
	if text := claudeTextFromRawContent(raw); text != "" {
		return text
	}
	if len(raw) == 0 {
		return ""
	}
	return string(raw)
}

func decodeRawJSON(raw json.RawMessage) any {
	if len(raw) == 0 {
		return nil
	}
	var decoded any
	if err := json.Unmarshal(raw, &decoded); err == nil {
		return decoded
	}
	return string(raw)
}

func truncateText(text string, limit int) string {
	runes := []rune(strings.TrimSpace(text))
	if limit <= 0 || len(runes) <= limit {
		return string(runes)
	}
	if limit == 1 {
		return string(runes[:1])
	}
	return string(runes[:limit-1]) + "..."
}

func claudeTextFromRawContent(raw json.RawMessage) string {
	var blocks []claudeContentBlock
	if err := json.Unmarshal(raw, &blocks); err != nil {
		return ""
	}
	var texts []string
	for _, block := range blocks {
		if block.Text != "" {
			texts = append(texts, block.Text)
		}
	}
	return strings.Join(texts, "\n")
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
