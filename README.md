# axon-code

Create the Go module at the repo root: `go.mod` declaring module path (e.g. `github.com/org/axon-code`), import axon-loop, axon-talk, axon-tool. Define the local `Agent` interface in `agent.go` with `Implement(projectDir string, step plan.Step, feedback string) (string, error)`. Define `plan.Step` struct locally (Title, Description string fields to match maestro's shape). Define the `Coder` struct with constructor `New(client talk.LLMClient, opts ...Option) *Coder`. Add a `Config` struct and `Option` functional-options type covering: `MaxIterations` (default 50), `Timeout` (default 15 min), `SystemPromptPrefix` string, `Verbose io.Writer`. Verify: `go build ./...` passes with no implementation bodies yet (stub returns).

## Prerequisites

- Go 1.24+
- [just](https://github.com/casey/just)

## Build & Run

```bash
just build
just install
axon-code --help
```

## Development

```bash
just test   # run tests
just vet    # run go vet
```
