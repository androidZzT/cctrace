package app

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/agentz/cctrace/internal/collectors"
	"github.com/agentz/cctrace/internal/server"
	"github.com/agentz/cctrace/internal/store"
	"github.com/agentz/cctrace/internal/trace"
)

type Config struct {
	Provider string
	Command  []string
	StoreDir string
	Addr     string
}

func ParseArgs(args []string) (Config, error) {
	if len(args) >= 2 && args[0] == "view" {
		return Config{Provider: "view", Command: []string{args[1]}}, nil
	}
	if len(args) < 3 {
		return Config{}, fmt.Errorf("usage: cctrace claude|codex -- <command>")
	}
	provider := args[0]
	if provider != "claude" && provider != "codex" {
		return Config{}, fmt.Errorf("unsupported provider %q", provider)
	}
	if args[1] != "--" {
		return Config{}, fmt.Errorf("expected -- before command")
	}
	return Config{Provider: provider, Command: args[2:]}, nil
}

func Run(ctx context.Context, cfg Config) error {
	if cfg.StoreDir == "" {
		cfg.StoreDir = filepath.Join(os.TempDir(), "cctrace-sessions")
	}
	if cfg.Addr == "" {
		cfg.Addr = "127.0.0.1:43177"
	}
	if cfg.Provider == "view" {
		hub := server.NewHub()
		st := store.New(cfg.StoreDir)
		events, err := st.ReadEvents(cfg.Command[0])
		if err != nil {
			return err
		}
		for _, event := range events {
			hub.Publish(event)
		}
		return server.Listen(cfg.Addr, hub)
	}
	sessionID := "sess_" + strings.NewReplacer(".", "", "-", "").Replace(time.Now().Format("20060102_150405.000000000"))
	hub := server.NewHub()
	st := store.New(cfg.StoreDir)
	session := trace.Session{ID: sessionID, Command: cfg.Command, StartedAt: time.Now()}
	if err := st.CreateSession(session); err != nil {
		return err
	}

	addr := cfg.Addr
	if strings.HasSuffix(addr, ":0") {
		ln, err := net.Listen("tcp", addr)
		if err != nil {
			return err
		}
		addr = ln.Addr().String()
		_ = ln.Close()
	}
	go func() { _ = server.Listen(addr, hub) }()

	collector := collectors.ProcessCollector{SessionID: sessionID}
	events, _, err := collector.Run(ctx, cfg.Command)
	for _, event := range events {
		hub.Publish(event)
		_ = st.AppendEvent(event)
	}
	return err
}
