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
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(session, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "session.json"), data, 0o644)
}

func (s *Store) AppendEvent(event trace.Event) error {
	dir := s.sessionDir(event.SessionID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	file, err := os.OpenFile(filepath.Join(dir, "events.jsonl"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	defer file.Close()
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}
	if _, err := file.Write(append(data, '\n')); err != nil {
		return err
	}
	return nil
}

func (s *Store) ReadEvents(sessionID string) ([]trace.Event, error) {
	file, err := os.Open(filepath.Join(s.sessionDir(sessionID), "events.jsonl"))
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var events []trace.Event
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 1024*1024), 64*1024*1024)
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
