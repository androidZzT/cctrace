package correlate

import (
	"fmt"
	"testing"
	"time"

	"github.com/androidZzT/cctrace/internal/trace"
)

func BenchmarkCorrelate(b *testing.B) {
	for _, size := range []int{1_000, 10_000, 100_000} {
		b.Run(fmt.Sprintf("%d_events", size), func(b *testing.B) {
			events := benchmarkEvents(size)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				out := Correlate(events, 250*time.Millisecond)
				if len(out) != len(events) {
					b.Fatalf("len(out) = %d, want %d", len(out), len(events))
				}
			}
		})
	}
}

func benchmarkEvents(count int) []trace.Event {
	events := make([]trace.Event, count)
	base := time.Unix(1_800_000_000, 0)
	types := []trace.EventType{
		trace.EventToolCall,
		trace.EventAgentTurn,
		trace.EventSkill,
		trace.EventSubagent,
		trace.EventAPIRequest,
		trace.EventAPIResponse,
	}
	for i := range events {
		events[i] = trace.Event{
			ID:             fmt.Sprintf("evt_%06d", i),
			SessionID:      "bench",
			Type:           types[i%len(types)],
			Title:          "benchmark event",
			Timestamp:      base.Add(time.Duration(i) * time.Millisecond),
			Status:         trace.StatusOK,
			Source:         trace.SourceDerived,
			CorrelationIDs: []string{},
			Confidence:     trace.ConfidenceExact,
			Summary:        map[string]any{},
		}
	}
	return events
}
