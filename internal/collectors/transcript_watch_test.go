package collectors

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestEncodeClaudeProjectPath(t *testing.T) {
	cwd := "/Users/agentz/Workspace/cctrace/.worktrees/cctrace-mvp"
	want := "-Users-agentz-Workspace-cctrace--worktrees-cctrace-mvp"
	if got := encodeClaudeProjectPath(cwd); got != want {
		t.Fatalf("encodeClaudeProjectPath() = %q, want %q", got, want)
	}
}

func TestNewestClaudeTranscriptForCWD(t *testing.T) {
	root := t.TempDir()
	cwd := filepath.Join(root, "workspace", "cctrace")
	projectDir := filepath.Join(root, ".claude", "projects", encodeClaudeProjectPath(cwd))
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatal(err)
	}
	oldPath := filepath.Join(projectDir, "old.jsonl")
	newPath := filepath.Join(projectDir, "new.jsonl")
	if err := os.WriteFile(oldPath, []byte("{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(newPath, []byte("{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	oldTime := time.Now().Add(-time.Hour)
	newTime := time.Now()
	if err := os.Chtimes(oldPath, oldTime, oldTime); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(newPath, newTime, newTime); err != nil {
		t.Fatal(err)
	}

	got, err := NewestClaudeTranscript(root, cwd)
	if err != nil {
		t.Fatal(err)
	}
	if got != newPath {
		t.Fatalf("NewestClaudeTranscript() = %q, want %q", got, newPath)
	}
}
