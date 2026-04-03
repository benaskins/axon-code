# axon-code — Architecture

## Module Selections

| Module | Role | Boundary |
|--------|------|----------|
| **axon-loop** | Core conversation loop: orchestrates LLM turns, drives tool calls, iterates until `done` or max iterations | non-deterministic |
| **axon-talk** | Provider-agnostic LLM adapter: Anthropic, OpenRouter/Qwen, Ollama, Cloudflare AI Gateway | deterministic |
| **axon-tool** | Tool definitions: `read_file`, `write_file`, `edit_file`, `list_dir`, `grep`, `glob`, `bash`, `done` | deterministic |

## Boundary Classification

| Caller | Callee | Type | Notes |
|--------|--------|------|-------|
| `coder.Implement` | `axon-loop` | non-det | LLM drives the loop |
| `axon-loop` | `axon-talk` | det | provider selection is fixed |
| `axon-loop` | `axon-tool` | det | tool dispatch is deterministic |
| `axon-tool` file tools | `internal/sandbox` | det | path resolution is rule-based |
| `axon-tool` bash tool | OS shell | non-det | shell execution is non-deterministic |

## Dependency Graph

```
axon-code
├── github.com/benaskins/axon-loop   (conversation orchestration)
├── github.com/benaskins/axon-talk   (LLM provider adapter)
├── github.com/benaskins/axon-tool   (tool definitions + execution)
└── internal/
    ├── prompt/   (system prompt builder)
    ├── sandbox/  (path resolver + traversal guard)
    └── tools/    (tool registry builder)
```

No external third-party dependencies beyond axon-loop, axon-talk, axon-tool, and the Go standard library.

## Sandbox Invariant

Every file-system and shell tool resolves its path arguments against the caller-supplied `projectDir` before any OS call. The `internal/sandbox` package rejects any path that escapes `projectDir` via `..` traversal or absolute references. This invariant is enforced in `internaltools.Build` at tool registration time and is tested in `internal/sandbox` and `internal/tools`.
