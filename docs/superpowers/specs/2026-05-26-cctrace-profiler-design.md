# cctrace Real-Time Agent Profiler Design

## Overview

`cctrace` is a real-time profiler for AI coding agents. The first version focuses on Claude Code and Codex power users who want to understand what an agent just did, why it was slow, and where it got blocked.

The tool starts from ccglass' strengths: capturing model API traffic and session data. `cctrace` adds an agent-behavior layer by combining ccglass captures with local Claude Code/Codex transcripts and process events, then presenting the result as a profiler-style timeline waterfall.

## MVP Scope

The MVP is a wrapper-based profiler:

```sh
cctrace claude -- <command>
cctrace codex -- <command>
```

The wrapper starts a profiler server, starts or connects to ccglass capture, injects the required proxy/base-url environment variables, launches the target agent command, and streams normalized trace events to a local web UI.

The MVP does not replace ccglass, provide a team audit platform, or guarantee perfect reconstruction of every hidden agent decision. It should make uncertainty explicit and preserve raw source references so users can inspect the underlying data.

## User Goals

The first target user is a heavy Claude Code/Codex user who wants to answer:

- What did the agent do after my prompt?
- Which tools, skills, subagents, hooks, permissions, and API requests were involved?
- Where did time go?
- Which steps failed, retried, blocked, or produced large outputs?
- Which events are known facts and which are inferred correlations?

## Architecture

The recommended architecture is an independent Go or Rust profiler core with a web UI, using ccglass as a companion data source for the first version. The project should not fork ccglass for the MVP.

### Components

1. **CLI wrapper**
   - Provides `cctrace claude -- <command>` and `cctrace codex -- <command>`.
   - Starts the local profiler server and web UI.
   - Starts or connects to ccglass capture.
   - Injects the provider environment needed by the target agent.
   - Records child process lifecycle events, stdout/stderr metadata, and exit status.

2. **Collectors**
   - `CcglassCollector` watches the active ccglass session for API request/response data, token usage, cache data, latency, cost, and raw request references.
   - `TranscriptCollector` watches Claude Code/Codex local transcripts or logs for tool calls, tool results, permission prompts, hooks, skills, subagents, tasks, and plan events.
   - `ProcessCollector` records wrapper-level process start, exit, error, and command metadata.

3. **Event normalizer**
   - Converts source-specific records into a shared `TraceEvent` model.
   - Preserves `source`, `rawRef`, `confidence`, timestamps, duration, status, and correlation metadata.
   - Allows partially known events so the UI can update in real time.

4. **Correlation engine**
   - Links API requests to nearby agent turns, tool calls, skill triggers, and subagent activity.
   - Starts with heuristics: time windows, event order, message/content hashes, request metadata, and stream completion timing.
   - Uses confidence levels instead of pretending inferred relationships are exact.
   - Keeps unmatched events as orphan timeline entries rather than dropping them.

5. **Profiler server and UI**
   - Streams event additions and corrections to the browser over WebSocket or SSE.
   - Persists completed sessions for `cctrace view <session>` replay.
   - Presents a timeline waterfall as the primary interaction model.

## Data Model

A session contains timeline events that may also form parent-child spans when correlation is known.

```ts
type TraceEvent = {
  id: string
  sessionId: string
  type:
    | "process"
    | "user_message"
    | "agent_turn"
    | "api_request"
    | "api_response"
    | "tool_call"
    | "tool_result"
    | "skill"
    | "subagent"
    | "permission"
    | "hook"
    | "error"
  title: string
  timestamp: number
  endTimestamp?: number
  durationMs?: number
  status: "running" | "ok" | "error" | "blocked" | "unknown"
  source: "ccglass" | "claude_transcript" | "codex_log" | "process" | "derived"
  parentId?: string
  correlationIds: string[]
  confidence: "exact" | "likely" | "possible" | "unknown"
  summary: object
  rawRef?: {
    file?: string
    offset?: number
    requestId?: string
    blobHash?: string
  }
}
```

Events are allowed to be revised. During live capture, `cctrace` may first show a running or possibly correlated event, then update its duration, parent, status, or confidence when more source data arrives.

## Real-Time Flow

1. User runs `cctrace claude -- claude` or an equivalent wrapped command.
2. The CLI creates a `TraceSession`, starts the profiler server/UI, and prepares ccglass capture.
3. The child agent process starts with the necessary environment injected.
4. ccglass captures model API traffic while local transcripts/logs grow.
5. Collectors tail each source and emit source-specific records.
6. The normalizer converts those records to `TraceEvent` updates.
7. The correlation engine links related events and updates confidence.
8. The server pushes additions and revisions to the UI.
9. On process exit, the session is finalized and remains available for replay.

## UI Design

The MVP UI has three regions.

### Trace Tree

The left pane groups events by session, prompt, turn, and inferred child activity. It helps users jump from a prompt to the agent actions that followed.

### Waterfall Timeline

The center pane is the primary view. It shows time horizontally and events as bars. Event colors distinguish categories such as agent turns, API requests, tool calls, skills, subagents, permissions, hooks, and errors. Running events extend live until completed. Concurrent or overlapping events should be visible rather than flattened.

### Details Panel

The right pane shows the selected event's summary, duration, source, raw reference, token/cost/cache fields when available, linked events, and confidence. Large raw outputs are summarized by default and expandable on demand.

## Error Handling

- If ccglass is unavailable, continue with transcript/process events and show that API traffic is unavailable.
- If transcripts/logs are unavailable, continue with ccglass API events and show that agent behavior events are unavailable.
- If parsing fails for a record, emit an `error` event and continue tailing.
- If events cannot be correlated, keep them visible as orphan events with `unknown` or `possible` confidence.
- Permission waits, hook failures, tool errors, subagent failures, and retries are first-class timeline events.
- Redacted or unavailable raw fields should be labeled as unavailable rather than omitted silently.

## Testing Strategy

### Parser and Normalizer Tests

Use fixtures for ccglass session fragments, Claude Code transcript JSONL, and Codex logs. Verify that inputs produce stable `TraceEvent` records with preserved raw references. Cover missing fields, unknown fields, partial JSON, unfinished streams, and absent duration data.

### Correlation Engine Tests

Use synthetic timelines for common cases:

- single prompt with one API request
- tool call surrounded by multiple API requests
- permission blocking
- hook failure
- subagent concurrency
- failed tool followed by retry
- events that cannot be confidently matched

Assertions should cover `parentId`, `correlationIds`, `durationMs`, and `confidence`. Ambiguous cases must not be marked `exact`.

### End-to-End Smoke Test

Run a simple wrapped session or a mock agent fixture. Verify that:

- the UI receives live events
- running events complete
- ccglass and transcript events appear in the same session
- session replay works after process exit
- errors or missing sources are visible rather than fatal

## MVP Completion Criteria

The first implementation is complete when:

- A user can run a wrapped Claude Code or Codex command.
- The profiler UI opens and streams live events.
- The timeline includes process, agent/user turn, API request/response, tool call/result, and error or permission events where the sources expose them.
- ccglass and transcript events are correlated for common single-turn and tool-call scenarios.
- Low-confidence or unmatched events remain visible and clearly labeled.
- A completed session can be reopened for replay.
