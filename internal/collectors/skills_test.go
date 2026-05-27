package collectors

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/agentz/cctrace/internal/trace"
)

func TestCollectSkillsParsesFrontmatter(t *testing.T) {
	root := t.TempDir()
	skillDir := filepath.Join(root, "plugins", "cache", "superpowers", "skills", "using-superpowers")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(skillDir, "SKILL.md")
	content := `---
name: using-superpowers
description: Use when starting any conversation
---

# Using Superpowers
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	events, err := CollectSkills("sess_1", []string{root})
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 {
		t.Fatalf("len(events) = %d, want 1", len(events))
	}
	event := events[0]
	if event.Type != trace.EventSkill || event.Title != "using-superpowers" {
		t.Fatalf("event = %#v", event)
	}
	if event.Summary["description"] != "Use when starting any conversation" {
		t.Fatalf("description = %#v", event.Summary["description"])
	}
	if event.Summary["phase"] != "startup" {
		t.Fatalf("phase = %#v", event.Summary["phase"])
	}
}
