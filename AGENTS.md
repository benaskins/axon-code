# axon-code

Create the Go module at the repo root: `go.mod` declaring module path (e.g. `github.com/org/axon-code`), import axon-loop, axon-talk, axon-tool. Define the local `Agent` interface in `agent.go` with `Implement(projectDir string, step plan.Step, feedback string) (string, error)`. Define `plan.Step` struct locally (Title, Description string fields to match maestro's shape). Define the `Coder` struct with constructor `New(client talk.LLMClient, opts ...Option) *Coder`. Add a `Config` struct and `Option` functional-options type covering: `MaxIterations` (default 50), `Timeout` (default 15 min), `SystemPromptPrefix` string, `Verbose io.Writer`. Verify: `go build ./...` passes with no implementation bodies yet (stub returns).

## Build & Test

```bash
go test ./...
go vet ./...
just build     # builds to bin/axon-code
just install   # copies to ~/.local/bin/axon-code
```

## Module Selections

- **axon-loop**: Core conversation loop that orchestrates LLM turns: receives system prompt + step description, drives tool calls, and iterates until the agent calls the 'done' tool or hits max iterations. (non-deterministic)
- **axon-talk**: Provider-agnostic LLM adapter. The library accepts a talk.LLMClient at construction time so callers can use Anthropic, OpenRouter/Qwen, Ollama, or Cloudflare AI Gateway without code changes. (deterministic)
- **axon-tool**: Defines and executes all coding agent tools: read_file, write_file, edit_file, list_dir, grep, glob, bash, and done. Each tool is a ToolDef with ParameterSchema and Execute function following axon-tool conventions. (deterministic)

## Deterministic / Non-deterministic Boundary

| From | To | Type |
|------|----|------|
| coder.Agent (Implement method) | axon-loop | non-det |
| axon-loop | axon-talk | det |
| axon-loop | axon-tool | det |
| axon-tool | file system tools (read_file, write_file, edit_file, list_dir) | det |
| axon-tool | search tools (grep, glob) | det |
| axon-tool | bash tool (shell execution) | non-det |
| sandbox (path resolver) | file system tools (read_file, write_file, edit_file, list_dir) | det |

