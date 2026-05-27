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
    :root { color-scheme: dark; --bg: #0f172a; --panel: #111827; --muted: #94a3b8; --line: #334155; --text: #e2e8f0; --lane: #0b1220; --px-ms: 0.12; }
    * { box-sizing: border-box; }
    body { margin: 0; font-family: ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif; background: var(--bg); color: var(--text); }
    header { display: flex; justify-content: space-between; align-items: center; gap: 16px; padding: 14px 20px; border-bottom: 1px solid var(--line); background: #020617; }
    header strong { font-size: 18px; }
    header span { color: var(--muted); font-size: 13px; }
    main { display: grid; grid-template-rows: auto auto minmax(0, 1fr) 260px; height: calc(100vh - 54px); }
    .toolbar { display: flex; justify-content: space-between; align-items: center; gap: 16px; padding: 10px 16px; border-bottom: 1px solid var(--line); background: var(--panel); }
    .zoom-controls { display: flex; align-items: center; gap: 8px; color: var(--muted); font-size: 12px; }
    .zoom-button { border: 1px solid var(--line); color: var(--text); background: #0f172a; border-radius: 8px; padding: 6px 10px; cursor: pointer; font-size: 12px; }
    .zoom-button:hover { border-color: #38bdf8; }
    h3 { margin: 0 0 10px; font-size: 12px; color: var(--muted); text-transform: uppercase; letter-spacing: 0.08em; }
    .filters { display: flex; flex-wrap: wrap; gap: 8px; }
    .filter { border: 1px solid var(--line); color: var(--text); background: #0f172a; border-radius: 999px; padding: 6px 10px; cursor: pointer; font-size: 12px; }
    .filter.active { border-color: #38bdf8; background: #075985; }
    .overview-shell { position: relative; height: 72px; padding: 10px 16px; border-bottom: 1px solid var(--line); background: #020617; }
    .overview-canvas { display: block; width: 100%; height: 50px; border: 1px solid var(--line); border-radius: 10px; background: #0b1220; cursor: crosshair; }
    .overview-window { position: absolute; top: 10px; bottom: 10px; border: 2px solid #38bdf8; border-radius: 10px; background: rgba(56,189,248,.12); cursor: grab; }
    .overview-window:active { cursor: grabbing; }
    .overview-frame-handle { position: absolute; top: 0; bottom: 0; width: 10px; background: rgba(226,232,240,.85); }
    .overview-frame-handle.left { left: -5px; cursor: ew-resize; }
    .overview-frame-handle.right { right: -5px; cursor: ew-resize; }
    .timeline-shell { display: grid; grid-template-columns: 160px minmax(0, 1fr); overflow: auto; border-bottom: 1px solid var(--line); background: #020617; }
    .lane-labels { position: sticky; left: 0; z-index: 5; background: #020617; border-right: 1px solid var(--line); }
    .corner, .time-ruler { position: sticky; top: 0; z-index: 4; height: 36px; background: #020617; border-bottom: 1px solid var(--line); }
    .corner { display: flex; align-items: center; padding: 0 12px; color: var(--muted); font-size: 12px; z-index: 6; }
    .lane-label { height: 52px; display: flex; align-items: center; padding: 0 12px; border-bottom: 1px solid #1e293b; color: #cbd5e1; font-size: 13px; }
    .timeline-canvas { min-width: 1400px; position: relative; }
    .time-ruler { position: sticky; top: 0; overflow: hidden; }
    .tick { position: absolute; top: 0; bottom: 0; border-left: 1px solid #1e293b; color: var(--muted); font-size: 11px; padding-left: 4px; }
    .lane-track { position: relative; height: 52px; border-bottom: 1px solid #1e293b; background: linear-gradient(90deg, rgba(51,65,85,.26) 1px, transparent 1px) 0 0 / 120px 100%, var(--lane); }
    .event-block, .event-marker { position: absolute; top: 12px; height: 28px; border: 1px solid rgba(255,255,255,.25); color: #020617; font-size: 12px; font-weight: 700; overflow: hidden; white-space: nowrap; text-overflow: ellipsis; cursor: pointer; box-shadow: 0 6px 18px rgba(0,0,0,.25); }
    .event-block { min-width: 56px; border-radius: 8px; padding: 5px 8px; }
    .event-marker { width: 18px; transform: rotate(45deg); border-radius: 4px; }
    .event-marker span { display: none; }
    .event-block.selected, .event-marker.selected { outline: 2px solid #f8fafc; outline-offset: 2px; }
    .user_message { background: #60a5fa; }
    .agent_turn { background: #c084fc; }
    .tool_call, .tool_result { background: #facc15; }
    .skill { background: #22d3ee; }
    .subagent { background: #f472b6; }
    .permission { background: #fb923c; }
    .hook { background: #94a3b8; }
    .process { background: #4ade80; }
    .api_request, .api_response { background: #34d399; }
    .error { background: #f87171; }
    .details-pane { display: grid; grid-template-columns: minmax(360px, 42vw) 1fr; gap: 16px; padding: 16px; overflow: auto; background: var(--panel); }
    .details-card { border: 1px solid var(--line); border-radius: 14px; background: #0f172a; padding: 16px; }
    .details-grid { display: grid; grid-template-columns: 120px 1fr; gap: 8px 12px; font-size: 14px; }
    .details-grid dt { color: var(--muted); }
    .details-grid dd { margin: 0; word-break: break-word; }
    pre { margin: 0; height: 100%; white-space: pre-wrap; word-break: break-word; background: #020617; border: 1px solid var(--line); border-radius: 12px; padding: 12px; overflow: auto; }
  </style>
</head>
<body>
<header><strong>cctrace</strong><span id="session-mode">Loading session...</span></header>
<main>
  <div class="toolbar">
    <div class="filters">
      <button class="filter active" data-filter="all">All</button>
      <button class="filter" data-filter="user_message">User</button>
      <button class="filter" data-filter="agent_turn">AI</button>
      <button class="filter" data-filter="tool_call">Tool</button>
      <button class="filter" data-filter="skill">Skill</button>
      <button class="filter" data-filter="subagent">Agent</button>
      <button class="filter" data-filter="process">Process</button>
      <button class="filter" data-filter="api_request">API</button>
      <button class="filter" data-filter="error">Error</button>
    </div>
    <div class="zoom-controls">
      <button class="zoom-button" id="zoom-out">−</button>
      <button class="zoom-button" id="zoom-reset">100%</button>
      <button class="zoom-button" id="zoom-in">+</button>
    </div>
  </div>
  <div class="overview-shell">
    <canvas class="overview-canvas" id="overview-canvas"></canvas><div class="overview-window" id="overview-window"><div class="overview-frame-handle left"></div><div class="overview-frame-handle right"></div></div>
  </div>
  <div class="timeline-shell">
    <div class="lane-labels" id="lane-labels"><div class="corner">Time</div></div>
    <div class="timeline-canvas" id="timeline-canvas"><div class="time-ruler" id="time-ruler"></div></div>
  </div>
  <div class="details-pane">
    <div>
      <h3>Selected Event</h3>
      <div id="details" class="details-card">Select an event</div>
    </div>
    <div>
      <h3>Raw JSON</h3>
      <pre id="raw">Select an event</pre>
    </div>
  </div>
</main>
<script>
const lanes = [
  { id: 'process', label: 'Process' },
  { id: 'user', label: 'User' },
  { id: 'assistant', label: 'Assistant' },
  { id: 'skills', label: 'Skills' },
  { id: 'tools', label: 'Tools' },
  { id: 'subagents', label: 'Subagents' },
  { id: 'permissions', label: 'Permissions / Hooks' },
  { id: 'api', label: 'API' },
  { id: 'errors', label: 'Errors' }
];
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
let draftViewportStart = null;
let draftViewportEnd = null;
let overviewDragStart = null;
let overviewDragMode = '';
let overviewMoveAnchor = 0;
let overviewDragFrameStart = 0;
let overviewDragFrameEnd = 0;
let timelinePanAnchor = null;
let timelinePanViewportStart = 0;
let timelinePanViewportEnd = 0;

async function loadSession() {
  const res = await fetch('/api/session');
  sessionMetadata = await res.json();
  const mode = String(sessionMetadata.mode || 'live').toUpperCase();
  const provider = sessionMetadata.provider || 'unknown';
  const id = sessionMetadata.id || '';
  document.getElementById('session-mode').textContent = mode + ' · ' + id + ' · ' + provider;
}

function agentLaneKey(event) {
  if (event.type === 'process' || event.type === 'user_message') return 'main';
  if (event.type === 'skill' && event.summary && event.summary.phase === 'startup') return 'main';
  if (event.type === 'subagent') return 'subagent:' + (event.title || 'unknown');
  if (event.type === 'permission' || event.type === 'hook' || event.type === 'error') return 'system';
  if (event.type === 'api_request' || event.type === 'api_response') return 'api';
  return 'provider:' + providerLabel(event);
}

function agentLaneForEvent(event) {
  const key = agentLaneKey(event);
  if (key === 'main') return { id: 'main', label: 'Main Session' };
  if (key === 'system') return { id: 'system', label: 'System / Hooks' };
  if (key === 'api') return { id: 'api', label: 'API / Model' };
  if (key.startsWith('subagent:')) return { id: key, label: 'Subagent: ' + shortText(key.slice('subagent:'.length), 80) };
  return { id: key, label: providerLabel(event) };
}

function discoverAgentLanes(events) {
  const order = ['main'];
  const byID = { main: { id: 'main', label: 'Main Session' } };
  events.forEach((event) => {
    const lane = agentLaneForEvent(event);
    if (!byID[lane.id]) {
      byID[lane.id] = lane;
      order.push(lane.id);
    }
  });
  for (const lane of [{ id: 'system', label: 'System / Hooks' }, { id: 'api', label: 'API / Model' }]) {
    if (!byID[lane.id]) {
      byID[lane.id] = lane;
      order.push(lane.id);
    }
  }
  return order.map((id) => byID[id]);
}

function providerLabel(event) {
  const summary = event.summary || {};
  const provider = summary.provider || sessionMetadata.provider || 'agent';
  if (provider === 'claude') return 'Claude';
  if (provider === 'codex') return 'Codex';
  if (provider === 'view') return 'Agent';
  return String(provider).charAt(0).toUpperCase() + String(provider).slice(1);
}

function typeBadge(type) {
  const labels = { user_message: 'USER', agent_turn: 'AI', tool_call: 'TOOL', tool_result: 'RESULT', skill: 'SKILL', subagent: 'AGENT', permission: 'PERM', hook: 'HOOK', process: 'PROC', api_request: 'API', api_response: 'API', error: 'ERR' };
  return labels[type] || type.toUpperCase();
}

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

function filterMatches(event) {
  if (currentFilter === 'all') return true;
  if (currentFilter === 'tool_call') return event.type === 'tool_call' || event.type === 'tool_result';
  if (currentFilter === 'api_request') return event.type === 'api_request' || event.type === 'api_response';
  return event.type === currentFilter;
}

function initializeViewport(renderItems) {
  traceStart = renderItems.length ? Math.min(...renderItems.map((item) => item.start)) : Date.now();
  traceEnd = renderItems.length ? Math.max(...renderItems.map((item) => item.end)) : traceStart + 10000;
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
  const fullSpan = Math.max(minViewportMs, traceEnd - traceStart);
  const span = Math.min(viewportSpan(), fullSpan);
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
  const fullSpan = Math.max(minViewportMs, traceEnd - traceStart);
  const span = Math.max(minViewportMs, Math.min(fullSpan, nextSpanMs));
  setViewport(anchor - span * ratio, anchor + span * (1 - ratio));
}

function formatClockTime(timestampMs) {
  return new Date(timestampMs).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit' });
}

async function loadEvents() {
  const res = await fetch('/api/events');
  allEvents = await res.json();
  renderItems = buildRenderModel(allEvents);
  render();
}

function appendEvent(event) {
  if (allEvents.some((item) => item.id === event.id)) return;
  allEvents.push(event);
  renderItems.push(buildRenderModel([event])[0]);
  const shouldFollow = viewportEnd >= traceEnd - 1000;
  initializeViewport(renderItems);
  if (shouldFollow) {
    const span = viewportSpan();
    viewportEnd = traceEnd;
    viewportStart = traceEnd - span;
    clampViewport();
  }
  render();
}

function connectEventStream() {
  if (!window.EventSource || sessionMetadata.mode !== 'live') return;
  const source = new EventSource('/api/events/stream');
  source.onmessage = (message) => appendEvent(JSON.parse(message.data));
}

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

function renderLanes(agentLanes) {
  const labels = document.getElementById('lane-labels');
  labels.innerHTML = '<div class="corner">Time</div>';
  agentLanes.forEach((lane) => {
    const label = document.createElement('div');
    label.className = 'lane-label';
    label.textContent = lane.label;
    labels.appendChild(label);
  });
}

function renderRuler(width, spanMs, sessionStart) {
  const canvas = document.getElementById('timeline-canvas');
  canvas.style.width = width + 'px';
  canvas.innerHTML = '<div class="time-ruler" id="time-ruler"></div>';
  const ruler = document.getElementById('time-ruler');
  ruler.style.width = width + 'px';
  const stepMs = spanMs > 10 * 60 * 1000 ? 60 * 1000 : spanMs > 60 * 1000 ? 10 * 1000 : 1000;
  for (let ms = 0; ms <= Math.max(spanMs, stepMs); ms += stepMs) {
    const tick = document.createElement('div');
    tick.className = 'tick';
    tick.style.left = (ms / spanMs * width) + 'px';
    tick.textContent = formatClockTime(sessionStart + ms);
    ruler.appendChild(tick);
  }
}

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
  const start = draftViewportStart ?? viewportStart;
  const end = draftViewportEnd ?? viewportEnd;
  windowNode.style.left = (16 + (start - sessionStart) / span * rect.width) + 'px';
  windowNode.style.width = Math.max(8, (end - start) / span * rect.width) + 'px';
}

function previewViewport(startMs, endMs) {
  draftViewportStart = Math.max(traceStart, Math.min(startMs, endMs));
  draftViewportEnd = Math.min(traceEnd, Math.max(startMs, endMs));
  if (draftViewportEnd - draftViewportStart < minViewportMs) draftViewportEnd = Math.min(traceEnd, draftViewportStart + minViewportMs);
  renderOverviewWindow(traceStart, traceEnd);
}

function commitPreviewViewport() {
  if (draftViewportStart == null || draftViewportEnd == null) return;
  viewportStart = draftViewportStart;
  viewportEnd = draftViewportEnd;
  draftViewportStart = null;
  draftViewportEnd = null;
  clampViewport();
  render();
}

function overviewDragHitMode(event) {
  if (event.target.classList.contains('left')) return 'resize-left';
  if (event.target.classList.contains('right')) return 'resize-right';
  if (event.target.closest('#overview-window')) return 'move';
  return 'select';
}

function overviewMsFromClientX(clientX) {
  const overviewCanvas = document.getElementById('overview-canvas');
  const rect = overviewCanvas.getBoundingClientRect();
  const ratio = Math.max(0, Math.min(1, (clientX - rect.left) / rect.width));
  return traceStart + ratio * Math.max(1, traceEnd - traceStart);
}

function startOverviewRangeSelection(event) {
  overviewDragMode = overviewDragHitMode(event);
  overviewDragStart = overviewMsFromClientX(event.clientX);
  overviewMoveAnchor = overviewDragStart;
  overviewDragFrameStart = viewportStart;
  overviewDragFrameEnd = viewportEnd;
  if (overviewDragMode === 'select') {
    const nextEnd = overviewDragStart + Math.max(minViewportMs, (traceEnd - traceStart) * 0.05);
    previewViewport(overviewDragStart, nextEnd);
  }
}

function updateOverviewRangeSelection(event) {
  if (overviewDragStart == null) return;
  const current = overviewMsFromClientX(event.clientX);
  let nextStart = overviewDragFrameStart;
  let nextEnd = overviewDragFrameEnd;
  if (overviewDragMode === 'resize-left') {
    nextStart = Math.min(current, overviewDragFrameEnd - minViewportMs);
  } else if (overviewDragMode === 'resize-right') {
    nextEnd = Math.max(current, overviewDragFrameStart + minViewportMs);
  } else if (overviewDragMode === 'move') {
    const width = overviewDragFrameEnd - overviewDragFrameStart;
    nextStart = Math.max(traceStart, Math.min(traceEnd - width, overviewDragFrameStart + current - overviewMoveAnchor));
    nextEnd = nextStart + width;
  } else {
    nextStart = overviewDragStart;
    nextEnd = current;
  }
  previewViewport(nextStart, nextEnd);
}

function finishOverviewRangeSelection() {
  if (overviewDragStart != null) commitPreviewViewport();
  overviewDragStart = null;
  overviewDragMode = '';
}

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

function renderTrack(lane, width) {
  const canvas = document.getElementById('timeline-canvas');
  const track = document.createElement('div');
  track.className = 'lane-track';
  track.dataset.lane = lane.id;
  track.style.width = width + 'px';
  canvas.appendChild(track);
}

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

function show(event) {
  selectedId = event.id;
  const summary = event.summary || {};
  const rows = [
    ['Type', event.type],
    ['Lane', agentLaneForEvent(event).label],
    ['Title', event.title],
    ['Description', summary.description || summary.text || ''],
    ['Status', event.status],
    ['Source', event.source],
    ['Confidence', event.confidence],
    ['Timestamp', new Date(event.timestamp).toLocaleString()]
  ];
  document.getElementById('details').innerHTML = '<dl class="details-grid">' + rows.map(([k, v]) => '<dt>' + escapeHtml(k) + '</dt><dd>' + escapeHtml(String(v || '-')) + '</dd>').join('') + '</dl>';
  document.getElementById('raw').textContent = JSON.stringify(event, null, 2);
  render();
}

function escapeHtml(value) {
  return value.replace(/[&<>"']/g, (ch) => ({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#039;'}[ch]));
}

document.querySelectorAll('.filter').forEach((button) => {
  button.onclick = () => {
    document.querySelectorAll('.filter').forEach((item) => item.classList.remove('active'));
    button.classList.add('active');
    currentFilter = button.dataset.filter;
    render();
  };
});

const timelineShell = document.querySelector('.timeline-shell');
const overviewCanvas = document.getElementById('overview-canvas');
document.getElementById('zoom-out').onclick = () => zoomViewportAt(window.innerWidth / 2, viewportSpan() * 1.5);
document.getElementById('zoom-reset').onclick = () => setViewport(Math.max(traceStart, traceEnd - maxInitialViewportMs), traceEnd);
document.getElementById('zoom-in').onclick = () => zoomViewportAt(window.innerWidth / 2, viewportSpan() / 1.5);
timelineShell.addEventListener('wheel', (event) => {
  event.preventDefault();
  if (event.ctrlKey || event.metaKey) {
    zoomViewportAt(event.clientX, viewportSpan() * (event.deltaY < 0 ? 0.8 : 1.25));
    return;
  }
  const delta = Math.abs(event.deltaX) > Math.abs(event.deltaY) ? event.deltaX : event.deltaY;
  panViewport(delta / Math.max(1, timelineShell.clientWidth - 160) * viewportSpan());
}, { passive: false });
timelineShell.addEventListener('pointerdown', startTimelinePan);
window.addEventListener('pointermove', updateTimelinePan);
window.addEventListener('pointerup', finishTimelinePan);
overviewCanvas.addEventListener('pointerdown', startOverviewRangeSelection);
overviewCanvas.addEventListener('pointermove', updateOverviewRangeSelection);
window.addEventListener('pointerup', finishOverviewRangeSelection);
window.addEventListener('gesturestart', (event) => event.preventDefault());
window.addEventListener('gesturechange', (event) => {
  event.preventDefault();
  zoomViewportAt(window.innerWidth / 2, viewportSpan() / Math.max(0.1, event.scale));
});
loadSession().then(() => loadEvents().then(connectEventStream));
</script>
</body>
</html>`

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
	return mux
}

func Listen(addr string, hub *Hub) error {
	fmt.Printf("cctrace UI: http://%s\n", addr)
	return http.ListenAndServe(addr, NewHandler(hub))
}
