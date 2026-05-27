# Perfetto Viewport Interaction Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace cctrace's slow scroll-width timeline with a Perfetto-style viewport interaction model that stays usable on large Claude sessions.

**Architecture:** Keep `internal/server/server.go` as the single HTML/JS delivery point for this MVP. Frontend state changes from scroll position to `viewportStart`/`viewportEnd`; event data is normalized once into render items; overview rendering moves from per-event DOM nodes to a canvas density pass.

**Tech Stack:** Go `net/http` server tests, embedded vanilla HTML/CSS/JavaScript, browser DOM and Canvas APIs.

---

## File Structure

- Modify `internal/server/server_test.go`: add regression tests for viewport architecture, canvas overview, bounded labels, and removal of per-event overview DOM rendering.
- Modify `internal/server/server.go`: update embedded CSS/HTML/JS to implement viewport pan/zoom, canvas overview, drag interactions, and bounded render labels.

---

### Task 1: Add viewport and canvas overview regression test

**Files:**
- Modify: `internal/server/server_test.go`
- Test: `internal/server/server_test.go`

- [ ] **Step 1: Write the failing test**

Add this test to `internal/server/server_test.go`:

```go
func TestIndexUsesPerfettoViewportCanvasOverviewAndBoundedLabels(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	res := httptest.NewRecorder()
	NewHandler(NewHub()).ServeHTTP(res, req)
	body := res.Body.String()
	for _, want := range []string{
		"<canvas class=\"overview-canvas\" id=\"overview-canvas\"></canvas>",
		"let viewportStart = null",
		"let viewportEnd = null",
		"function buildRenderModel(events)",
		"function shortText(value, maxLength)",
		"function initializeViewport(renderItems)",
		"function panViewport(deltaMs)",
		"function zoomViewportAt(clientX, nextSpanMs)",
		"function drawOverviewCanvas(items, sessionStart, sessionEnd)",
		"function renderOverviewWindow(sessionStart, sessionEnd)",
		"function startTimelinePan(event)",
		"timelineShell.addEventListener('pointerdown', startTimelinePan)",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("index HTML missing %q", want)
		}
	}
	for _, forbidden := range []string{
		"function renderOverviewEvent(event, sessionStart, sessionEnd)",
		"events.forEach((event) => renderOverviewEvent(event, sessionStart, sessionEnd))",
		"overview-mini-event",
		"const width = Math.max(1400, (rangeEnd - rangeStart + 5000) * pxPerMs())",
	} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("index HTML still contains slow path %q", forbidden)
		}
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run:

```bash
go test ./internal/server -run TestIndexUsesPerfettoViewportCanvasOverviewAndBoundedLabels -count=1
```

Expected: FAIL because the current HTML still uses `.overview-mini-event`, `renderOverviewEvent`, and lacks the new viewport functions.

- [ ] **Step 3: Commit not required**

Do not commit unless the user explicitly asks.

---

### Task 2: Implement Perfetto-style viewport timeline

**Files:**
- Modify: `internal/server/server.go`
- Test: `internal/server/server_test.go`

- [ ] **Step 1: Replace overview DOM with canvas element**

Change the overview markup to:

```html
<div class="overview-shell">
  <canvas class="overview-canvas" id="overview-canvas"></canvas><div class="overview-window" id="overview-window"></div>
