package collectors

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/androidZzT/cctrace/internal/trace"
)

func TestWatchTranscriptPublishesAppendedEvents(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "session.jsonl")
	if err := os.WriteFile(path, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	events := make(chan trace.Event, 4)
	done := make(chan error, 1)
	go func() { done <- WatchTranscript(ctx, "sess_1", "claude", path, events) }()

	line := `{"type":"user","uuid":"user_1","timestamp":"2026-05-27T01:43:39.055Z","message":{"role":"user","content":"hi"}}` + "\n"
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.WriteString(line); err != nil {
		t.Fatal(err)
	}
	_ = file.Close()

	select {
	case event := <-events:
		if event.Type != trace.EventUserMessage || event.Title != "hi" {
			t.Fatalf("event = %#v", event)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for transcript event")
	}
	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("watcher did not stop after context cancellation")
	}
}

func TestWatchTranscriptBuffersPartialLines(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "session.jsonl")
	if err := os.WriteFile(path, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	events := make(chan trace.Event, 4)
	done := make(chan error, 1)
	go func() { done <- WatchTranscript(ctx, "sess_1", "claude", path, events) }()

	file, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	partial := `{"type":"user","uuid":"user_1","timestamp":"2026-05-27T01:43:39.055Z","message":{"role":"user","content":"hi"}}`
	if _, err := file.WriteString(partial); err != nil {
		t.Fatal(err)
	}

	select {
	case event := <-events:
		t.Fatalf("published event before newline: %#v", event)
	case <-time.After(150 * time.Millisecond):
	}

	if _, err := file.WriteString("\n"); err != nil {
		t.Fatal(err)
	}
	_ = file.Close()

	select {
	case event := <-events:
		if event.Type != trace.EventUserMessage || event.Title != "hi" {
			t.Fatalf("event = %#v", event)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for completed transcript line")
	}
	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("watcher did not stop after context cancellation")
	}
}
