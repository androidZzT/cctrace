package server

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/agentz/cctrace/internal/collectors"
	"github.com/agentz/cctrace/internal/trace"
)

type SessionImportRequest struct {
	Path string `json:"path"`
}

type SessionImportResponse struct {
	Session    SessionMetadata `json:"session"`
	EventCount int             `json:"eventCount"`
	Source     string          `json:"source"`
}

type codexRolloutRecord struct {
	Timestamp string          `json:"timestamp"`
	Type      string          `json:"type"`
	Payload   json.RawMessage `json:"payload"`
	Offset    int64
	Index     int
}

type codexRolloutCall struct {
	CallID    string
	Name      string
	Namespace string
	Args      any
	ArgsText  string
	Timestamp time.Time
	AgentID   string
	Nickname  string
	Output    string
}

type codexSessionMeta struct {
	ID            string `json:"id"`
	Timestamp     string `json:"timestamp"`
	CWD           string `json:"cwd"`
	Originator    string `json:"originator"`
	CLIVersion    string `json:"cli_version"`
	Source        string `json:"source"`
	ThreadSource  string `json:"thread_source"`
	ModelProvider string `json:"model_provider"`
}

type codexSkillDefinition struct {
	Name        string
	Description string
	Path        string
}

type codexSkillIndex struct {
	Definitions []codexSkillDefinition
	ByName      map[string]codexSkillDefinition
	ByPath      map[string]codexSkillDefinition
}

type importFileKind string

const (
	importFileTraceEvents importFileKind = "events_jsonl"
	importFileCodex       importFileKind = "codex_rollout"
	importFileClaude      importFileKind = "claude_transcript"
)

type importCandidate struct {
	Path string
	Kind importFileKind
}

type claudeTranscriptHint struct {
	SessionID   string
	AgentID     string
	IsSidechain *bool
	FirstTime   time.Time
	CWD         string
}

type claudeSubagentMeta struct {
	AgentType   string `json:"agentType"`
	Description string `json:"description"`
	ToolUseID   string `json:"toolUseId"`
}

func ImportSessionPath(path string) (SessionMetadata, []trace.Event, string, error) {
	rawPath := strings.TrimSpace(path)
	if rawPath == "" {
		return SessionMetadata{}, nil, "", errors.New("path is required")
	}
	return ImportSessionPaths([]string{rawPath})
}

func ImportSessionPaths(paths []string) (SessionMetadata, []trace.Event, string, error) {
	candidates, commands, err := collectImportCandidates(paths)
	if err != nil {
		return SessionMetadata{}, nil, "", err
	}
	if len(candidates) == 0 {
		return SessionMetadata{}, nil, "", errors.New("no supported session JSONL files found")
	}
	if len(candidates) == 1 {
		return importSingleCandidate(candidates[0], commands[0])
	}
	return importBundleCandidates(candidates, commands)
}

func ImportUploadedFile(filename string, reader io.Reader) (SessionMetadata, []trace.Event, string, error) {
	safeName := safeUploadFilename(filename)
	tmpDir, err := os.MkdirTemp("", "cctrace-upload-*")
	if err != nil {
		return SessionMetadata{}, nil, "", err
	}
	defer os.RemoveAll(tmpDir)
	tmpPath := filepath.Join(tmpDir, safeName)
	out, err := os.Create(tmpPath)
	if err != nil {
		return SessionMetadata{}, nil, "", err
	}
	_, copyErr := io.Copy(out, reader)
	closeErr := out.Close()
	if copyErr != nil {
		return SessionMetadata{}, nil, "", copyErr
	}
	if closeErr != nil {
		return SessionMetadata{}, nil, "", closeErr
	}
	return ImportUploadedPaths([]string{tmpPath}, map[string]string{tmpPath: "uploaded:" + safeName})
}

func ImportUploadedPaths(paths []string, displayByPath map[string]string) (SessionMetadata, []trace.Event, string, error) {
	session, events, source, err := ImportSessionPaths(paths)
	if err != nil {
		return SessionMetadata{}, nil, "", err
	}
	var command []string
	for _, path := range paths {
		if display := displayByPath[path]; display != "" {
			command = append(command, display)
		}
	}
	if len(command) == 0 {
		command = []string{"uploaded:" + fmt.Sprint(len(paths)) + " files"}
	}
	if len(command) > 1 {
		session.Command = []string{fmt.Sprintf("uploaded:%d files", len(command))}
	} else {
		session.Command = command
	}
	for i := range events {
		if events[i].RawRef == nil {
			continue
		}
		if display := displayByPath[events[i].RawRef.File]; display != "" {
			events[i].RawRef.File = display
		}
	}
	return session, events, source, nil
}

func importSingleCandidate(candidate importCandidate, command string) (SessionMetadata, []trace.Event, string, error) {
	switch candidate.Kind {
	case importFileTraceEvents:
		events, err := readTraceEventsJSONL(candidate.Path)
		if err != nil {
			return SessionMetadata{}, nil, "", err
		}
		session := importedTraceSessionMetadata(command, events)
		normalizeImportedEvents(events, session.ID)
		return session, events, string(candidate.Kind), nil
	case importFileCodex:
		session, events, err := parseCodexRolloutJSONL(candidate.Path)
		if err != nil {
			return SessionMetadata{}, nil, "", err
		}
		session.Command = []string{command}
		return session, events, string(candidate.Kind), nil
	case importFileClaude:
		sessionID := "claude_" + sanitizeSessionID(strings.TrimSuffix(filepath.Base(candidate.Path), filepath.Ext(candidate.Path)))
		if hint := inspectClaudeTranscript(candidate.Path); hint.SessionID != "" {
			sessionID = "claude_" + sanitizeSessionID(hint.SessionID)
		}
		events, err := parseClaudeTranscriptCandidate(sessionID, candidate.Path)
		if err != nil {
			return SessionMetadata{}, nil, "", err
		}
		session := SessionMetadata{ID: sessionID, Provider: "claude", Mode: "history", Command: []string{command}}
		normalizeImportedEvents(events, session.ID)
		sortEvents(events)
		return session, events, string(candidate.Kind), nil
	default:
		return SessionMetadata{}, nil, "", fmt.Errorf("unsupported import file kind %q", candidate.Kind)
	}
}

