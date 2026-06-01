package collectors

import (
	"bytes"
	"fmt"
	"testing"
)

func BenchmarkParseTranscriptJSONL(b *testing.B) {
	for _, size := range []int{1_000, 10_000, 100_000} {
		b.Run(fmt.Sprintf("%d_records", size), func(b *testing.B) {
			data := benchmarkTranscript(size)
			b.SetBytes(int64(len(data)))
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				events, err := ParseTranscriptJSONL("bench", "claude", "bench.jsonl", bytes.NewReader(data))
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

func benchmarkTranscript(count int) []byte {
	var buf bytes.Buffer
	for i := 0; i < count; i++ {
		if i%2 == 0 {
			fmt.Fprintf(&buf, `{"type":"assistant","uuid":"assistant_%06d","timestamp":"2026-05-27T01:43:43Z","message":{"role":"assistant","content":[{"type":"tool_use","id":"toolu_%06d","name":"Read","input":{"file_path":"/tmp/file_%06d.md"}}]}}`+"\n", i, i, i)
			continue
		}
		fmt.Fprintf(&buf, `{"type":"user","uuid":"result_%06d","timestamp":"2026-05-27T01:43:44Z","message":{"role":"user","content":[{"tool_use_id":"toolu_%06d","type":"tool_result","content":[{"type":"text","text":"contents %06d"}]}]}}`+"\n", i, i-1, i)
	}
	return buf.Bytes()
}
