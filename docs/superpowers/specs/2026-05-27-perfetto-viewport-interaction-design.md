# Perfetto Viewport Interaction Design

## Goal

Make cctrace usable on large Claude sessions by replacing the current scroll-width timeline with a Perfetto-style viewport model and reducing DOM work during pan, zoom, and overview interactions.

## Root Cause

The current UI freezes because each render rebuilds the overview, ruler, lanes, and visible event DOM. On the current session, `/api/events` yields thousands of events across more than 21 hours, and even the latest five-minute window contains hundreds of events. The overview currently creates one DOM node per event, and event titles can contain tens of thousands of characters, which makes each redraw expensive.

## Interaction Model

The timeline has a single time viewport: `viewportStart` and `viewportEnd` in epoch milliseconds. The viewport replaces the current scroll-width/scrollLeft model.

- Wheel or trackpad gestures pan the viewport horizontally.
- Shift+wheel also pans horizontally.
- Ctrl/Meta+wheel zooms around the pointer timestamp.
- Pinch gestures zoom around the center or pointer-equivalent anchor.
- Dragging blank timeline space pans the viewport.
- The overview shows the whole trace and a draggable viewport window.
- Dragging the overview window pans the viewport.
- Dragging outside the overview window selects a new viewport range.

## Rendering Model

After loading events, the UI builds a lightweight render model once per event. Each render item includes:

- original event reference
- numeric start/end timestamps
- lane key
- lane label
- short title for DOM labels and tooltips

The full event payload remains available for the selected event details panel, but timeline blocks never render full transcript text directly.

## Performance Strategy

The main timeline renders only events intersecting `[viewportStart, viewportEnd]`. The overview is a canvas density view rather than one DOM node per event. Viewport changes redraw only the timeline elements required for the current window and the overview selection overlay.

The implementation will remove the per-event overview DOM path and avoid creating enormous DOM attributes from unbounded event titles.

## Scope

In scope:

- viewport state and pan/zoom helpers
- Perfetto-style wheel, pinch, drag, and overview interactions
- canvas overview density rendering
- bounded timeline labels/tooltips
- regression tests for the presence of the new architecture and absence of the old per-event overview path
- current-session verification

Out of scope:

- full Perfetto SQL/query engine
- web workers
- canvas-rendered main tracks
- persistent UI preferences
- server-side pagination

## Testing

Tests should verify the served HTML includes the new viewport functions and canvas overview elements, and no longer includes the per-event overview DOM render path. Full verification must run `go test ./...` and a current-session `cctrace view` smoke check.