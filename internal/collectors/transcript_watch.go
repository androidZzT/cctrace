package collectors

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/androidZzT/cctrace/internal/trace"
)

func NewestClaudeTranscript(home, cwd string) (string, error) {
	projectDir := filepath.Join(home, ".claude", "projects", encodeClaudeProjectPath(cwd))
	entries, err := os.ReadDir(projectDir)
	if err != nil {
		return "", err
	}
	var newest string
	var newestMod int64
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".jsonl" {
			continue
		}
		path := filepath.Join(projectDir, entry.Name())
		info, err := entry.Info()
		if err != nil {
			return "", err
		}
		mod := info.ModTime().UnixNano()
		if newest == "" || mod > newestMod {
			newest = path
			newestMod = mod
		}
	}
	if newest == "" {
		return "", fmt.Errorf("no Claude transcript JSONL files in %s", projectDir)
	}
	return newest, nil
}

func WatchTranscript(ctx context.Context, sessionID, provider, path string, events chan<- trace.Event) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	var offset int64
	buf := make([]byte, 0, 4096)
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}
		chunk := make([]byte, 4096)
		n, err := file.Read(chunk)
		if n > 0 {
			buf = append(buf, chunk[:n]...)
			for {
				idx := strings.IndexByte(string(buf), '\n')
				if idx < 0 {
					break
				}
				line := append([]byte(nil), buf[:idx+1]...)
				parsed, parseErr := ParseTranscriptJSONL(sessionID, provider, path, strings.NewReader(string(line)))
				if parseErr != nil {
					return parseErr
				}
				for _, event := range parsed {
					if event.RawRef != nil {
						event.RawRef.Offset = offset
					}
					events <- event
				}
				offset += int64(len(line))
				buf = buf[idx+1:]
			}
		}
		if err != nil && err != io.EOF {
			return err
		}
		time.Sleep(50 * time.Millisecond)
	}
}

func encodeClaudeProjectPath(cwd string) string {
	encoded := strings.ReplaceAll(cwd, string(filepath.Separator), "-")
	return strings.ReplaceAll(encoded, ".", "-")
}