func importBundleCandidates(candidates []importCandidate, commands []string) (SessionMetadata, []trace.Event, string, error) {
	provider, source := bundleProviderAndSource(candidates)
	sessionID := bundleSessionID(provider, candidates, commands)
	var events []trace.Event
	for _, candidate := range candidates {
		switch candidate.Kind {
		case importFileTraceEvents:
			parsed, err := readTraceEventsJSONL(candidate.Path)
			if err != nil {
				return SessionMetadata{}, nil, "", err
			}
			for i := range parsed {
				parsed[i] = normalizeEventShape(parsed[i])
				markBundleSource(&parsed[i], parsed[i].SessionID, candidate.Path)
				parsed[i].SessionID = sessionID
			}
			events = append(events, parsed...)
		case importFileCodex:
			sourceSession, parsed, err := parseCodexRolloutJSONL(candidate.Path)
			if err != nil {
				return SessionMetadata{}, nil, "", err
			}
			for i := range parsed {
				parsed[i] = normalizeEventShape(parsed[i])
				markBundleSource(&parsed[i], sourceSession.ID, candidate.Path)
				parsed[i].SessionID = sessionID
			}
			events = append(events, parsed...)
		case importFileClaude:
			parsed, err := parseClaudeTranscriptCandidate(sessionID, candidate.Path)
			if err != nil {
				return SessionMetadata{}, nil, "", err
			}
			hint := inspectClaudeTranscript(candidate.Path)
			for i := range parsed {
				parsed[i] = normalizeEventShape(parsed[i])
				if sourceID, ok := parsed[i].Summary["sourceSessionId"].(string); !ok || sourceID == "" {
					if hint.SessionID != "" {
						parsed[i].Summary["sourceSessionId"] = hint.SessionID
					}
				}
				parsed[i].Summary["sourceFile"] = candidate.Path
				parsed[i].SessionID = sessionID
			}
			events = append(events, parsed...)
		}
	}
	if len(events) == 0 {
		return SessionMetadata{}, nil, "", errors.New("session bundle contains no events")
	}
	sortEvents(events)
	return SessionMetadata{ID: sessionID, Provider: provider, Mode: "history", Command: commands}, events, source, nil
}

func safeUploadFilename(filename string) string {
	name := filepath.Base(strings.TrimSpace(filename))
	if name == "." || name == string(filepath.Separator) || name == "" {
		return "session.jsonl"
	}
	name = strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z':
			return r
		case r >= 'A' && r <= 'Z':
			return r
		case r >= '0' && r <= '9':
			return r
		case r == '.', r == '-', r == '_':
			return r
		default:
			return '_'
		}
	}, name)
	name = strings.Trim(name, "._-")
	if name == "" {
		return "session.jsonl"
	}
	return name
}

func expandHomePath(path string) string {
	if path == "~" || strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err == nil {
			if path == "~" {
				return home
			}
			return filepath.Join(home, strings.TrimPrefix(path, "~/"))
		}
	}
	return path
}

func looksLikeTraceEventsJSONL(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1024*1024), 64*1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var probe struct {
			ID        string `json:"id"`
			SessionID string `json:"sessionId"`
		}
		return json.Unmarshal([]byte(line), &probe) == nil && probe.ID != "" && probe.SessionID != ""
	}
	return false
}

func readTraceEventsJSONL(path string) ([]trace.Event, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var events []trace.Event
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1024*1024), 64*1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var event trace.Event
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			return nil, err
		}
		if event.ID == "" {
			return nil, errors.New("events.jsonl contains an event without id")
		}
		events = append(events, normalizeEventShape(event))
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if len(events) == 0 {
		return nil, errors.New("events.jsonl contains no events")
	}
	return events, nil
}

func importedTraceSessionMetadata(path string, events []trace.Event) SessionMetadata {
	sessionID := ""
	provider := "import"
	for _, event := range events {
		if sessionID == "" && event.SessionID != "" {
			sessionID = event.SessionID
		}
		if provider == "import" {
			if value, ok := event.Summary["provider"].(string); ok && value != "" {
				provider = value
			} else if event.Source == trace.SourceCodexLog {
				provider = "codex"
			} else if event.Source == trace.SourceClaudeTranscript {
				provider = "claude"
			}
		}
	}
	if sessionID == "" {
		sessionID = "import_" + sanitizeSessionID(filepath.Base(filepath.Dir(path)))
	}
	return SessionMetadata{ID: sessionID, Provider: provider, Mode: "history", Command: []string{path}}
}

func normalizeImportedEvents(events []trace.Event, sessionID string) {
	for i := range events {
		events[i] = normalizeEventShape(events[i])
		events[i].SessionID = sessionID
	}
}

