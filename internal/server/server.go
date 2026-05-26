package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"github.com/agentz/cctrace/internal/trace"
)

const indexHTML = `<!doctype html>
<html>
<head>
  <meta charset="utf-8">
  <title>cctrace</title>
  <style>
    body { margin: 0; font-family: ui-sans-serif, system-ui; background: #0f172a; color: #e2e8f0; }
    header { padding: 16px 20px; border-bottom: 1px solid #334155; }
    main { display: grid; grid-template-columns: 260px 1fr 320px; height: calc(100vh - 57px); }
    aside, section { padding: 16px; overflow: auto; border-right: 1px solid #334155; }
    .event { margin: 8px 0; padding: 8px; border-radius: 8px; background: #1e293b; cursor: pointer; }
    .bar { margin: 8px 0; padding: 8px; border-radius: 8px; background: #2563eb; min-width: 120px; }
    .api_request, .api_response { background: #16a34a; }
    .tool_call, .tool_result { background: #ca8a04; }
    .error { background: #dc2626; }
    pre { white-space: pre-wrap; word-break: break-word; }
  </style>
</head>
<body>
<header><strong>cctrace</strong> real-time agent profiler</header>
<main>
  <aside><h3>Trace Tree</h3><div id="tree"></div></aside>
  <section><h3>Waterfall Timeline</h3><div id="timeline"></div></section>
  <section><h3>Details</h3><pre id="details">Select an event</pre></section>
</main>
<script>
async function loadEvents() {
  const res = await fetch('/api/events');
  const events = await res.json();
  const tree = document.getElementById('tree');
  const timeline = document.getElementById('timeline');
  tree.innerHTML = '';
  timeline.innerHTML = '';
  events.forEach((event) => {
    const item = document.createElement('div');
    item.className = 'event';
    item.textContent = event.type + ': ' + event.title;
    item.onclick = () => show(event);
    tree.appendChild(item);
    const bar = document.createElement('div');
    bar.className = 'bar ' + event.type;
    const width = Math.max(120, Number(event.durationMs || 100));
    bar.style.width = Math.min(width, 900) + 'px';
    bar.textContent = event.title + ' · ' + event.status + ' · ' + event.confidence;
    bar.onclick = () => show(event);
    timeline.appendChild(bar);
  });
}
function show(event) { document.getElementById('details').textContent = JSON.stringify(event, null, 2); }
loadEvents();
setInterval(loadEvents, 1000);
</script>
</body>
</html>`

type Hub struct {
	mu     sync.Mutex
	events []trace.Event
}

func NewHub() *Hub { return &Hub{} }

func (h *Hub) Publish(event trace.Event) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.events = append(h.events, event)
}

func (h *Hub) Events() []trace.Event {
	h.mu.Lock()
	defer h.mu.Unlock()
	return append([]trace.Event(nil), h.events...)
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
	return mux
}

func Listen(addr string, hub *Hub) error {
	fmt.Printf("cctrace UI: http://%s\n", addr)
	return http.ListenAndServe(addr, NewHandler(hub))
}
