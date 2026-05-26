package correlate

import (
	"slices"
	"time"

	"github.com/agentz/cctrace/internal/trace"
)

func Correlate(events []trace.Event, window time.Duration) []trace.Event {
	out := append([]trace.Event(nil), events...)
	slices.SortFunc(out, func(a, b trace.Event) int { return a.Timestamp.Compare(b.Timestamp) })
	for i := range out {
		if out[i].Type != trace.EventAPIRequest && out[i].Type != trace.EventAPIResponse {
			continue
		}
		parent := nearestParent(out, i, window)
		if parent == "" {
			out[i].Confidence = trace.ConfidenceUnknown
			continue
		}
		out[i].ParentID = parent
		out[i].Confidence = trace.ConfidenceLikely
	}
	return out
}

func nearestParent(events []trace.Event, index int, window time.Duration) string {
	candidateTypes := map[trace.EventType]bool{trace.EventToolCall: true, trace.EventAgentTurn: true, trace.EventSkill: true, trace.EventSubagent: true}
	var bestID string
	bestDelta := window + time.Nanosecond
	for i := range events {
		if i == index || !candidateTypes[events[i].Type] {
			continue
		}
		delta := events[index].Timestamp.Sub(events[i].Timestamp)
		if delta < 0 {
			delta = -delta
		}
		if delta <= window && delta < bestDelta {
			bestDelta = delta
			bestID = events[i].ID
		}
	}
	return bestID
}