func collectImportCandidates(paths []string) ([]importCandidate, []string, error) {
	var commands []string
	var sourcePaths []string
	seenPaths := map[string]struct{}{}
	for _, rawPath := range paths {
		rawPath = strings.TrimSpace(rawPath)
		if rawPath == "" {
			continue
		}
		cleanPath := filepath.Clean(expandHomePath(rawPath))
		commands = append(commands, cleanPath)
		collected, err := collectCandidatePaths(cleanPath)
		if err != nil {
			return nil, nil, err
		}
		for _, path := range collected {
			path = filepath.Clean(path)
			if _, ok := seenPaths[path]; ok {
				continue
			}
			seenPaths[path] = struct{}{}
			sourcePaths = append(sourcePaths, path)
		}
	}
	if len(commands) == 0 {
		return nil, nil, errors.New("path is required")
	}
	sourcePaths = expandClaudeRelatedFiles(sourcePaths)
	var candidates []importCandidate
	for _, path := range sourcePaths {
		candidate, ok, err := classifyImportPath(path)
		if err != nil {
			return nil, nil, err
		}
		if ok {
			candidates = append(candidates, candidate)
		}
	}
	if len(candidates) == 0 {
		return nil, nil, fmt.Errorf("no supported session JSONL files found in %s", strings.Join(commands, ", "))
	}
	return candidates, commands, nil
}

func collectCandidatePaths(path string) ([]string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return []string{path}, nil
	}
	if eventsPath := filepath.Join(path, "events.jsonl"); fileExists(eventsPath) {
		return []string{eventsPath}, nil
	}
	var paths []string
	if filepath.Base(path) == "subagents" {
		parent := filepath.Dir(path)
		mainPath := parent + ".jsonl"
		appendFileIfExists(&paths, mainPath)
		appendGlob(&paths, filepath.Join(path, "agent-*.jsonl"))
		return paths, nil
	}
	if fileExists(filepath.Join(path, "subagents")) {
		mainPath := path + ".jsonl"
		appendFileIfExists(&paths, mainPath)
		appendGlob(&paths, filepath.Join(path, "*.jsonl"))
		appendGlob(&paths, filepath.Join(path, "subagents", "agent-*.jsonl"))
		return paths, nil
	}
	rollouts := globFiles(filepath.Join(path, "rollout-*.jsonl"))
	if len(rollouts) > 0 {
		paths = append(paths, rollouts...)
		return paths, nil
	}
	appendGlob(&paths, filepath.Join(path, "*.jsonl"))
	return paths, nil
}

func expandClaudeRelatedFiles(paths []string) []string {
	seen := map[string]struct{}{}
	var expanded []string
	add := func(path string) {
		path = filepath.Clean(path)
		if _, ok := seen[path]; ok {
			return
		}
		seen[path] = struct{}{}
		expanded = append(expanded, path)
	}
	for _, path := range paths {
		add(path)
		if filepath.Ext(path) != ".jsonl" {
			continue
		}
		subagentsDir := strings.TrimSuffix(path, filepath.Ext(path))
		if !fileExists(filepath.Join(subagentsDir, "subagents")) {
			continue
		}
		for _, subagentPath := range globFiles(filepath.Join(subagentsDir, "subagents", "agent-*.jsonl")) {
			add(subagentPath)
		}
	}
	return expanded
}

func classifyImportPath(path string) (importCandidate, bool, error) {
	info, err := os.Stat(path)
	if err != nil {
		return importCandidate{}, false, err
	}
	if info.IsDir() || filepath.Ext(path) != ".jsonl" {
		return importCandidate{}, false, nil
	}
	switch {
	case filepath.Base(path) == "events.jsonl" || looksLikeTraceEventsJSONL(path):
		return importCandidate{Path: path, Kind: importFileTraceEvents}, true, nil
	case looksLikeCodexRolloutJSONL(path):
		return importCandidate{Path: path, Kind: importFileCodex}, true, nil
	case looksLikeClaudeTranscriptJSONL(path):
		return importCandidate{Path: path, Kind: importFileClaude}, true, nil
	default:
		return importCandidate{}, false, nil
	}
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func appendFileIfExists(paths *[]string, path string) {
	info, err := os.Stat(path)
	if err == nil && !info.IsDir() {
		*paths = append(*paths, path)
	}
}

func appendGlob(paths *[]string, pattern string) {
	*paths = append(*paths, globFiles(pattern)...)
}

func globFiles(pattern string) []string {
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil
	}
	files := matches[:0]
	for _, match := range matches {
		info, err := os.Stat(match)
		if err == nil && !info.IsDir() {
			files = append(files, match)
		}
	}
	sort.Strings(files)
	return files
}

func looksLikeCodexRolloutJSONL(path string) bool {
	line, ok := firstJSONLine(path)
	if !ok {
		return false
	}
	var probe struct {
		Type    string          `json:"type"`
		Payload json.RawMessage `json:"payload"`
	}
	if err := json.Unmarshal([]byte(line), &probe); err != nil {
		return false
	}
	if probe.Type == "" || len(probe.Payload) == 0 {
		return false
	}
	switch probe.Type {
	case "session_meta", "turn_context", "event_msg", "response_item":
		return true
	default:
		return false
	}
}

func looksLikeClaudeTranscriptJSONL(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1024*1024), 64*1024*1024)
	checked := 0
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		checked++
		var probe struct {
			Type      string          `json:"type"`
			SessionID string          `json:"sessionId"`
			UUID      string          `json:"uuid"`
			Message   json.RawMessage `json:"message"`
			Payload   json.RawMessage `json:"payload"`
		}
		if json.Unmarshal([]byte(line), &probe) == nil && len(probe.Payload) == 0 {
			if probe.SessionID != "" || probe.UUID != "" || len(probe.Message) > 0 {
				return true
			}
			switch probe.Type {
			case "permission-mode", "file-history-snapshot", "user", "assistant":
				return true
			}
		}
		if checked >= 20 {
			break
		}
	}
	return false
}

func firstJSONLine(path string) (string, bool) {
	f, err := os.Open(path)
	if err != nil {
		return "", false
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1024*1024), 64*1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			return line, true
		}
	}
	return "", false
}

func bundleProviderAndSource(candidates []importCandidate) (string, string) {
	counts := map[importFileKind]int{}
	for _, candidate := range candidates {
		counts[candidate.Kind]++
	}
	if len(counts) == 1 {
		switch candidates[0].Kind {
		case importFileCodex:
			return "codex", "codex_bundle"
		case importFileClaude:
			return "claude", "claude_bundle"
		case importFileTraceEvents:
			return "import", "events_bundle"
		}
	}
	return "mixed", "mixed_bundle"
}

