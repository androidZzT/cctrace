package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/androidZzT/cctrace/internal/trace"
)

func BenchmarkAppendEvent(b *testing.B) {
	for _, size := range []int{1_000, 10_000, 100_000} {
		b.Run(fmt.Sprintf("%d_events", size), func(b *testing.B) {
			root := b.TempDir()
			st := New(root)
			events := benchmarkEvents(size, "bench_append")
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				for _, event := range events {
					event.SessionID = fmt.Sprintf("bench_append_%d", i)
					if err := st.AppendEvent(event); err != nil {
						b.Fatal(err)
					}
				}
			}
		})
	}
}

func BenchmarkReadEvents(b *testing.B) {
	for _, size := range []int{1_000, 10_000, 100_000} {
		b.Run(fmt.Sprintf("%d_events", size), func(b *testing.B) {
			root := b.TempDir()
			sessionID := "bench_read"
			if err := writeBenchmarkEvents(root, sessionID, benchmarkEvents(size, sessionID)); err != nil {
				b.Fatal(err)
			}
			st := New(root)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				events, err := st.ReadEvents(sessionID)
				if err != nil {
					b.Fatal(err)
				}
				if len(events) != size {
					b.Fatalf("len(events) = %d, want %d", len(events), size)
				}
			}
		})
	}
}

func benchmarkEvents(count int, sessionID string) []trace.Event {
	events := make([]trace.Event, count)
	base := time.Unix(1_800_000_000, 0)
	for i := range events {
		events[i] = trace.Event{
			ID:             fmt.Sprintf("evt_%06d", i),
			SessionID:      sessionID,
			Type:           trace.EventToolCall,
			Title:          "benchmark event",
			Timestamp:      base.Add(time.Duration(i) * time.Millisecond),
			Status:         trace.StatusOK,
			Source:         trace.SourceDerived,
			CorrelationIDs: []string{fmt.Sprintf("call_%06d", i)},
			Confidence:     trace.ConfidenceExact,
			Summary:        map[string]any{"tool": "bench"},
		}
	}
	return events
}

func writeBenchmarkEvents(root, sessionID string, events []trace.Event) error {
	dir := filepath.Join(root, sessionID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	file, err := os.Create(filepath.Join(dir, "events.jsonl"))
	if err != nil {
		return err
	}
	defer file.Close()
	encoder := json.NewEncoder(file)
	for _, event := range events {
		if err := encoder.Encode(event); err != nil {
			return err
		}
	}
	return nil
}
