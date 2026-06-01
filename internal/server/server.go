package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"

	"github.com/androidZzT/cctrace/internal/trace"
)

type SessionMetadata struct {
	ID       string   `json:"id"`
	Provider string   `json:"provider"`
	Mode     string   `json:"mode"`
	Command  []string `json:"command"`
}

type Hub struct {
	mu          sync.Mutex
	events      []trace.Event
	session     SessionMetadata
	subscribers map[chan trace.Event]struct{}
}

func NewHub() *Hub { return &Hub{subscribers: map[chan trace.Event]struct{}{}} }

func (h *Hub) Publish(event trace.Event) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.events = append(h.events, event)
	for subscriber := range h.subscribers {
		select {
		case subscriber <- event:
		default:
		}
	}
}

func (h *Hub) Subscribe() chan trace.Event {
	subscriber := make(chan trace.Event, 16)
	h.mu.Lock()
	h.subscribers[subscriber] = struct{}{}
	h.mu.Unlock()
	return subscriber
}

func (h *Hub) Unsubscribe(subscriber chan trace.Event) {
	h.mu.Lock()
	delete(h.subscribers, subscriber)
	close(subscriber)
	h.mu.Unlock()
}

func (h *Hub) Events() []trace.Event {
	h.mu.Lock()
	defer h.mu.Unlock()
	return append([]trace.Event(nil), h.events...)
}

func (h *Hub) SetSession(session SessionMetadata) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.session = session
}

func (h *Hub) Replace(session SessionMetadata, events []trace.Event) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.session = session
	h.events = append([]trace.Event(nil), events...)
}

func (h *Hub) Session() SessionMetadata {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.session
}

func NewHandler(hub *Hub) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(indexHTML))
	})
	mux.HandleFunc("/api/events", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(hub.Events())
	})
	mux.HandleFunc("/api/events/stream", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "streaming unsupported", http.StatusInternalServerError)
			return
		}
		_, _ = fmt.Fprint(w, ": connected\n\n")
		flusher.Flush()
		subscriber := hub.Subscribe()
		defer hub.Unsubscribe(subscriber)
		for {
			select {
			case <-r.Context().Done():
				return
			case event := <-subscriber:
				data, err := json.Marshal(event)
				if err != nil {
					continue
				}
				_, _ = fmt.Fprintf(w, "data: %s\n\n", data)
				flusher.Flush()
			}
		}
	})
	mux.HandleFunc("/api/session", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(hub.Session())
	})
	mux.HandleFunc("/api/import", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		var req SessionImportRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, http.StatusBadRequest, err.Error())
			return
		}
		session, events, source, err := ImportSessionPath(req.Path)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, err.Error())
			return
		}
		hub.Replace(session, events)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(SessionImportResponse{Session: session, EventCount: len(events), Source: source})
	})
	mux.HandleFunc("/api/import/upload", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, 512<<20)
		reader, err := r.MultipartReader()
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, err.Error())
			return
		}
		tmpDir, err := os.MkdirTemp("", "cctrace-upload-*")
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, err.Error())
			return
		}
		defer os.RemoveAll(tmpDir)
		displayByPath := map[string]string{}
		seenNames := map[string]int{}
		var paths []string
		for {
			part, err := reader.NextPart()
			if err == io.EOF {
				break
			}
			if err != nil {
				writeJSONError(w, http.StatusBadRequest, err.Error())
				return
			}
			if part.FormName() != "file" {
				continue
			}
			filename := part.FileName()
			if filename == "" {
				writeJSONError(w, http.StatusBadRequest, "uploaded file is missing a name")
				return
			}
			safeName := safeUploadFilename(filename)
			if seenNames[safeName] > 0 {
				safeName = fmt.Sprintf("%03d_%s", seenNames[safeName]+1, safeName)
			}
			seenNames[safeUploadFilename(filename)]++
			tmpPath := filepath.Join(tmpDir, safeName)
			out, err := os.Create(tmpPath)
			if err != nil {
				writeJSONError(w, http.StatusInternalServerError, err.Error())
				return
			}
			_, copyErr := io.Copy(out, part)
			closeErr := out.Close()
			if copyErr != nil {
				writeJSONError(w, http.StatusBadRequest, copyErr.Error())
				return
			}
			if closeErr != nil {
				writeJSONError(w, http.StatusInternalServerError, closeErr.Error())
				return
			}
			paths = append(paths, tmpPath)
			displayByPath[tmpPath] = "uploaded:" + filename
		}
		if len(paths) == 0 {
			writeJSONError(w, http.StatusBadRequest, "file is required")
			return
		}
		session, events, source, err := ImportUploadedPaths(paths, displayByPath)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, err.Error())
			return
		}
		hub.Replace(session, events)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(SessionImportResponse{Session: session, EventCount: len(events), Source: source})
	})
	return mux
}

func writeJSONError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}

func Listen(addr string, hub *Hub) error {
	fmt.Printf("cctrace UI: http://%s\n", addr)
	return http.ListenAndServe(addr, NewHandler(hub))
}