func bundleSessionID(provider string, candidates []importCandidate, commands []string) string {
	if provider == "claude" {
		ids := map[string]struct{}{}
		for _, candidate := range candidates {
			if hint := inspectClaudeTranscript(candidate.Path); hint.SessionID != "" {
				ids[hint.SessionID] = struct{}{}
			}
		}
		if len(ids) == 1 {
			for id := range ids {
				return "claude_" + sanitizeSessionID(id)
			}
		}
	}
	label := strings.Join(commands, "_")
	if label == "" && len(candidates) > 0 {
		label = filepath.Base(filepath.Dir(candidates[0].Path))
	}
	id := sanitizeSessionID(label)
	if len(id) > 96 {
		id = id[len(id)-96:]
	}
	return provider + "_bundle_" + id
}

func markBundleSource(event *trace.Event, sourceSessionID, sourceFile string) {
	if event.Summary == nil {
		event.Summary = map[string]any{}
	}
	if sourceSessionID != "" {
		event.Summary["sourceSessionId"] = sourceSessionID
	}
	event.Summary["sourceFile"] = sourceFile
}

func sortEvents(events []trace.Event) {
	sort.SliceStable(events, func(i, j int) bool {
		return events[i].Timestamp.Before(events[j].Timestamp)
	})
}

func parseClaudeTranscriptCandidate(sessionID, path string) ([]trace.Event, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	events, err := collectors.ParseTranscriptJSONL(sessionID, "claude", path, f)
	if err != nil {
		return nil, err
	}
	hint := inspectClaudeTranscript(path)
	meta := readClaudeSubagentMeta(path)
	for i := range events {
		events[i] = normalizeEventShape(events[i])
		if hint.SessionID != "" {
			events[i].Summary["sourceSessionId"] = hint.SessionID
		}
		events[i].Summary["sourceFile"] = path
		if hint.CWD != "" {
			events[i].Summary["cwd"] = hint.CWD
		}
		if hint.IsSidechain != nil {
			events[i].Summary["isSidechain"] = *hint.IsSidechain
		}
		if hint.AgentID != "" {
			if events[i].Summary["agentId"] == nil {
				events[i].Summary["agentId"] = hint.AgentID
			}
			events[i].CorrelationIDs = append(events[i].CorrelationIDs, hint.AgentID)
		}
		if meta.ToolUseID != "" {
			events[i].CorrelationIDs = append(events[i].CorrelationIDs, meta.ToolUseID)
		}
		if meta.AgentType != "" && events[i].Summary["agentType"] == nil {
			events[i].Summary["agentType"] = meta.AgentType
		}
		if meta.Description != "" && events[i].Summary["description"] == nil {
			events[i].Summary["description"] = meta.Description
		}
		events[i].CorrelationIDs = compactStrings(events[i].CorrelationIDs)
	}
	if synthetic, ok := claudeSubagentTranscriptEvent(sessionID, path, hint, meta, events); ok {
		events = append(events, synthetic)
	}
	return events, nil
}

func inspectClaudeTranscript(path string) claudeTranscriptHint {
	f, err := os.Open(path)
	if err != nil {
		return claudeTranscriptHint{}
	}
	defer f.Close()
	var hint claudeTranscriptHint
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1024*1024), 64*1024*1024)
	checked := 0
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		checked++
		var rec struct {
			SessionID   string `json:"sessionId"`
			AgentID     string `json:"agentId"`
			IsSidechain *bool  `json:"isSidechain"`
			Timestamp   string `json:"timestamp"`
			CWD         string `json:"cwd"`
		}
		if err := json.Unmarshal([]byte(line), &rec); err != nil {
			continue
		}
		if hint.SessionID == "" {
			hint.SessionID = rec.SessionID
		}
		if hint.AgentID == "" {
			hint.AgentID = rec.AgentID
		}
		if hint.IsSidechain == nil {
			hint.IsSidechain = rec.IsSidechain
		}
		if hint.CWD == "" {
			hint.CWD = rec.CWD
		}
		if hint.FirstTime.IsZero() && rec.Timestamp != "" {
			if ts, err := time.Parse(time.RFC3339Nano, rec.Timestamp); err == nil {
				hint.FirstTime = ts
			}
		}
		if hint.SessionID != "" && hint.AgentID != "" && hint.IsSidechain != nil && !hint.FirstTime.IsZero() {
			break
		}
		if checked >= 200 {
			break
		}
	}
	return hint
}

func readClaudeSubagentMeta(path string) claudeSubagentMeta {
	metaPath := strings.TrimSuffix(path, filepath.Ext(path)) + ".meta.json"
	data, err := os.ReadFile(metaPath)
	if err != nil {
		return claudeSubagentMeta{}
	}
	var meta claudeSubagentMeta
	_ = json.Unmarshal(data, &meta)
	return meta
}

