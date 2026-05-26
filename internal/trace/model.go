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
	ConfidenceExact    Confidence = "exact"
	ConfidenceLikely   Confidence = "likely"
	ConfidencePossible Confidence = "possible"
	ConfidenceUnknown  Confidence = "unknown"
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
