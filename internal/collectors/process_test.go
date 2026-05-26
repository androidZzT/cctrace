package collectors

import (
	"context"
	"testing"

	"github.com/agentz/cctrace/internal/trace"
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