func claudeSubagentTranscriptEvent(sessionID, path string, hint claudeTranscriptHint, meta claudeSubagentMeta, events []trace.Event) (trace.Event, bool) {
	agentID := hint.AgentID
	if agentID == "" {
		base := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
		if strings.HasPrefix(base, "agent-") {
			agentID = strings.TrimPrefix(base, "agent-")
		}
	}
	if agentID == "" {
		return trace.Event{}, false
	}
	ts := hint.FirstTime
	if ts.IsZero() && len(events) > 0 {
		ts = events[0].Timestamp
	}
	if ts.IsZero() {
		ts = time.Now()
	}
	title := "Agent " + agentID
	if meta.AgentType != "" && meta.Description != "" {
		title = meta.AgentType + ": " + meta.Description
	} else if meta.Description != "" {
		title = meta.Description
	} else if meta.AgentType != "" {
		title = meta.AgentType
	}
	summary := map[string]any{
		"provider":    "claude",
		"recordType":  "subagent_transcript",
		"phase":       "transcript",
		"agentId":     agentID,
		"agentType":   meta.AgentType,
		"description": meta.Description,
		"sourceFile":  path,
	}
	if hint.SessionID != "" {
		summary["sourceSessionId"] = hint.SessionID
	}
	if hint.IsSidechain != nil {
		summary["isSidechain"] = *hint.IsSidechain
	}
	return trace.Event{
		ID:             importStableID(sessionID, path, "subagent_transcript", agentID),
		SessionID:      sessionID,
		Type:           trace.EventSubagent,
		Title:          title,
		Timestamp:      ts,
		Status:         trace.StatusOK,
		Source:         trace.SourceClaudeTranscript,
		CorrelationIDs: compactStrings([]string{agentID, meta.ToolUseID}),
		Confidence:     trace.ConfidenceExact,
		Summary:        summary,
		RawRef:         &trace.RawRef{File: path},
	}, true
}

func normalizeEventShape(event trace.Event) trace.Event {
	if event.CorrelationIDs == nil {
		event.CorrelationIDs = []string{}
	}
	if event.Summary == nil {
		event.Summary = map[string]any{}
	}
	if event.Status == "" {
		event.Status = trace.StatusUnknown
	}
	if event.Confidence == "" {
		event.Confidence = trace.ConfidenceUnknown
	}
	return event
}

func parseCodexRolloutJSONL(path string) (SessionMetadata, []trace.Event, error) {
	records, err := readCodexRolloutRecords(path)
	if err != nil {
		return SessionMetadata{}, nil, err
	}
	if len(records) == 0 {
		return SessionMetadata{}, nil, errors.New("rollout JSONL contains no records")
	}
	meta := codexSessionMeta{}
	for _, rec := range records {
		if rec.Type == "session_meta" {
			_ = json.Unmarshal(rec.Payload, &meta)
			break
		}
	}
	sessionID := "codex_" + sanitizeSessionID(meta.ID)
	if meta.ID == "" {
		sessionID = "import_" + sanitizeSessionID(strings.TrimSuffix(filepath.Base(path), filepath.Ext(path)))
	}
	session := SessionMetadata{ID: sessionID, Provider: "codex", Mode: "history", Command: []string{path}}
	calls := indexCodexRolloutCalls(records)
	skills := indexCodexRolloutSkills(records)
	var events []trace.Event
	for _, rec := range records {
		ts := codexRecordTime(rec)
		switch rec.Type {
		case "session_meta":
			events = append(events, codexSessionMetaEvent(sessionID, path, rec, meta, ts))
		case "turn_context":
			events = append(events, codexBasicEvent(sessionID, path, rec, trace.EventHook, "turn context", trace.StatusOK, map[string]any{"provider": "codex"}))
		case "event_msg":
			events = append(events, codexEventMsgEvents(sessionID, path, rec)...)
		case "response_item":
			events = append(events, codexResponseItemEvents(sessionID, path, rec, calls, skills)...)
		default:
			events = append(events, codexBasicEvent(sessionID, path, rec, trace.EventHook, rec.Type, trace.StatusUnknown, map[string]any{"provider": "codex"}))
		}
	}
	return session, events, nil
}

func readCodexRolloutRecords(path string) ([]codexRolloutRecord, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var records []codexRolloutRecord
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1024*1024), 64*1024*1024)
	var offset int64
	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			offset += int64(len(line) + 1)
			continue
		}
		var rec codexRolloutRecord
		if err := json.Unmarshal([]byte(trimmed), &rec); err != nil {
			return nil, err
		}
		rec.Offset = offset
		rec.Index = len(records)
		records = append(records, rec)
		offset += int64(len(line) + 1)
	}
	return records, scanner.Err()
}

func indexCodexRolloutCalls(records []codexRolloutRecord) map[string]*codexRolloutCall {
	calls := map[string]*codexRolloutCall{}
	for _, rec := range records {
		if rec.Type != "response_item" {
			continue
		}
		payloadType := rawFieldString(rec.Payload, "type")
		switch payloadType {
		case "function_call", "custom_tool_call", "tool_search_call":
			callID := rawFieldString(rec.Payload, "call_id")
			if callID == "" {
				callID = importStableID("call", rec.Index)
			}
			name := rawFieldString(rec.Payload, "name")
			if name == "" {
				name = payloadType
			}
			argsText := rawFieldString(rec.Payload, "arguments")
			calls[callID] = &codexRolloutCall{
				CallID:    callID,
				Name:      name,
				Namespace: rawFieldString(rec.Payload, "namespace"),
				Args:      decodeJSONText(argsText),
				ArgsText:  argsText,
				Timestamp: codexRecordTime(rec),
			}
		}
	}
	for _, rec := range records {
		if rec.Type != "response_item" || rawFieldString(rec.Payload, "type") != "function_call_output" {
			continue
		}
		callID := rawFieldString(rec.Payload, "call_id")
		call := calls[callID]
		if call == nil {
			continue
		}
		call.Output = rawFieldString(rec.Payload, "output")
		if call.Name == "spawn_agent" {
			if output := decodeJSONObject(call.Output); output != nil {
				call.AgentID = stringFromAny(output["agent_id"])
				if call.AgentID == "" {
					call.AgentID = stringFromAny(output["agentId"])
				}
				call.Nickname = stringFromAny(output["nickname"])
			}
		}
	}
	return calls
}

