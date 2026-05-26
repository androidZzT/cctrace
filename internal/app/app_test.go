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
