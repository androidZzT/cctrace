# cctrace

`cctrace` is a real-time profiler for AI coding agents. The MVP wraps Claude Code or Codex commands and shows process and trace events in a local waterfall UI.

## Commands

Run all tests:

```sh
go test ./...
```

Run the CLI in development:

```sh
go run ./cmd/cctrace claude -- claude
```

Run a smoke command:

```sh
go run ./cmd/cctrace claude -- sh -c 'printf cctrace-smoke'
```

Replay a saved session:

```sh
go run ./cmd/cctrace view <session-id>
```

## MVP status

The first implementation establishes the trace model, store, process collection, parsers, correlation, and a local waterfall UI. ccglass and transcript parsing are represented by tested parser entry points; deeper live tailing integration is the next iteration after the MVP pipeline is working end to end.