func indexCodexRolloutSkills(records []codexRolloutRecord) codexSkillIndex {
	index := codexSkillIndex{
		ByName: map[string]codexSkillDefinition{},
		ByPath: map[string]codexSkillDefinition{},
	}
	seen := map[string]struct{}{}
	for _, rec := range records {
		if rec.Type != "response_item" || rawFieldString(rec.Payload, "type") != "message" {
			continue
		}
		role := rawFieldString(rec.Payload, "role")
		if role != "developer" && role != "system" {
			continue
		}
		text := codexContentText(rawFieldRaw(rec.Payload, "content"))
		for _, skill := range parseCodexSkillDefinitions(text) {
			key := normalizeSkillKey(skill.Name + "|" + skill.Path)
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			index.Definitions = append(index.Definitions, skill)
			if skill.Name != "" {
				index.ByName[normalizeSkillKey(skill.Name)] = skill
			}
			if skill.Path != "" {
				index.ByPath[filepath.Clean(expandHomePath(skill.Path))] = skill
			}
		}
	}
	return index
}

func parseCodexSkillDefinitions(text string) []codexSkillDefinition {
	var skills []codexSkillDefinition
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "- ") || !strings.Contains(line, "(file:") {
			continue
		}
		pathStart := strings.LastIndex(line, "(file:")
		pathEnd := strings.LastIndex(line, ")")
		if pathStart < 0 || pathEnd <= pathStart {
			continue
		}
		path := strings.TrimSpace(line[pathStart+len("(file:") : pathEnd])
		head := strings.TrimSpace(strings.TrimPrefix(line[:pathStart], "- "))
		name, description, ok := strings.Cut(head, ": ")
		if !ok {
			continue
		}
		name = strings.TrimSpace(name)
		description = strings.TrimSpace(description)
		if name == "" || path == "" {
			continue
		}
		skills = append(skills, codexSkillDefinition{Name: name, Description: description, Path: path})
	}
	return skills
}

func codexEventMsgEvents(sessionID, path string, rec codexRolloutRecord) []trace.Event {
	payloadType := rawFieldString(rec.Payload, "type")
	switch payloadType {
	case "user_message":
		text := rawFieldString(rec.Payload, "message")
		return []trace.Event{codexBasicEvent(sessionID, path, rec, trace.EventUserMessage, shortTitle(text, "user message"), trace.StatusOK, map[string]any{"provider": "codex", "text": text})}
	case "agent_message":
		text := rawFieldString(rec.Payload, "message")
		return []trace.Event{codexBasicEvent(sessionID, path, rec, trace.EventAgentTurn, shortTitle(text, "agent message"), trace.StatusOK, map[string]any{"provider": "codex", "text": text, "phase": rawFieldString(rec.Payload, "phase")})}
	case "token_count":
		return []trace.Event{codexBasicEvent(sessionID, path, rec, trace.EventAPIResponse, "token count", trace.StatusOK, map[string]any{"provider": "codex", "info": rawFieldAny(rec.Payload, "info"), "rateLimits": rawFieldAny(rec.Payload, "rate_limits")})}
	case "task_started":
		return []trace.Event{codexBasicEvent(sessionID, path, rec, trace.EventProcess, "task started", trace.StatusRunning, map[string]any{"provider": "codex"})}
	case "task_complete":
		return []trace.Event{codexBasicEvent(sessionID, path, rec, trace.EventProcess, "task complete", trace.StatusOK, map[string]any{"provider": "codex"})}
	case "patch_apply_end":
		return []trace.Event{codexBasicEvent(sessionID, path, rec, trace.EventToolResult, "apply_patch result", trace.StatusOK, map[string]any{"provider": "codex", "tool": "apply_patch"})}
	default:
		return []trace.Event{codexBasicEvent(sessionID, path, rec, trace.EventHook, payloadType, trace.StatusUnknown, map[string]any{"provider": "codex"})}
	}
}

func codexResponseItemEvents(sessionID, path string, rec codexRolloutRecord, calls map[string]*codexRolloutCall, skills codexSkillIndex) []trace.Event {
	payloadType := rawFieldString(rec.Payload, "type")
	switch payloadType {
	case "message":
		role := rawFieldString(rec.Payload, "role")
		text := codexContentText(rawFieldRaw(rec.Payload, "content"))
		eventType := trace.EventAgentTurn
		if role == "user" {
			eventType = trace.EventUserMessage
		} else if role == "developer" || role == "system" {
			eventType = trace.EventHook
		}
		events := []trace.Event{codexBasicEvent(sessionID, path, rec, eventType, shortTitle(text, role), trace.StatusOK, map[string]any{"provider": "codex", "role": role, "text": text})}
		if role == "developer" || role == "system" {
			for _, skill := range parseCodexSkillDefinitions(text) {
				events = append(events, codexSkillEvent(sessionID, path, rec, skill, "available", "skills_instructions", trace.StatusOK, trace.ConfidenceExact))
			}
		}
		return events
	case "reasoning":
		text := codexContentText(rawFieldRaw(rec.Payload, "summary"))
		if text == "" {
			text = "reasoning"
		}
		return []trace.Event{codexBasicEvent(sessionID, path, rec, trace.EventAgentTurn, shortTitle(text, "reasoning"), trace.StatusOK, map[string]any{"provider": "codex", "phase": "reasoning", "text": text})}
	case "function_call", "custom_tool_call", "tool_search_call":
		callID := rawFieldString(rec.Payload, "call_id")
		call := calls[callID]
		if call != nil && call.Name == "spawn_agent" && call.AgentID != "" {
			title := "spawn " + call.AgentID
			if call.Nickname != "" {
				title = "spawn " + call.Nickname
			}
			return []trace.Event{codexBasicEventWithIDs(sessionID, path, rec, trace.EventSubagent, title, trace.StatusRunning, []string{callID, call.AgentID}, map[string]any{"provider": "codex", "phase": "start", "tool": call.Name, "agentId": call.AgentID, "nickname": call.Nickname, "arguments": call.Args})}
		}
		name := rawFieldString(rec.Payload, "name")
		if name == "" {
			name = payloadType
		}
		events := []trace.Event{codexBasicEventWithIDs(sessionID, path, rec, trace.EventToolCall, name, trace.StatusRunning, []string{callID}, map[string]any{"provider": "codex", "tool": name, "namespace": rawFieldString(rec.Payload, "namespace"), "arguments": decodeJSONText(rawFieldString(rec.Payload, "arguments"))})}
		events = append(events, codexSkillInvocationEvents(sessionID, path, rec, skills)...)
		return events
	case "function_call_output", "tool_search_output":
		return codexFunctionOutputEvents(sessionID, path, rec, calls)
	default:
		return []trace.Event{codexBasicEvent(sessionID, path, rec, trace.EventHook, payloadType, trace.StatusUnknown, map[string]any{"provider": "codex"})}
	}
}

