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
	Provider   string
	Command    []string
	StoreDir   string
	Addr       string
	SkillRoots []string
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
		cfg.Addr = "127.0.0.1:43179"
	}
	if cfg.Provider == "view" {
		hub := server.NewHub()
		hub.SetSession(server.SessionMetadata{ID: cfg.Command[0], Provider: "view", Mode: "history", Command: cfg.Command})
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
	hub.SetSession(server.SessionMetadata{ID: sessionID, Provider: cfg.Provider, Mode: "live", Command: cfg.Command})
	st := store.New(cfg.StoreDir)
	session := trace.Session{ID: sessionID, Command: cfg.Command, StartedAt: time.Now()}
	if err := st.CreateSession(session); err != nil {
		return err
	}
	publishStartupSkills(hub, st, sessionID, cfg.SkillRoots)

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
	started, err := collector.Start(ctx, cfg.Command)
	if err != nil {
		return err
	}
	publish(hub, st, started.StartEvent)

	watchCtx, cancelWatch := context.WithCancel(ctx)
	defer cancelWatch()
	if cfg.Provider == "claude" {
		go watchClaudeTranscript(watchCtx, sessionID, hub, st)
	}

	exitEvent, _, waitErr := started.Wait()
	cancelWatch()
	publish(hub, st, exitEvent)
	return waitErr
}

func watchClaudeTranscript(ctx context.Context, sessionID string, hub *server.Hub, st *store.Store) {
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}
	cwd, err := os.Getwd()
	if err != nil {
		return
	}
	path, err := waitForClaudeTranscript(ctx, home, cwd)
	if err != nil {
		return
	}
	events := make(chan trace.Event, 32)
	go func() { _ = collectors.WatchTranscript(ctx, sessionID, "claude", path, events) }()
	for {
		select {
		case <-ctx.Done():
			return
		case event := <-events:
			publish(hub, st, event)
		}
	}
}

func waitForClaudeTranscript(ctx context.Context, home, cwd string) (string, error) {
	for {
		path, err := collectors.NewestClaudeTranscript(home, cwd)
		if err == nil {
			return path, nil
		}
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(100 * time.Millisecond):
		}
	}
}

func publishStartupSkills(hub *server.Hub, st *store.Store, sessionID string, roots []string) {
	if len(roots) == 0 {
		home, err := os.UserHomeDir()
		if err != nil {
			return
		}
		roots = collectors.DefaultSkillRoots(home)
	}
	events, err := collectors.CollectSkills(sessionID, roots)
	if err != nil {
		return
	}
	for _, event := range events {
		publish(hub, st, event)
	}
}

func publish(hub *server.Hub, st *store.Store, event trace.Event) {
	hub.Publish(event)
	_ = st.AppendEvent(event)
}
