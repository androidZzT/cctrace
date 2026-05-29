package server

const indexHTML = `<!doctype html>
<html>
<head>
  <meta charset="utf-8">
  <title>cctrace</title>
  <style>
    :root { color-scheme: dark; --bg: #0f172a; --panel: #111827; --muted: #94a3b8; --line: #334155; --text: #e2e8f0; --soft: #1e293b; --accent: #38bdf8; --danger: #f87171; }
    * { box-sizing: border-box; }
    body { margin: 0; font-family: ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif; background: var(--bg); color: var(--text); }
    header { display: flex; justify-content: space-between; align-items: center; gap: 16px; height: 54px; padding: 0 20px; border-bottom: 1px solid var(--line); background: #020617; }
    header strong { font-size: 18px; }
    header span { color: var(--muted); font-size: 13px; }
    main { display: grid; grid-template-rows: auto auto minmax(0, 1fr); height: calc(100vh - 54px); min-height: 0; }
    h3 { margin: 0 0 10px; font-size: 12px; color: var(--muted); text-transform: uppercase; letter-spacing: .08em; }
    button, input { font: inherit; }
    .toolbar { display: flex; align-items: center; gap: 16px; padding: 10px 16px; border-bottom: 1px solid var(--line); background: var(--panel); flex-wrap: wrap; }
    .filters, .import-controls { display: flex; align-items: center; gap: 8px; }
    .filters { flex-wrap: wrap; }
    .import-controls { flex: 1 1 420px; justify-content: center; min-width: min(460px, 100%); }
    .filter, .zoom-button, .nav-button, .details-action { border: 1px solid var(--line); color: var(--text); background: #0f172a; border-radius: 8px; padding: 6px 10px; cursor: pointer; font-size: 12px; }
    .filter { border-radius: 999px; }
    .filter.active, .nav-button.active, .details-action.active { border-color: var(--accent); background: #075985; }
    .filter:hover, .zoom-button:hover, .nav-button:hover, .details-action:hover { border-color: var(--accent); }
    .import-input { width: min(360px, 100%); min-width: 220px; border: 1px solid var(--line); color: var(--text); background: #020617; border-radius: 8px; padding: 7px 10px; font-size: 12px; outline: none; }
    .import-input:focus { border-color: var(--accent); }
    .file-input { display: none; }
    .import-status { min-width: 110px; color: var(--muted); font-size: 12px; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
    .drop-overlay { position: fixed; inset: 0; z-index: 100; display: grid; place-items: center; padding: 24px; background: rgba(2,6,23,.78); backdrop-filter: blur(8px); opacity: 0; visibility: hidden; pointer-events: none; transition: opacity .14s ease, visibility .14s ease; }
    .drop-overlay.dragging { opacity: 1; visibility: visible; }
    .drop-overlay-panel { width: min(560px, 100%); border: 2px dashed var(--accent); border-radius: 8px; background: rgba(15,23,42,.92); box-shadow: 0 28px 80px rgba(0,0,0,.42); padding: 34px 28px; text-align: center; }
    .drop-overlay-title { font-size: 22px; font-weight: 800; }
    .drop-overlay-subtitle { color: var(--muted); font-size: 13px; margin-top: 8px; }
    .overview-shell { position: relative; height: 72px; padding: 10px 16px; border-bottom: 1px solid var(--line); background: #020617; touch-action: none; }
    .overview-canvas { display: block; width: 100%; height: 50px; border: 1px solid var(--line); border-radius: 8px; background: #0b1220; cursor: crosshair; touch-action: none; }
    .overview-window { position: absolute; top: 10px; bottom: 10px; min-width: 28px; border: 2px solid var(--accent); border-radius: 8px; background: rgba(56,189,248,.12); cursor: grab; z-index: 3; touch-action: none; }
    .overview-window:active { cursor: grabbing; }
    .overview-frame-handle { position: absolute; top: 0; bottom: 0; width: 10px; background: rgba(226,232,240,.85); }
    .overview-frame-handle.left { left: -5px; cursor: ew-resize; }
    .overview-frame-handle.right { right: -5px; cursor: ew-resize; }
    .explorer-shell { display: grid; grid-template-columns: 240px minmax(0, 1fr) minmax(340px, 36vw); min-height: 0; background: #020617; }
    .sidebar, .event-stream, .details-pane { min-height: 0; overflow: auto; }
    .sidebar { border-right: 1px solid var(--line); padding: 14px 12px; background: #07111f; }
    .event-stream { padding: 14px 18px 28px; }
    .details-pane { border-left: 1px solid var(--line); padding: 14px; background: var(--panel); }
    .nav-section { margin-bottom: 18px; }
    .nav-button { width: 100%; display: grid; grid-template-columns: minmax(0, 1fr) auto; gap: 8px; text-align: left; margin: 6px 0; align-items: center; }
    .nav-label { overflow: hidden; white-space: nowrap; text-overflow: ellipsis; }
    .nav-count { color: var(--muted); font-size: 11px; }
    .stream-header { display: flex; align-items: end; justify-content: space-between; gap: 12px; margin-bottom: 12px; }
    .stream-title { font-weight: 750; font-size: 18px; }
    .stream-subtitle { color: var(--muted); font-size: 12px; }
    .turn-group { border: 1px solid var(--line); background: #0b1220; border-radius: 8px; margin-bottom: 12px; overflow: hidden; }
    .turn-header { display: grid; grid-template-columns: auto minmax(0, 1fr) auto; gap: 12px; align-items: center; padding: 10px 12px; background: #111827; border-bottom: 1px solid #1e293b; }
    .turn-index { color: var(--accent); font-size: 12px; font-weight: 800; text-transform: uppercase; letter-spacing: .06em; }
    .turn-title { overflow: hidden; white-space: nowrap; text-overflow: ellipsis; font-size: 14px; font-weight: 700; }
    .turn-meta { color: var(--muted); font-size: 12px; white-space: nowrap; }
    .event-row, .operation-row { display: grid; grid-template-columns: 76px 92px 120px minmax(0, 1fr) auto; gap: 10px; align-items: center; padding: 9px 12px; border-bottom: 1px solid #1e293b; cursor: pointer; }
    .event-row:last-child, .operation-row:last-child { border-bottom: 0; }
    .event-row:hover, .operation-row:hover, .event-row.selected, .operation-row.selected { background: rgba(56,189,248,.08); }
    .event-row.related, .operation-row.related { box-shadow: inset 3px 0 0 var(--accent); }
    .time-cell { color: var(--muted); font-size: 11px; font-variant-numeric: tabular-nums; }
    .badge { border-radius: 999px; padding: 3px 7px; color: #020617; font-size: 11px; font-weight: 800; text-align: center; }
    .thread-pill { border: 1px solid var(--line); border-radius: 999px; padding: 3px 7px; color: #cbd5e1; font-size: 11px; overflow: hidden; white-space: nowrap; text-overflow: ellipsis; }
    .row-main { min-width: 0; }
    .row-title { overflow: hidden; white-space: nowrap; text-overflow: ellipsis; font-size: 13px; font-weight: 650; }
    .row-summary { color: var(--muted); font-size: 12px; margin-top: 2px; overflow: hidden; white-space: nowrap; text-overflow: ellipsis; }
    .duration-chip { color: var(--muted); font-size: 11px; font-variant-numeric: tabular-nums; }
    .operation-row { background: rgba(250,204,21,.04); }
    .operation-row .row-title:before { content: "Operation · "; color: #facc15; }
    .duration-bar { height: 4px; background: #334155; border-radius: 999px; overflow: hidden; margin-top: 5px; }
    .duration-fill { height: 100%; width: 0%; background: #facc15; border-radius: inherit; }
    .details-card { border: 1px solid var(--line); border-radius: 8px; background: #0f172a; padding: 14px; margin-bottom: 12px; }
    .details-grid { display: grid; grid-template-columns: 112px minmax(0, 1fr); gap: 8px 12px; font-size: 13px; }
    .details-grid dt { color: var(--muted); }
    .details-grid dd { margin: 0; word-break: break-word; }
    .details-actions { display: flex; gap: 8px; flex-wrap: wrap; margin-bottom: 12px; }
    pre { margin: 0; min-height: 180px; max-height: 42vh; white-space: pre-wrap; word-break: break-word; background: #020617; border: 1px solid var(--line); border-radius: 8px; padding: 12px; overflow: auto; font-size: 12px; }
    .empty-state { color: var(--muted); border: 1px dashed var(--line); border-radius: 8px; padding: 18px; text-align: center; }
    .user_message { background: #60a5fa; }
    .agent_turn { background: #c084fc; }
    .tool_call, .tool_result, .operation { background: #facc15; }
    .skill { background: #22d3ee; }
    .skill-available { background: #67e8f9; }
    .skill-invoked { background: #2dd4bf; }
    .subagent { background: #f472b6; }
    .permission { background: #fb923c; }
    .hook { background: #94a3b8; }
    .process { background: #4ade80; }
    .api_request, .api_response { background: #34d399; }
    .error { background: var(--danger); }
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
    <form class="import-controls" id="import-form">
      <input class="import-input" id="import-path" type="text" placeholder="Session path, rollout JSONL, or folder" autocomplete="off">
      <button class="zoom-button" id="import-submit" type="submit">Import</button>
      <input class="file-input" id="import-file" type="file" multiple accept=".jsonl,.json,.txt,application/json">
      <input class="file-input" id="import-folder" type="file" webkitdirectory multiple>
      <button class="zoom-button" id="file-choose" type="button">File</button>
      <button class="zoom-button" id="folder-choose" type="button">Folder</button>
      <span class="import-status" id="import-status"></span>
    </form>
  </div>
  <div class="overview-shell">
    <canvas class="overview-canvas" id="overview-canvas"></canvas><div class="overview-window" id="overview-window"><div class="overview-frame-handle left"></div><div class="overview-frame-handle right"></div></div>
  </div>
  <div class="explorer-shell">
    <aside class="sidebar">
      <div class="nav-section">
        <h3>Thread</h3>
        <div id="thread-nav"></div>
      </div>
      <div class="nav-section">
        <h3>请求轮次</h3>
        <div id="turn-nav"></div>
      </div>
    </aside>
    <section class="event-stream" id="event-stream">
      <div class="stream-header">
        <div>
          <div class="stream-title">Event Stream</div>
          <div class="stream-subtitle" id="stream-summary">Loading events...</div>
        </div>
      </div>
      <div id="turn-groups"></div>
    </section>
    <section class="details-pane">
      <h3>Details</h3>
      <div class="details-actions">
        <button class="details-action" id="related-toggle" type="button">Related events</button>
      </div>
      <div id="details" class="details-card">Select an event or operation</div>
      <h3>Raw JSON</h3>
      <pre id="raw">Select an event or operation</pre>
    </section>
  </div>
</main>
<div class="drop-overlay" id="import-dropzone" aria-hidden="true">
  <div class="drop-overlay-panel">
    <div class="drop-overlay-title">松开导入 Trace 包</div>
    <div class="drop-overlay-subtitle">支持 JSONL 文件或文件夹</div>
  </div>
</div>
<script>
const minViewportMs = 1000;
const maxInitialViewportMs = 5 * 60 * 1000;
const minOverviewWindowPx = 28;
let currentFilter = 'all';
let selectedThreadId = 'all';
let selectedTurnId = '';
let selectedId = '';
let selectedCorrelationIds = [];
let relatedOnly = false;
let allEvents = [];
let renderItems = [];
let operations = [];
let turnGroups = [];
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
let eventStream = null;

async function loadSession() {
  const res = await fetch('/api/session');
  sessionMetadata = await res.json();
  const mode = String(sessionMetadata.mode || 'live').toUpperCase();
  document.getElementById('session-mode').textContent = mode + ' · ' + (sessionMetadata.id || '') + ' · ' + (sessionMetadata.provider || 'unknown');
}

async function loadEvents() {
  const res = await fetch('/api/events');
  allEvents = await res.json();
  rebuildViewModel();
  render();
}

function rebuildViewModel() {
  renderItems = buildRenderModel(allEvents);
  operations = buildOperations(renderItems);
  turnGroups = buildTurnGroups(renderItems, operations);
}

function providerLabel(event) {
  const summary = event.summary || {};
  const provider = summary.provider || sessionMetadata.provider || 'agent';
  if (provider === 'claude') return 'Claude';
  if (provider === 'codex') return 'Codex';
  if (provider === 'view') return 'Agent';
  return String(provider).charAt(0).toUpperCase() + String(provider).slice(1);
}

function mainThreadLabel(events) {
  const providerEvent = events.find((event) => event.summary && event.summary.provider && event.summary.provider !== 'view') || {};
  return 'Agent: ' + providerLabel(providerEvent);
}

function subagentThreadInfo(event) {
  const summary = event.summary || {};
  const correlationIds = event.correlationIds || [];
  const toolUseCorrelation = correlationIds.find((id) => String(id).startsWith('toolu_'));
  const correlationId = toolUseCorrelation || correlationIds.find(Boolean);
  const identity = (summary.provider === 'claude' && toolUseCorrelation) ? toolUseCorrelation : (summary.agentId || summary.agentID || correlationId || summary.nickname || summary.agentType || summary.subagentType || event.title || 'unknown');
  const type = summary.agentType || summary.subagentType || '';
  let label = summary.nickname || '';
  if (!label && type && summary.description) label = type + ': ' + summary.description;
  if (!label) label = type || summary.description || event.title || identity;
  return { id: 'agent:' + identity, label: 'Agent: ' + shortText(label, 80) };
}

function buildAgentThreadIndex(events) {
  const main = { id: 'agent:main', label: mainThreadLabel(events) };
  const index = { byID: { [main.id]: main }, byEventID: {}, byCorrelationID: {} };
  events.forEach((event) => {
    if (event.type !== 'subagent') return;
    const thread = subagentThreadInfo(event);
    if (!index.byID[thread.id]) index.byID[thread.id] = thread;
    index.byEventID[event.id] = thread.id;
    (event.correlationIds || []).forEach((id) => { if (id) index.byCorrelationID[id] = thread.id; });
  });
  return index;
}

function agentThreadForEvent(event, threadIndex) {
  if (threadIndex.byEventID[event.id]) return threadIndex.byID[threadIndex.byEventID[event.id]];
  if (event.parentId && threadIndex.byEventID[event.parentId]) return threadIndex.byID[threadIndex.byEventID[event.parentId]];
  for (const id of event.correlationIds || []) {
    if (threadIndex.byCorrelationID[id]) return threadIndex.byID[threadIndex.byCorrelationID[id]];
  }
  return threadIndex.byID['agent:main'];
}

function buildRenderModel(events) {
  const threadIndex = buildAgentThreadIndex(events);
  return events.map((event, index) => {
    const start = new Date(event.timestamp).getTime();
    const duration = Number(event.durationMs || 0);
    const thread = agentThreadForEvent(event, threadIndex);
    return { kind: 'event', id: 'event:' + event.id, event, index, start, end: start + duration, durationMs: duration, threadId: thread.id, threadLabel: thread.label, title: event.title || event.type };
  });
}

function correlationKey(event) {
  return (event.correlationIds || []).find(Boolean) || '';
}

function buildOperations(items) {
  const calls = new Map();
  const results = new Map();
  items.forEach((item) => {
    const key = correlationKey(item.event);
    if (!key) return;
    if (item.event.type === 'tool_call') calls.set(key, item);
    if (item.event.type === 'tool_result') results.set(key, item);
  });
  const ops = [];
  calls.forEach((callItem, key) => {
    const resultItem = results.get(key);
    const end = resultItem ? resultItem.start : callItem.end;
    const duration = Math.max(0, end - callItem.start);
    const tool = (callItem.event.summary || {}).tool || callItem.event.title || 'tool';
    const status = resultItem ? resultItem.event.status : callItem.event.status;
    ops.push({
      kind: 'operation',
      id: 'operation:' + key,
      key,
      call: callItem.event,
      result: resultItem ? resultItem.event : null,
      events: resultItem ? [callItem.event, resultItem.event] : [callItem.event],
      start: callItem.start,
      end,
      durationMs: duration,
      status,
      tool,
      title: tool,
      threadId: callItem.threadId,
      threadLabel: callItem.threadLabel,
      correlationIds: compactStrings([key].concat(callItem.event.correlationIds || [], resultItem ? resultItem.event.correlationIds || [] : []))
    });
  });
  return ops;
}

function buildTurnGroups(items, ops) {
  const opByCallID = new Map(ops.map((op) => [op.call.id, op]));
  const pairedResultIDs = new Set(ops.filter((op) => op.result).map((op) => op.result.id));
  const groups = [];
  let current = { id: 'turn:setup', label: '会话准备', title: 'Session Setup', start: items[0] ? items[0].start : Date.now(), streamItems: [] };
  groups.push(current);
  let turnNumber = 0;
  items.forEach((item) => {
    const event = item.event;
    if (event.type === 'user_message') {
      turnNumber++;
      current = { id: 'turn:' + turnNumber, label: '请求 ' + turnNumber, title: item.title, start: item.start, streamItems: [] };
      groups.push(current);
    }
    if (event.type === 'tool_call' && opByCallID.has(event.id)) {
      current.streamItems.push(opByCallID.get(event.id));
      return;
    }
    if (pairedResultIDs.has(event.id)) return;
    current.streamItems.push(item);
  });
  return groups.map((group) => {
    const end = group.streamItems.length ? Math.max.apply(null, group.streamItems.map((item) => item.end || item.start)) : group.start;
    return Object.assign(group, { end, count: group.streamItems.length });
  }).filter((group) => group.count > 0 || group.id === 'turn:setup');
}

function buildThreadSummary(items) {
  const byID = {};
  items.forEach((item) => {
    const entry = byID[item.threadId] || { id: item.threadId, label: item.threadLabel, count: 0, tools: 0, errors: 0 };
    entry.count++;
    if (item.event.type === 'tool_call' || item.event.type === 'tool_result') entry.tools++;
    if (item.event.type === 'error' || item.event.status === 'error') entry.errors++;
    byID[item.threadId] = entry;
  });
  return [{ id: 'all', label: 'All Threads', count: items.length, tools: operations.length, errors: items.filter((item) => item.event.type === 'error' || item.event.status === 'error').length }].concat(Object.values(byID));
}

function buildEventDensityBuckets(items, sessionStart, sessionEnd, width) {
  const span = Math.max(1, sessionEnd - sessionStart);
  const buckets = new Array(width).fill(null).map(() => ({ count: 0, tool: 0, skill: 0, subagent: 0, error: 0 }));
  items.forEach((item) => {
    const x = Math.max(0, Math.min(width - 1, Math.floor((item.start - sessionStart) / span * width)));
    buckets[x].count++;
    if (item.event.type === 'tool_call' || item.event.type === 'tool_result') buckets[x].tool++;
    if (item.event.type === 'skill') buckets[x].skill++;
    if (item.event.type === 'subagent') buckets[x].subagent++;
    if (item.event.type === 'error' || item.event.status === 'error') buckets[x].error++;
  });
  return buckets;
}

function filterMatchesEvent(event) {
  if (currentFilter === 'all') return true;
  if (currentFilter === 'tool_call') return event.type === 'tool_call' || event.type === 'tool_result';
  if (currentFilter === 'api_request') return event.type === 'api_request' || event.type === 'api_response';
  return event.type === currentFilter;
}

function streamItemMatches(item) {
  const events = item.kind === 'operation' ? item.events : [item.event];
  if (selectedThreadId !== 'all' && item.threadId !== selectedThreadId) return false;
  if (relatedOnly && selectedCorrelationIds.length > 0 && !hasRelatedCorrelation(item)) return false;
  return events.some((event) => filterMatchesEvent(event));
}

function isInViewport(item) {
  return (item.end || item.start) >= viewportStart && item.start <= viewportEnd;
}

function hasRelatedCorrelation(item) {
  const ids = item.kind === 'operation' ? item.correlationIds : item.event.correlationIds || [];
  return ids.some((id) => selectedCorrelationIds.includes(id));
}

function initializeViewport(items) {
  traceStart = items.length ? Math.min.apply(null, items.map((item) => item.start)) : Date.now();
  traceEnd = items.length ? Math.max.apply(null, items.map((item) => item.end || item.start)) : traceStart + 10000;
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

function setViewport(startMs, endMs) {
  viewportStart = Math.min(startMs, endMs);
  viewportEnd = Math.max(startMs, endMs);
  if (viewportEnd - viewportStart < minViewportMs) viewportEnd = viewportStart + minViewportMs;
  clampViewport();
  render();
}

function panViewport(deltaMs) {
  viewportStart += deltaMs;
  viewportEnd += deltaMs;
  clampViewport();
  render();
}

function overviewMsFromClientX(clientX) {
  const rect = overviewCanvas.getBoundingClientRect();
  const ratio = Math.max(0, Math.min(1, (clientX - rect.left) / Math.max(1, rect.width)));
  return traceStart + ratio * Math.max(1, traceEnd - traceStart);
}

function zoomViewportAt(clientX, nextSpanMs) {
  const anchor = overviewMsFromClientX(clientX);
  const ratio = (anchor - viewportStart) / viewportSpan();
  const fullSpan = Math.max(minViewportMs, traceEnd - traceStart);
  const span = Math.max(minViewportMs, Math.min(fullSpan, nextSpanMs));
  setViewport(anchor - span * ratio, anchor + span * (1 - ratio));
}

function render() {
  initializeViewport(renderItems);
  drawOverviewCanvas(renderItems, traceStart, traceEnd);
  renderOverviewWindow(traceStart, traceEnd);
  renderSidebar();
  renderEventExplorer(turnGroups);
  updateRelatedButton();
}

function renderSidebar() {
  const threadNav = document.getElementById('thread-nav');
  threadNav.innerHTML = '';
  buildThreadSummary(renderItems).forEach((thread) => {
    const button = document.createElement('button');
    button.className = 'nav-button' + (selectedThreadId === thread.id ? ' active' : '');
    button.innerHTML = '<span class="nav-label">' + escapeHtml(thread.label) + '</span><span class="nav-count">' + thread.count + '</span>';
    button.onclick = () => focusThread(thread);
    threadNav.appendChild(button);
  });
  const turnNav = document.getElementById('turn-nav');
  turnNav.innerHTML = '';
  turnGroups.forEach((group) => {
    const button = document.createElement('button');
    button.className = 'nav-button' + (selectedTurnId === group.id ? ' active' : '');
    button.dataset.turnId = group.id;
    button.innerHTML = '<span class="nav-label">' + escapeHtml(group.label) + '</span><span class="nav-count">' + group.count + '</span>';
    button.onclick = () => focusTurnGroup(group);
    turnNav.appendChild(button);
  });
}

function focusThread(thread) {
  selectedThreadId = thread.id;
  selectedTurnId = '';
  if (thread.id === 'all') {
    render();
    return;
  }
  const item = findThreadFocusItem(thread.id);
  if (!item) {
    render();
    return;
  }
  ensureStreamItemMatchesFilter(item);
  focusStreamItem(item);
}

function findThreadFocusItem(threadId) {
  const items = allStreamItems().filter((item) => item.threadId === threadId);
  return items.find((item) => item.kind === 'event' && item.event.type === 'subagent') || items[0] || null;
}

function allStreamItems() {
  return turnGroups.reduce((items, group) => items.concat(group.streamItems), []);
}

function ensureStreamItemMatchesFilter(item) {
  if (streamItemMatches(item)) return;
  setEventTypeFilter('all');
}

function focusStreamItem(item) {
  relatedOnly = false;
  updateSelectedDetails(item);
  const fullSpan = Math.max(minViewportMs, traceEnd - traceStart);
  const currentSpan = Math.min(Math.max(viewportSpan(), minViewportMs), fullSpan);
  const itemStart = item.start;
  const itemEnd = Math.max(item.end || item.start, item.start + minViewportMs);
  const itemSpan = Math.max(minViewportMs, itemEnd - itemStart);
  const span = Math.min(fullSpan, Math.max(currentSpan, itemSpan + Math.min(itemSpan * .2, 30000)));
  const leadPadding = Math.min(span * .18, 20000);
  let start = itemStart - leadPadding;
  let end = start + span;
  if (itemEnd > end) {
    end = itemEnd + Math.min(span * .05, 10000);
    start = end - span;
  }
  setViewport(start, end);
  requestAnimationFrame(() => {
    const row = Array.from(document.querySelectorAll('[data-item-id]')).find((node) => node.dataset.itemId === item.id);
    if (row) row.scrollIntoView({ block: 'center', behavior: 'smooth' });
  });
}

function focusTurnGroup(group) {
  selectedTurnId = group.id;
  const fullSpan = Math.max(minViewportMs, traceEnd - traceStart);
  const currentSpan = Math.min(Math.max(viewportSpan(), minViewportMs), fullSpan);
  const groupStart = group.start;
  const groupEnd = Math.max(group.end || group.start, group.start + minViewportMs);
  const groupSpan = Math.max(minViewportMs, groupEnd - groupStart);
  const span = Math.min(fullSpan, Math.max(currentSpan, groupSpan + Math.min(groupSpan * 0.12, 30000)));
  const leadPadding = Math.min(span * 0.12, 15000);
  let start = groupStart - leadPadding;
  let end = start + span;
  if (groupEnd > end) {
    end = groupEnd + Math.min(span * 0.04, 10000);
    start = end - span;
  }
  setViewport(start, end);
  requestAnimationFrame(() => {
    const node = document.getElementById(group.id);
    if (node) node.scrollIntoView({ block: 'start', behavior: 'smooth' });
  });
}

function renderEventExplorer(groups) {
  const container = document.getElementById('turn-groups');
  container.innerHTML = '';
  let shown = 0;
  groups.forEach((group) => {
    const items = group.streamItems.filter((item) => streamItemMatches(item) && isInViewport(item));
    if (!items.length) return;
    shown += items.length;
    const section = document.createElement('section');
    section.className = 'turn-group';
    section.id = group.id;
    section.dataset.turnId = group.id;
    const header = document.createElement('div');
    header.className = 'turn-header';
    header.innerHTML = '<div class="turn-index">' + escapeHtml(group.label) + '</div><div class="turn-title">' + escapeHtml(shortText(group.title, 150)) + '</div><div class="turn-meta">' + formatClockTime(group.start) + ' · ' + items.length + ' events</div>';
    section.appendChild(header);
    items.forEach((item) => {
      section.appendChild(item.kind === 'operation' ? renderOperationRow(item) : renderEventRow(item));
    });
    container.appendChild(section);
  });
  document.getElementById('stream-summary').textContent = shown + ' visible items · ' + renderItems.length + ' raw events · ' + operations.length + ' operations';
  if (!shown) {
    container.innerHTML = '<div class="empty-state">No events in the current filters or time range</div>';
  }
}

function renderOperationRow(operation) {
  const row = document.createElement('div');
  row.className = rowClass(operation);
  row.dataset.itemId = operation.id;
  row.onclick = () => showOperation(operation);
  const pct = Math.min(100, Math.max(4, operation.durationMs / Math.max(1, viewportSpan()) * 100));
  row.innerHTML =
    '<div class="time-cell">' + formatClockTime(operation.start) + '</div>' +
    '<div class="badge operation">TOOL</div>' +
    '<div class="thread-pill">' + escapeHtml(operation.threadLabel) + '</div>' +
    '<div class="row-main"><div class="row-title">' + escapeHtml(operation.title) + '</div><div class="row-summary">' + escapeHtml(operationSummary(operation)) + '</div><div class="duration-bar"><div class="duration-fill" style="width:' + pct + '%"></div></div></div>' +
    '<div class="duration-chip">' + formatDuration(operation.durationMs) + '</div>';
  return row;
}

function renderEventRow(item) {
  const event = item.event;
  const row = document.createElement('div');
  row.className = rowClass(item);
  row.dataset.itemId = item.id;
  row.onclick = () => showEvent(item);
  row.innerHTML =
    '<div class="time-cell">' + formatClockTime(item.start) + '</div>' +
    '<div class="' + eventBadgeClass(event) + '">' + eventBadgeLabel(event) + '</div>' +
    '<div class="thread-pill">' + escapeHtml(item.threadLabel) + '</div>' +
    '<div class="row-main"><div class="row-title">' + escapeHtml(item.title) + '</div><div class="row-summary">' + escapeHtml(eventSummary(event)) + '</div></div>' +
    '<div class="duration-chip">' + escapeHtml(event.status || '-') + '</div>';
  return row;
}

function rowClass(item) {
  const selected = selectedId === item.id ? ' selected' : '';
  const related = selectedCorrelationIds.length > 0 && hasRelatedCorrelation(item) ? ' related' : '';
  return (item.kind === 'operation' ? 'operation-row' : 'event-row') + selected + related;
}

function showEvent(item) {
  updateSelectedEventDetails(item);
  render();
}

function updateSelectedDetails(item) {
  if (item.kind === 'operation') updateSelectedOperationDetails(item);
  else updateSelectedEventDetails(item);
}

function updateSelectedEventDetails(item) {
  selectedId = item.id;
  selectedCorrelationIds = compactStrings(item.event.correlationIds || []);
  const event = item.event;
  const summary = event.summary || {};
  const rows = [
    ['Kind', 'Event'],
    ['Type', event.type],
    ['Thread', item.threadLabel],
    ['Title', event.title],
    ['Phase', summary.phase || summary.payloadType || ''],
    ['Description', summary.description || ''],
    ['Text', summary.text || ''],
    ['Path', summary.path || ''],
    ['Status', event.status],
    ['Time', new Date(event.timestamp).toLocaleString()],
    ['Correlation', selectedCorrelationIds.join(', ') || '-']
  ];
  document.getElementById('details').innerHTML = renderDetailsGrid(rows);
  document.getElementById('raw').textContent = JSON.stringify(event, null, 2);
}

function showOperation(operation) {
  updateSelectedOperationDetails(operation);
  render();
}

function updateSelectedOperationDetails(operation) {
  selectedId = operation.id;
  selectedCorrelationIds = compactStrings(operation.correlationIds || []);
  const rows = [
    ['Kind', 'Operation'],
    ['Tool', operation.tool],
    ['Thread', operation.threadLabel],
    ['Status', operation.status],
    ['Duration', formatDuration(operation.durationMs)],
    ['Started', new Date(operation.start).toLocaleString()],
    ['Arguments', previewValue((operation.call.summary || {}).arguments)],
    ['Output', operation.result ? previewValue((operation.result.summary || {}).output) : '-'],
    ['Correlation', selectedCorrelationIds.join(', ') || '-']
  ];
  document.getElementById('details').innerHTML = renderDetailsGrid(rows);
  document.getElementById('raw').textContent = JSON.stringify(operation, null, 2);
}

function updateRelatedButton() {
  const button = document.getElementById('related-toggle');
  button.className = 'details-action' + (relatedOnly ? ' active' : '');
  button.disabled = selectedCorrelationIds.length === 0;
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
  const buckets = buildEventDensityBuckets(items, sessionStart, sessionEnd, width);
  const max = Math.max(1, ...buckets.map((bucket) => bucket.count));
  buckets.forEach((bucket, x) => {
    if (!bucket.count) return;
    ctx.fillStyle = bucket.error ? '#f87171' : bucket.subagent ? '#f472b6' : bucket.tool ? '#facc15' : bucket.skill ? '#22d3ee' : '#38bdf8';
    const barHeight = Math.max(2, bucket.count / max * (height - 8));
    ctx.fillRect(x, height - barHeight - 4, 1, barHeight);
  });
}

function renderOverviewWindow(sessionStart, sessionEnd) {
  const rect = overviewCanvas.getBoundingClientRect();
  const span = Math.max(1, sessionEnd - sessionStart);
  const start = draftViewportStart ?? viewportStart;
  const end = draftViewportEnd ?? viewportEnd;
  const rawLeft = (start - sessionStart) / span * rect.width;
  const rawWidth = (end - start) / span * rect.width;
  const visualWidth = Math.min(rect.width, Math.max(minOverviewWindowPx, rawWidth));
  const centeredLeft = rawLeft + rawWidth / 2 - visualWidth / 2;
  const visualLeft = Math.max(0, Math.min(rect.width - visualWidth, centeredLeft));
  overviewWindow.style.left = (overviewCanvas.offsetLeft + visualLeft) + 'px';
  overviewWindow.style.width = visualWidth + 'px';
}

function overviewDragHitMode(event) {
  if (event.target.classList.contains('left')) return 'resize-left';
  if (event.target.classList.contains('right')) return 'resize-right';
  if (event.target.closest('#overview-window')) return 'move';
  return 'select';
}

function startOverviewRangeSelection(event) {
  event.preventDefault();
  overviewDragMode = overviewDragHitMode(event);
  overviewDragStart = overviewMsFromClientX(event.clientX);
  overviewMoveAnchor = overviewDragStart;
  overviewDragFrameStart = viewportStart;
  overviewDragFrameEnd = viewportEnd;
  if (overviewShell.setPointerCapture && event.pointerId != null) overviewShell.setPointerCapture(event.pointerId);
  if (overviewDragMode === 'select') previewViewport(overviewDragStart, overviewDragStart + Math.max(minViewportMs, (traceEnd - traceStart) * .05));
}

function updateOverviewRangeSelection(event) {
  if (overviewDragStart == null) return;
  const current = overviewMsFromClientX(event.clientX);
  let nextStart = overviewDragFrameStart;
  let nextEnd = overviewDragFrameEnd;
  if (overviewDragMode === 'resize-left') nextStart = Math.min(current, overviewDragFrameEnd - minViewportMs);
  else if (overviewDragMode === 'resize-right') nextEnd = Math.max(current, overviewDragFrameStart + minViewportMs);
  else if (overviewDragMode === 'move') {
    const width = overviewDragFrameEnd - overviewDragFrameStart;
    nextStart = Math.max(traceStart, Math.min(traceEnd - width, overviewDragFrameStart + current - overviewMoveAnchor));
    nextEnd = nextStart + width;
  } else {
    nextStart = overviewDragStart;
    nextEnd = current;
  }
  previewViewport(nextStart, nextEnd);
}

function previewViewport(startMs, endMs) {
  draftViewportStart = Math.max(traceStart, Math.min(startMs, endMs));
  draftViewportEnd = Math.min(traceEnd, Math.max(startMs, endMs));
  if (draftViewportEnd - draftViewportStart < minViewportMs) draftViewportEnd = Math.min(traceEnd, draftViewportStart + minViewportMs);
  viewportStart = draftViewportStart;
  viewportEnd = draftViewportEnd;
  clampViewport();
  draftViewportStart = viewportStart;
  draftViewportEnd = viewportEnd;
  render();
}

function finishOverviewRangeSelection(event) {
  draftViewportStart = null;
  draftViewportEnd = null;
  if (event && event.pointerId != null && overviewShell.hasPointerCapture && overviewShell.hasPointerCapture(event.pointerId)) overviewShell.releasePointerCapture(event.pointerId);
  overviewDragStart = null;
  overviewDragMode = '';
}

function handleOverviewWheel(event) {
  event.preventDefault();
  if (event.ctrlKey || event.metaKey) {
    zoomViewportAt(event.clientX, viewportSpan() * (event.deltaY < 0 ? .8 : 1.25));
    return;
  }
  const delta = Math.abs(event.deltaX) > Math.abs(event.deltaY) ? event.deltaX : event.deltaY;
  panViewport(delta / Math.max(1, overviewCanvas.getBoundingClientRect().width) * Math.max(minViewportMs, traceEnd - traceStart));
}

async function importSession(path) {
  const statusNode = document.getElementById('import-status');
  statusNode.textContent = 'Importing...';
  const res = await fetch('/api/import', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ path }) });
  await applyImportResponse(res, statusNode);
}

const droppedRelativePaths = new WeakMap();

async function importUploadedFiles(files) {
  const list = Array.from(files || []).filter(Boolean);
  if (!list.length) return;
  const statusNode = document.getElementById('import-status');
  statusNode.textContent = 'Importing ' + list.length + ' file' + (list.length === 1 ? '' : 's') + '...';
  const form = new FormData();
  list.forEach((file) => form.append('file', file, droppedRelativePaths.get(file) || file.webkitRelativePath || file.name));
  const res = await fetch('/api/import/upload', { method: 'POST', body: form });
  await applyImportResponse(res, statusNode);
}

async function importUploadedFile(file) {
  return importUploadedFiles(file ? [file] : []);
}

async function applyImportResponse(res, statusNode) {
  const body = await res.json().catch(() => ({}));
  if (!res.ok) throw new Error(body.error || 'Import failed');
  selectedId = '';
  selectedCorrelationIds = [];
  relatedOnly = false;
  viewportStart = null;
  viewportEnd = null;
  if (eventStream) {
    eventStream.close();
    eventStream = null;
  }
  await loadSession();
  await loadEvents();
  connectEventStream();
  statusNode.textContent = 'Imported ' + body.eventCount + ' events';
}

function appendEvent(event) {
  if (allEvents.some((item) => item.id === event.id)) return;
  allEvents.push(event);
  const shouldFollow = viewportEnd >= traceEnd - 1000;
  rebuildViewModel();
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
  if (eventStream) {
    eventStream.close();
    eventStream = null;
  }
  if (!window.EventSource || sessionMetadata.mode !== 'live') return;
  eventStream = new EventSource('/api/events/stream');
  eventStream.onmessage = (message) => appendEvent(JSON.parse(message.data));
}

function typeBadge(type) {
  const labels = { user_message: 'USER', agent_turn: 'AI', tool_call: 'TOOL', tool_result: 'RESULT', skill: 'SKILL', subagent: 'AGENT', permission: 'PERM', hook: 'HOOK', process: 'PROC', api_request: 'API', api_response: 'API', error: 'ERR' };
  return labels[type] || String(type || '').toUpperCase();
}

function eventBadgeLabel(event) {
  if (event.type === 'skill') {
    const phase = (event.summary || {}).phase;
    if (phase === 'available') return 'SKILL LOAD';
    if (phase === 'invoked') return 'SKILL USE';
  }
  return typeBadge(event.type);
}

function eventBadgeClass(event) {
  const phase = event.type === 'skill' ? (event.summary || {}).phase : '';
  const phaseClass = phase === 'available' || phase === 'invoked' ? ' skill-' + phase : '';
  return 'badge ' + event.type + phaseClass;
}

function eventSummary(event) {
  const summary = event.summary || {};
  if (event.type === 'skill') {
    const parts = [summary.phase || 'skill', summary.description || summary.path || ''].filter(Boolean);
    return shortText(parts.join(' · '), 180);
  }
  return summary.text || summary.description || summary.tool || summary.phase || summary.payloadType || '';
}

function operationSummary(operation) {
  const args = previewValue((operation.call.summary || {}).arguments);
  const output = operation.result ? previewValue((operation.result.summary || {}).output) : 'waiting for result';
  return shortText(args + ' -> ' + output, 180);
}

function renderDetailsGrid(rows) {
  return '<dl class="details-grid">' + rows.map(([k, v]) => '<dt>' + escapeHtml(k) + '</dt><dd>' + escapeHtml(String(v || '-')) + '</dd>').join('') + '</dl>';
}

function formatClockTime(timestampMs) {
  return new Date(timestampMs).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit' });
}

function formatDuration(ms) {
  if (!ms) return 'instant';
  if (ms < 1000) return Math.round(ms) + ' ms';
  return (ms / 1000).toFixed(ms < 10000 ? 1 : 0) + ' s';
}

function previewValue(value) {
  if (value == null || value === '') return '';
  if (typeof value === 'string') return shortText(value, 220);
  return shortText(JSON.stringify(value), 220);
}

function shortText(value, maxLength) {
  const text = String(value || '').replace(/\s+/g, ' ').trim();
  return text.length > maxLength ? text.slice(0, maxLength - 1) + '…' : text;
}

function compactStrings(values) {
  return Array.from(new Set((values || []).filter(Boolean)));
}

function escapeHtml(value) {
  return String(value || '').replace(/[&<>"']/g, (ch) => ({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#039;'}[ch]));
}

document.querySelectorAll('.filter').forEach((button) => {
  button.onclick = () => {
    setEventTypeFilter(button.dataset.filter);
    render();
    if (currentFilter === 'subagent') {
      const item = allStreamItems().find((entry) => entry.kind === 'event' && entry.event.type === 'subagent');
      if (item) focusStreamItem(item);
    }
  };
});

function setEventTypeFilter(nextFilter) {
  currentFilter = nextFilter || 'all';
  document.querySelectorAll('.filter').forEach((item) => item.classList.toggle('active', item.dataset.filter === currentFilter));
}

document.getElementById('import-form').addEventListener('submit', async (event) => {
  event.preventDefault();
  const path = document.getElementById('import-path').value.trim();
  if (!path) {
    document.getElementById('import-status').textContent = 'Path required';
    return;
  }
  try {
    await importSession(path);
  } catch (err) {
    document.getElementById('import-status').textContent = err.message || 'Import failed';
  }
});

document.getElementById('file-choose').onclick = () => document.getElementById('import-file').click();
document.getElementById('folder-choose').onclick = () => document.getElementById('import-folder').click();
document.getElementById('import-file').addEventListener('change', async (event) => {
  const files = event.target.files;
  event.target.value = '';
  if (!files || !files.length) return;
  try {
    await importUploadedFiles(files);
  } catch (err) {
    document.getElementById('import-status').textContent = err.message || 'Import failed';
  }
});
document.getElementById('import-folder').addEventListener('change', async (event) => {
  const files = event.target.files;
  event.target.value = '';
  if (!files || !files.length) return;
  try {
    await importUploadedFiles(files);
  } catch (err) {
    document.getElementById('import-status').textContent = err.message || 'Import failed';
  }
});

function hasDroppedFiles(event) {
  return event.dataTransfer && Array.from(event.dataTransfer.types || []).includes('Files');
}

let dragDepth = 0;

function setDropZoneDragging(active) {
  const overlay = document.getElementById('import-dropzone');
  overlay.classList.toggle('dragging', active);
  overlay.setAttribute('aria-hidden', active ? 'false' : 'true');
}

async function filesFromDrop(event) {
  const items = Array.from((event.dataTransfer && event.dataTransfer.items) || []);
  const entries = items.map((item) => item.webkitGetAsEntry && item.webkitGetAsEntry()).filter(Boolean);
  if (!entries.length) return Array.from((event.dataTransfer && event.dataTransfer.files) || []);
  const nested = await Promise.all(entries.map((entry) => readEntryFiles(entry, '')));
  const files = nested.flat();
  return files.length ? files : Array.from((event.dataTransfer && event.dataTransfer.files) || []);
}

function readEntryFiles(entry, prefix) {
  return new Promise((resolve) => {
    if (entry.isFile) {
      entry.file((file) => {
        droppedRelativePaths.set(file, prefix + file.name);
        resolve([file]);
      }, () => resolve([]));
      return;
    }
    if (!entry.isDirectory) {
      resolve([]);
      return;
    }
    const reader = entry.createReader();
    const files = [];
    const readBatch = () => {
      reader.readEntries(async (entries) => {
        if (!entries.length) {
          resolve(files);
          return;
        }
        const children = await Promise.all(entries.map((child) => readEntryFiles(child, prefix + entry.name + '/')));
        children.forEach((childFiles) => files.push(...childFiles));
        readBatch();
      }, () => resolve(files));
    };
    readBatch();
  });
}

document.addEventListener('dragenter', (event) => {
  if (!hasDroppedFiles(event)) return;
  event.preventDefault();
  dragDepth++;
  setDropZoneDragging(true);
});
document.addEventListener('dragover', (event) => {
  if (!hasDroppedFiles(event)) return;
  event.preventDefault();
  if (!document.getElementById('import-dropzone').classList.contains('dragging')) setDropZoneDragging(true);
});
document.addEventListener('dragleave', (event) => {
  if (!hasDroppedFiles(event)) return;
  dragDepth = Math.max(0, dragDepth - 1);
  if (dragDepth === 0 || !event.relatedTarget) setDropZoneDragging(false);
});
document.addEventListener('drop', async (event) => {
  if (!hasDroppedFiles(event)) return;
  event.preventDefault();
  dragDepth = 0;
  setDropZoneDragging(false);
  const files = await filesFromDrop(event);
  if (!files.length) return;
  try {
    await importUploadedFiles(files);
  } catch (err) {
    document.getElementById('import-status').textContent = err.message || 'Import failed';
  }
});

document.getElementById('related-toggle').onclick = () => {
  if (!selectedCorrelationIds.length) return;
  relatedOnly = !relatedOnly;
  render();
};

const overviewShell = document.querySelector('.overview-shell');
const overviewCanvas = document.getElementById('overview-canvas');
const overviewWindow = document.getElementById('overview-window');
overviewShell.addEventListener('pointerdown', startOverviewRangeSelection);
overviewShell.addEventListener('wheel', handleOverviewWheel, { passive: false });
window.addEventListener('pointermove', updateOverviewRangeSelection);
window.addEventListener('pointerup', finishOverviewRangeSelection);
window.addEventListener('gesturestart', (event) => event.preventDefault());
window.addEventListener('gesturechange', (event) => {
  event.preventDefault();
  zoomViewportAt(window.innerWidth / 2, viewportSpan() / Math.max(.1, event.scale));
});
loadSession().then(() => loadEvents().then(connectEventStream));
</script>
</body>
</html>`