func codexSkillInvocationEvents(sessionID, path string, rec codexRolloutRecord, skills codexSkillIndex) []trace.Event {
	text := strings.Join([]string{
		rawFieldString(rec.Payload, "name"),
		rawFieldString(rec.Payload, "namespace"),
		rawFieldString(rec.Payload, "arguments"),
	}, " ")
	matches := map[string]codexSkillDefinition{}
	for skillPath, skill := range skills.ByPath {
		if skillPath != "" && strings.Contains(text, skillPath) {
			matches[skillPath] = skill
		}
	}
	for _, skillPath := range skillDocumentPaths(text) {
		clean := filepath.Clean(expandHomePath(skillPath))
		if _, ok := matches[clean]; ok {
			continue
		}
		skill, ok := skills.ByPath[clean]
		if !ok {
			skill = codexSkillDefinition{Name: filepath.Base(filepath.Dir(clean)), Path: clean}
		}
		matches[clean] = skill
	}
	events := make([]trace.Event, 0, len(matches))
	for _, skill := range matches {
		events = append(events, codexSkillEvent(sessionID, path, rec, skill, "invoked", "skill_document_read", trace.StatusOK, trace.ConfidenceLikely))
	}
	return events
}

func skillDocumentPaths(text string) []string {
	matcher := regexp.MustCompile(`[A-Za-z0-9_./~+@:-]+/SKILL\.md`)
	return compactStrings(matcher.FindAllString(text, -1))
}

func codexSkillEvent(sessionID, path string, rec codexRolloutRecord, skill codexSkillDefinition, phase, trigger string, status trace.Status, confidence trace.Confidence) trace.Event {
	skillPath := filepath.Clean(expandHomePath(skill.Path))
	title := skill.Name
	if phase == "invoked" {
		title = skill.Name + " invoked"
	}
	summary := map[string]any{
		"provider":    "codex",
		"phase":       phase,
		"trigger":     trigger,
		"description": skill.Description,
		"path":        skillPath,
	}
	if summary["description"] == "" {
		delete(summary, "description")
	}
	if skillPath == "." {
		delete(summary, "path")
	}
	return trace.Event{
		ID:             importStableID(sessionID, rec.Index, trace.EventSkill, phase, skill.Name, skillPath),
		SessionID:      sessionID,
		Type:           trace.EventSkill,
		Title:          title,
		Timestamp:      codexRecordTime(rec),
		Status:         status,
		Source:         trace.SourceCodexLog,
		CorrelationIDs: compactStrings([]string{"skill:" + skill.Name, skillPath}),
		Confidence:     confidence,
		Summary:        summary,
		RawRef:         &trace.RawRef{File: path, Offset: rec.Offset},
	}
}

func codexFunctionOutputEvents(sessionID, path string, rec codexRolloutRecord, calls map[string]*codexRolloutCall) []trace.Event {
	callID := rawFieldString(rec.Payload, "call_id")
	call := calls[callID]
	output := rawFieldString(rec.Payload, "output")
	if call != nil && call.Name == "spawn_agent" && call.AgentID != "" {
		return nil
	}
	if call != nil && call.Name == "wait_agent" {
		if events := codexWaitAgentResultEvents(sessionID, path, rec, callID, output, calls); len(events) > 0 {
			return events
		}
	}
	title := "tool result"
	toolName := ""
	if call != nil {
		toolName = call.Name
		title = call.Name + " result"
	}
	return []trace.Event{codexBasicEventWithIDs(sessionID, path, rec, trace.EventToolResult, title, trace.StatusOK, []string{callID}, map[string]any{"provider": "codex", "tool": toolName, "output": truncate(output, 2000)})}
}

func codexWaitAgentResultEvents(sessionID, path string, rec codexRolloutRecord, callID, output string, calls map[string]*codexRolloutCall) []trace.Event {
	obj := decodeJSONObject(output)
	if obj == nil {
		return nil
	}
	statusRaw, ok := obj["status"].(map[string]any)
	if !ok {
		return nil
	}
	var events []trace.Event
	for agentID, raw := range statusRaw {
		statusMap, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		state := "completed"
		result := stringFromAny(statusMap["completed"])
		eventStatus := trace.StatusOK
		if result == "" {
			state = "failed"
			result = stringFromAny(statusMap["failed"])
			eventStatus = trace.StatusError
		}
		if result == "" {
			state = "blocked"
			result = stringFromAny(statusMap["blocked"])
			eventStatus = trace.StatusBlocked
		}
		if result == "" {
			continue
		}
		nickname := nicknameForAgent(calls, agentID)
		label := agentID
		if nickname != "" {
			label = nickname
		}
		events = append(events, codexBasicEventWithIDs(sessionID, path, rec, trace.EventSubagent, label+" "+state, eventStatus, []string{callID, agentID}, map[string]any{"provider": "codex", "phase": "result", "agentId": agentID, "nickname": nickname, "result": truncate(result, 2000)}))
	}
	return events
}

