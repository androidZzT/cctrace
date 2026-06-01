package app

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/androidZzT/cctrace/internal/trace"
)

func TestRunPublishesStartupSkillEvents(t *testing.T) {
	home := t.TempDir()
	skillDir := filepath.Join(home, ".claude", "plugins", "cache", "superpowers", "skills", "using-superpowers")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatal(err)
	}
	content := `---
name: using-superpowers
description: Use when starting any conversation
---
`
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	dir := t.TempDir()
	cfg := Config{Provider: "claude", Command: []string{"sh", "-c", "sleep 1"}, StoreDir: dir, Addr: "127.0.0.1:43211", SkillRoots: []string{home}}
	done := make(chan error, 1)
	go func() { done <- Run(context.Background(), cfg) }()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		res, err := http.Get("http://127.0.0.1:43211/api/events")
		if err == nil {
			var events []trace.Event
			decodeErr := json.NewDecoder(res.Body).Decode(&events)
			_ = res.Body.Close()
			if decodeErr == nil {
				for _, event := range events {
					if event.Type == trace.EventSkill && event.Title == "using-superpowers" && event.Summary["description"] == "Use when starting any conversation" {
						return
					}
				}
			}
		}
		time.Sleep(25 * time.Millisecond)
	}
	t.Fatal("server did not expose startup skill metadata event")
}

func TestParseArgsForWrappedCommand(t *testing.T) {
	cfg, err := ParseArgs([]string{"claude", "--", "sh", "-c", "true"})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Provider != "claude" {
		t.Fatalf("Provider = %q", cfg.Provider)
	}
	if len(cfg.Command) != 3 || cfg.Command[0] != "sh" {
		t.Fatalf("Command = %#v", cfg.Command)
	}
}

func TestRunWrappedCommandCompletes(t *testing.T) {
	dir := t.TempDir()
	cfg := Config{Provider: "claude", Command: []string{"sh", "-c", "true"}, StoreDir: dir, Addr: "127.0.0.1:0"}
	if err := Run(context.Background(), cfg); err != nil {
		t.Fatal(err)
	}
}

func TestRunPublishesProcessStartWhileCommandRuns(t *testing.T) {
	dir := t.TempDir()
	cfg := Config{Provider: "claude", Command: []string{"sh", "-c", "sleep 1"}, StoreDir: dir, Addr: "127.0.0.1:43210"}
	done := make(chan error, 1)
	go func() { done <- Run(context.Background(), cfg) }()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		res, err := http.Get("http://127.0.0.1:43210/api/events")
		if err == nil {
			var events []trace.Event
			decodeErr := json.NewDecoder(res.Body).Decode(&events)
			_ = res.Body.Close()
			if decodeErr == nil {
				for _, event := range events {
					if event.Type == trace.EventProcess && event.Status == trace.StatusRunning {
						return
					}
				}
			}
		}
		time.Sleep(25 * time.Millisecond)
	}
	t.Fatal("server did not expose running process event while command was still running")
}

func TestParseArgsForView(t *testing.T) {
	cfg, err := ParseArgs([]string{"view", "sess_1"})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Provider != "view" {
		t.Fatalf("Provider = %q", cfg.Provider)
	}
	if len(cfg.Command) != 1 || cfg.Command[0] != "sess_1" {
		t.Fatalf("Command = %#v", cfg.Command)
	}
}
