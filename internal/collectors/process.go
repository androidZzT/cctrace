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

func (c ProcessCollector) Run(ctx context.Context, command []string) ([]trace.Event, int, error) {
	if len(command) == 0 {
		return nil, -1, fmt.Errorf("missing command")
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
	err := cmd.Run()
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
	duration := ended.Sub(started).Milliseconds()
	exitEvent := trace.Event{
		ID:             "process_exit_" + ended.Format("150405.000000000"),
		SessionID:      c.SessionID,
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
	return []trace.Event{startEvent, exitEvent}, exitCode, nil
}
