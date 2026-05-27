package collectors

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/agentz/cctrace/internal/trace"
)

type ProcessCollector struct{ SessionID string }

type StartedProcess struct {
	Command    *exec.Cmd
	StartEvent trace.Event
	started    time.Time
}

func (c ProcessCollector) Start(ctx context.Context, command []string) (*StartedProcess, error) {
	if len(command) == 0 {
		return nil, fmt.Errorf("missing command")
	}
	started := time.Now()
	startEvent := trace.Event{
		ID:             "process_start_" + started.Format("150405.000000000"),
		SessionID:      c.SessionID,
		Type:           trace.EventProcess,
		Title:          "process started",
		Timestamp:      started,
		Status:         trace.StatusRunning,
		Source:         trace.SourceProcess,
		CorrelationIDs: []string{},
		Confidence:     trace.ConfidenceExact,
		Summary:        map[string]any{"command": command},
	}
	cmd := exec.CommandContext(ctx, command[0], command[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	return &StartedProcess{Command: cmd, StartEvent: startEvent, started: started}, nil
}

func (p *StartedProcess) Wait() (trace.Event, int, error) {
	err := p.Command.Wait()
	ended := time.Now()
	exitCode := 0
	status := trace.StatusOK
	if err != nil {
		status = trace.StatusError
		exitCode = 1
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		}
	}
	duration := ended.Sub(p.started).Milliseconds()
	exitEvent := trace.Event{
		ID:             "process_exit_" + ended.Format("150405.000000000"),
		SessionID:      p.StartEvent.SessionID,
		Type:           trace.EventProcess,
		Title:          "process exited",
		Timestamp:      ended,
		DurationMS:     &duration,
		Status:         status,
		Source:         trace.SourceProcess,
		CorrelationIDs: []string{},
		Confidence:     trace.ConfidenceExact,
		Summary:        map[string]any{"exitCode": float64(exitCode)},
	}
	return exitEvent, exitCode, err
}

func (c ProcessCollector) Run(ctx context.Context, command []string) ([]trace.Event, int, error) {
	started, err := c.Start(ctx, command)
	if err != nil {
		return nil, -1, err
	}
	exitEvent, exitCode, err := started.Wait()
	return []trace.Event{started.StartEvent, exitEvent}, exitCode, err
}