</div>
```

Update CSS so `.overview-canvas` is a block canvas and `.overview-window` overlays it:

```css
.overview-shell { position: relative; height: 72px; padding: 10px 16px; border-bottom: 1px solid var(--line); background: #020617; }
.overview-canvas { display: block; width: 100%; height: 50px; border: 1px solid var(--line); border-radius: 10px; background: #0b1220; cursor: crosshair; }
.overview-window { position: absolute; top: 10px; bottom: 10px; border: 2px solid #38bdf8; border-radius: 10px; background: rgba(56,189,248,.12); pointer-events: none; }
```

- [ ] **Step 2: Add render model helpers**

Replace raw event rendering state with render model state:

```js
const minViewportMs = 1000;
const maxInitialViewportMs = 5 * 60 * 1000;
let currentFilter = 'all';
let selectedId = '';
let allEvents = [];
let renderItems = [];
let sessionMetadata = { mode: 'live', provider: 'unknown', id: '' };
let traceStart = Date.now();
let traceEnd = traceStart + 10000;
let viewportStart = null;
let viewportEnd = null;
let overviewDragStart = null;
let overviewDragMode = '';
let overviewMoveAnchor = 0;
let timelinePanAnchor = null;
let timelinePanViewportStart = 0;
let timelinePanViewportEnd = 0;
```

Add:

```js
function shortText(value, maxLength) {
  const text = String(value || '');
  return text.length > maxLength ? text.slice(0, maxLength - 1) + '…' : text;
}

function buildRenderModel(events) {
  return events.map((event) => {
    const start = new Date(event.timestamp).getTime();
    const end = start + Number(event.durationMs || 0);
    const lane = agentLaneForEvent(event);
    return { event, start, end, laneKey: agentLaneKey(event), laneLabel: lane.label, shortTitle: shortText(event.title || event.type, 160) };
  });
}
```

- [ ] **Step 3: Add viewport helpers**

Add:

```js
function initializeViewport(items) {
  traceStart = items.length ? Math.min(...items.map((item) => item.start)) : Date.now();
  traceEnd = items.length ? Math.max(...items.map((item) => item.end)) : traceStart + 10000;
  if (viewportStart == null || viewportEnd == null) {
    viewportStart = Math.max(traceStart, traceEnd - maxInitialViewportMs);
    viewportEnd = traceEnd;
  }
  clampViewport();
}

function viewportSpan() {
  return Math.max(minViewportMs, viewportEnd - viewportStart);
}

function clampViewport() {
  const span = viewportSpan();
  if (viewportStart < traceStart) {
    viewportStart = traceStart;
    viewportEnd = traceStart + span;
  }
  if (viewportEnd > traceEnd) {
    viewportEnd = traceEnd;
    viewportStart = traceEnd - span;
  }
  if (viewportStart < traceStart) viewportStart = traceStart;
  if (viewportEnd > traceEnd) viewportEnd = traceEnd;
}

function panViewport(deltaMs) {
  viewportStart += deltaMs;
  viewportEnd += deltaMs;
  clampViewport();
  render();
}

function setViewport(startMs, endMs) {
  viewportStart = Math.min(startMs, endMs);
  viewportEnd = Math.max(startMs, endMs);
  if (viewportEnd - viewportStart < minViewportMs) viewportEnd = viewportStart + minViewportMs;
  clampViewport();
  render();
}

function timestampFromClientX(clientX) {
  const shell = document.querySelector('.timeline-shell');
  const rect = shell.getBoundingClientRect();
  const ratio = Math.max(0, Math.min(1, (clientX - rect.left - 160) / Math.max(1, rect.width - 160)));
  return viewportStart + ratio * viewportSpan();
}

function zoomViewportAt(clientX, nextSpanMs) {
  const anchor = timestampFromClientX(clientX);
  const ratio = (anchor - viewportStart) / viewportSpan();
  const span = Math.max(minViewportMs, Math.min(traceEnd - traceStart || minViewportMs, nextSpanMs));
  setViewport(anchor - span * ratio, anchor + span * (1 - ratio));
}
```

- [ ] **Step 4: Render only viewport items and canvas overview**

Update `render()` to use `renderItems`, filter by viewport, and avoid scroll-width sizing:

```js
function render() {
  initializeViewport(renderItems);
  const filteredItems = renderItems.filter((item) => filterMatches(item.event));
  const visibleItems = filteredItems.filter((item) => item.end >= viewportStart && item.start <= viewportEnd);
  const width = Math.max(1000, document.querySelector('.timeline-shell').clientWidth - 160);
  const agentLanes = discoverAgentLanes(visibleItems.map((item) => item.event));
  drawOverviewCanvas(filteredItems, traceStart, traceEnd);
  renderOverviewWindow(traceStart, traceEnd);
  renderLanes(agentLanes);
  renderRuler(width, viewportSpan(), viewportStart);
  agentLanes.forEach((lane) => renderTrack(lane, width));
  visibleItems.forEach((item) => positionEventNode(item, width));
}
```

Add canvas overview:

```js
function drawOverviewCanvas(items, sessionStart, sessionEnd) {
  const canvas = document.getElementById('overview-canvas');
  const rect = canvas.getBoundingClientRect();
  const dpr = window.devicePixelRatio || 1;
  const width = Math.max(1, Math.floor(rect.width));
  const height = Math.max(1, Math.floor(rect.height));
  if (canvas.width !== width * dpr || canvas.height !== height * dpr) {
    canvas.width = width * dpr;
    canvas.height = height * dpr;
  }
  const ctx = canvas.getContext('2d');
  ctx.setTransform(dpr, 0, 0, dpr, 0, 0);
  ctx.clearRect(0, 0, width, height);
  ctx.fillStyle = '#0b1220';
  ctx.fillRect(0, 0, width, height);
  const span = Math.max(1, sessionEnd - sessionStart);
  const buckets = new Array(width).fill(0);
  items.forEach((item) => {
    const x = Math.max(0, Math.min(width - 1, Math.floor((item.start - sessionStart) / span * width)));
    buckets[x]++;
  });
  const max = Math.max(1, ...buckets);
  ctx.fillStyle = '#38bdf8';
  buckets.forEach((count, x) => {
    if (!count) return;
    const barHeight = Math.max(2, count / max * (height - 8));
    ctx.fillRect(x, height - barHeight - 4, 1, barHeight);
  });
}

function renderOverviewWindow(sessionStart, sessionEnd) {
  const canvas = document.getElementById('overview-canvas');
  const windowNode = document.getElementById('overview-window');
  const rect = canvas.getBoundingClientRect();
  const span = Math.max(1, sessionEnd - sessionStart);
  windowNode.style.left = (16 + (viewportStart - sessionStart) / span * rect.width) + 'px';
  windowNode.style.width = Math.max(8, (viewportEnd - viewportStart) / span * rect.width) + 'px';
}
```

- [ ] **Step 5: Update ruler and event positioning**

Keep `renderRuler(width, spanMs, sessionStart)`, but choose an adaptive step:

```js
const stepMs = spanMs > 10 * 60 * 1000 ? 60 * 1000 : spanMs > 60 * 1000 ? 10 * 1000 : 1000;
```

Change event positioning to receive render items:

```js
function positionEventNode(item, trackWidth) {
  const track = document.querySelector('[data-lane="' + item.laneKey + '"]');
  if (!track) return;
  const event = item.event;
  const duration = Math.max(0, item.end - item.start);
  const left = (item.start - viewportStart) / viewportSpan() * trackWidth;
  const width = duration > 0 ? Math.max(56, duration / viewportSpan() * trackWidth) : 0;
  const node = document.createElement('div');
  node.className = (width > 0 ? 'event-block ' : 'event-marker ') + event.type + (event.id === selectedId ? ' selected' : '');
  node.style.left = left + 'px';
  if (width > 0) node.style.width = Math.min(width, 900) + 'px';
  node.title = typeBadge(event.type) + ': ' + item.shortTitle;
  node.onclick = () => show(event);
  const label = document.createElement('span');
  label.textContent = typeBadge(event.type) + ' · ' + item.shortTitle;
  node.appendChild(label);
  track.appendChild(node);
}
```

- [ ] **Step 6: Implement Perfetto-style interactions**

Update wheel/pinch handlers:

```js
timelineShell.addEventListener('wheel', (event) => {
  event.preventDefault();
  if (event.ctrlKey || event.metaKey) {
    zoomViewportAt(event.clientX, viewportSpan() * (event.deltaY < 0 ? 0.8 : 1.25));
    return;
  }
  const delta = Math.abs(event.deltaX) > Math.abs(event.deltaY) ? event.deltaX : event.deltaY;
  panViewport(delta / Math.max(1, timelineShell.clientWidth - 160) * viewportSpan());
}, { passive: false });

window.addEventListener('gesturestart', (event) => event.preventDefault());
window.addEventListener('gesturechange', (event) => {
  event.preventDefault();
  zoomViewportAt(window.innerWidth / 2, viewportSpan() / Math.max(0.1, event.scale));
});
```

Add timeline drag:

```js
function startTimelinePan(event) {
  if (event.target.closest('.event-block, .event-marker')) return;
  timelinePanAnchor = event.clientX;
  timelinePanViewportStart = viewportStart;
  timelinePanViewportEnd = viewportEnd;
}

function updateTimelinePan(event) {
  if (timelinePanAnchor == null) return;
  const width = Math.max(1, timelineShell.clientWidth - 160);
  const deltaMs = -(event.clientX - timelinePanAnchor) / width * (timelinePanViewportEnd - timelinePanViewportStart);
  viewportStart = timelinePanViewportStart + deltaMs;
  viewportEnd = timelinePanViewportEnd + deltaMs;
  clampViewport();
  render();
}

function finishTimelinePan() {
  timelinePanAnchor = null;
}
```

Add overview coordinate helper and update existing overview handlers to call `setViewport` and `panViewport`:

```js
function overviewMsFromClientX(clientX) {
  const overviewCanvas = document.getElementById('overview-canvas');
  const rect = overviewCanvas.getBoundingClientRect();
  const ratio = Math.max(0, Math.min(1, (clientX - rect.left) / rect.width));
  return traceStart + ratio * Math.max(1, traceEnd - traceStart);
}
```

Register:

```js
timelineShell.addEventListener('pointerdown', startTimelinePan);
window.addEventListener('pointermove', updateTimelinePan);
window.addEventListener('pointerup', finishTimelinePan);
```

- [ ] **Step 7: Run targeted test**

Run:

```bash
go test ./internal/server -run TestIndexUsesPerfettoViewportCanvasOverviewAndBoundedLabels -count=1
```

Expected: PASS.

---

### Task 3: Verify full suite and current session

**Files:**
- Modify: none unless verification exposes failures
- Test: all Go packages and current session view

- [ ] **Step 1: Run full test suite**

Run:

```bash
go test ./...
```

Expected: all packages pass.

- [ ] **Step 2: Run current-session smoke view**

Run:

```bash
go run ./tmp_import_current_session_full.go && go run ./cmd/cctrace view current_claude_session_full
```

Expected: output includes `imported ... events as current_claude_session_full` and `cctrace UI: http://127.0.0.1:<port>` without `bufio.Scanner: token too long` or server startup failure.

- [ ] **Step 3: Manual UI check**

Open the printed URL. Expected behavior:

- initial load shows latest viewport quickly
- overview appears as density bars, not thousands of DOM nodes
- wheel/trackpad pans the timeline
- Ctrl/Meta+wheel zooms around pointer
- dragging blank timeline space pans
- dragging overview window moves the viewport
- selecting an event still shows full details/raw JSON

---

## Self-Review

- Spec coverage: viewport pan/zoom, canvas overview, bounded labels, current-session verification are covered.
- Placeholder scan: no placeholders remain.
- Type consistency: plan consistently uses `viewportStart`, `viewportEnd`, `renderItems`, `buildRenderModel`, `drawOverviewCanvas`, `panViewport`, and `zoomViewportAt`.
