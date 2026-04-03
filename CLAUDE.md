# axon-code — Working Instructions

## What This Is

`axon-code` is a Go library that implements a coding agent. It provides a `Coder` struct that satisfies the `Agent` interface, using axon-loop for LLM conversation orchestration, axon-talk for provider-agnostic LLM access, and axon-tool for file-system and shell tools sandboxed to a project directory.

## Module Layout

```
axon-code/
  agent.go          # Agent interface definition
  coder.go          # Coder struct + Implement method
  options.go        # Option constructors (WithMaxIterations, etc.)
  plan/
    step.go         # plan.Step struct (Title, Description)
  internal/
    prompt/         # system prompt builder
    sandbox/        # path resolver + traversal guard
    tools/          # tool registry builder
  plans/            # commit-sized implementation steps
  justfile
  go.mod / go.sum
```

## Build & Test

```bash
just build   # go build ./...
just test    # go test ./...
just lint    # go vet ./...
```

## Constraints

- **No maestro import.** The `Agent` interface is defined locally in `agent.go`. Go structural typing satisfies the maestro boundary.
- **No writes outside `t.TempDir()`.** All test file operations must be scoped to a temporary directory.
- **Sandbox all paths.** Every file-system and shell tool must resolve paths through `internal/sandbox`. Reject `../` traversal before any OS call.
- **No third-party dependencies** beyond axon-loop, axon-talk, axon-tool, and the Go standard library.
- **Go library only.** No `main` package, no HTTP server, no CLI entry point.
