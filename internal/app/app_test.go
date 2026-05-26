package app

import (
	"context"
	"testing"
)

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