func nicknameForAgent(calls map[string]*codexRolloutCall, agentID string) string {
	for _, call := range calls {
		if call.AgentID == agentID {
			return call.Nickname
		}
	}
	return ""
}

func codexSessionMetaEvent(sessionID, path string, rec codexRolloutRecord, meta codexSessionMeta, ts time.Time) trace.Event {
	summary := map[string]any{
		"provider":      "codex",
		"cwd":           meta.CWD,
		"originator":    meta.Originator,
		"cliVersion":    meta.CLIVersion,
		"source":        meta.Source,
		"threadSource":  meta.ThreadSource,
		"modelProvider": meta.ModelProvider,
	}
	if meta.Timestamp != "" {
		if parsed, err := time.Parse(time.RFC3339Nano, meta.Timestamp); err == nil {
			ts = parsed
		}
	}
	return trace.Event{
		ID:             importStableID(sessionID, rec.Index, "session_meta"),
		SessionID:      sessionID,
		Type:           trace.EventProcess,
		Title:          "Codex session started",
		Timestamp:      ts,
		Status:         trace.StatusOK,
		Source:         trace.SourceCodexLog,
		CorrelationIDs: []string{},
		Confidence:     trace.ConfidenceExact,
		Summary:        summary,
		RawRef:         &trace.RawRef{File: path, Offset: rec.Offset},
	}
}

func codexBasicEvent(sessionID, path string, rec codexRolloutRecord, eventType trace.EventType, title string, status trace.Status, summary map[string]any) trace.Event {
	return codexBasicEventWithIDs(sessionID, path, rec, eventType, title, status, []string{}, summary)
}

func codexBasicEventWithIDs(sessionID, path string, rec codexRolloutRecord, eventType trace.EventType, title string, status trace.Status, correlationIDs []string, summary map[string]any) trace.Event {
	if summary == nil {
		summary = map[string]any{}
	}
	summary["recordType"] = rec.Type
	summary["payloadType"] = rawFieldString(rec.Payload, "type")
	return trace.Event{
		ID:             importStableID(sessionID, rec.Index, eventType, title),
		SessionID:      sessionID,
		Type:           eventType,
		Title:          title,
		Timestamp:      codexRecordTime(rec),
		Status:         status,
		Source:         trace.SourceCodexLog,
		CorrelationIDs: compactStrings(correlationIDs),
		Confidence:     trace.ConfidenceExact,
		Summary:        summary,
		RawRef:         &trace.RawRef{File: path, Offset: rec.Offset},
	}
}

func codexRecordTime(rec codexRolloutRecord) time.Time {
	if rec.Timestamp != "" {
		if ts, err := time.Parse(time.RFC3339Nano, rec.Timestamp); err == nil {
			return ts
		}
	}
	return time.Unix(0, 0).UTC().Add(time.Duration(rec.Index) * time.Millisecond)
}

func rawFieldRaw(raw json.RawMessage, field string) json.RawMessage {
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(raw, &obj); err != nil {
		return nil
	}
	return obj[field]
}

func rawFieldString(raw json.RawMessage, field string) string {
	value := rawFieldRaw(raw, field)
	if len(value) == 0 {
		return ""
	}
	var text string
	if err := json.Unmarshal(value, &text); err == nil {
		return text
	}
	return strings.Trim(string(value), `"`)
}

func rawFieldAny(raw json.RawMessage, field string) any {
	value := rawFieldRaw(raw, field)
	if len(value) == 0 {
		return nil
	}
	var decoded any
	if err := json.Unmarshal(value, &decoded); err != nil {
		return nil
	}
	return decoded
}

func codexContentText(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var text string
	if err := json.Unmarshal(raw, &text); err == nil {
		return text
	}
	var blocks []map[string]any
	if err := json.Unmarshal(raw, &blocks); err != nil {
		return ""
	}
	var parts []string
	for _, block := range blocks {
		if text := stringFromAny(block["text"]); text != "" {
			parts = append(parts, text)
		}
	}
	return strings.Join(parts, "\n")
}

func decodeJSONText(text string) any {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}
	var decoded any
	if err := json.Unmarshal([]byte(text), &decoded); err == nil {
		return decoded
	}
	return truncate(text, 2000)
}

func decodeJSONObject(text string) map[string]any {
	var decoded map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(text)), &decoded); err != nil {
		return nil
	}
	return decoded
}

func stringFromAny(value any) string {
	if value == nil {
		return ""
	}
	if text, ok := value.(string); ok {
		return text
	}
	return fmt.Sprint(value)
}

func normalizeSkillKey(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func shortTitle(text, fallback string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return fallback
	}
	return truncate(text, 96)
}

func truncate(text string, limit int) string {
	if limit <= 0 {
		return text
	}
	runes := []rune(text)
	if len(runes) <= limit {
		return text
	}
	if limit <= 1 {
		return string(runes[:limit])
	}
	return string(runes[:limit-1]) + "..."
}

func compactStrings(values []string) []string {
	compacted := make([]string, 0, len(values))
	seen := map[string]struct{}{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		compacted = append(compacted, value)
	}
	return compacted
}

func sanitizeSessionID(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "session"
	}
	var builder strings.Builder
	previousUnderscore := false
	for _, r := range value {
		ok := (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')
		if ok {
			builder.WriteRune(r)
			previousUnderscore = false
			continue
		}
		if !previousUnderscore {
			builder.WriteByte('_')
			previousUnderscore = true
		}
	}
	result := strings.Trim(builder.String(), "_")
	if result == "" {
		return "session"
	}
	return result
}

func importStableID(parts ...any) string {
	items := make([]string, 0, len(parts))
	for _, part := range parts {
		items = append(items, fmt.Sprint(part))
	}
	return sanitizeSessionID(strings.Join(items, "_"))
}
