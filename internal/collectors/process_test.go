package collectors

import (
	"context"
	"testing"

	"github.com/androidZzT/cctrace/internal/trace"
)

func TestRunProcessEmitsStartAndExit(t *testing.T) {
	collector := ProcessCollector{SessionID: "sess_1"}
	events, exitCode, err := collector.Run(context.Background(), []string{"sh", "-c", "printf hello"})
	if err != nil {
		t.Fatal(err)
	}
	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0", exitCode)
	}
	if len(events) != 2 {
		t.Fatalf("len(events) = %d, want 2", len(events))
	}
	if events[0].Type != trace.EventProcess || events[0].Status != trace.StatusRunning {
		t.Fatalf("start event = %#v", events[0])
	}
	if events[1].Type != trace.EventProcess || events[1].Status != trace.StatusOK {
		t.Fatalf("exit event = %#v", events[1])
	}
}

func TestStartProcessEmitsStartBeforeWait(t *testing.T) {
	collector := ProcessCollector{SessionID: "sess_1"}
	started, err := collector.Start(context.Background(), []string{"sh", "-c", "sleep 1"})
	if err != nil {
		t.Fatal(err)
	}
	defer started.Command.Process.Kill()

	if started.StartEvent.Type != trace.EventProcess || started.StartEvent.Status != trace.StatusRunning {
		t.Fatalf("StartEvent = %#v", started.StartEvent)
	}
	if started.StartEvent.Summary["command"] == nil {
		t.Fatalf("StartEvent summary missing command: %#v", started.StartEvent.Summary)
	}
}
