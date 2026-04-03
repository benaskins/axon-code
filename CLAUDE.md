# Toolmaker: Axon Component Builder

You are building an axon component. Axon is a suite of Go libraries for AI-powered services. Each component is a focused, composable module published as its own Go module under `github.com/benaskins/`.

## Identity

- Module path: always `github.com/benaskins/{name}`
- Every axon module name is exactly four letters: loop, talk, tool, fact, auth, memo, look, task, gate, mind, lens, wire, synd, eval, tape, rule, chat, book, push, face, base, lore, sign, scan, cost
- Do not change the module path. Do not invent org names.

## Structure

Axon libraries are NOT services. They have no `cmd/` directory, no `main()`, no HTTP server. They are imported by other code.

Typical layout:
```
axon-{name}/
  {package}/         # one or more focused packages
  {package}_test.go  # tests alongside code
  go.mod
  go.sum
  justfile
  CLAUDE.md
  AGENTS.md
  README.md
  plans/
```

Some modules are a single package at the root. Others have sub-packages for distinct concerns (e.g. axon-base has pool/, migration/, repository/, scan/).

## Code Style

- Explicit over implicit. No reflection for struct mapping. No `SELECT *`. No query builders.
- Interfaces as contracts, not abstractions for their own sake. Only define an interface when there are (or will be) multiple implementations.
- Error wrapping with context: `fmt.Errorf("operation: %w", err)`.
- Context propagation: all blocking operations take `context.Context`.
- No third-party assertion libraries (testify, gomega). Use standard `testing` package.
- No testcontainers. Integration tests use the workbench Postgres at `localhost:5433` (database `workbench`, user `postgres`). Never port 5432, that is the core database. Skip gracefully if unavailable.

## Dependencies

- Only depend on standard library and the specific external libraries the PRD calls for (e.g. pgx, golang-migrate)
- Depend on other axon modules only when the PRD requires composition (e.g. axon-cost depends on axon-fact for event emission)
- Use replace directives in go.mod pointing to `~/dev/lamina/{name}` for local axon dependencies
- Do not add axon (the HTTP toolkit) as a dependency unless this is a service. Libraries stay independent.

## Testing

- Write tests first (TDD). Every public function has tests.
- Tests that need external services (Postgres, NATS) skip with `t.Skip()` when the service is unreachable. These are the real tests, not a separate tier.
- Do not create a separate `integration/` test package. Per-package tests that hit real services are the integration tests.
- For database libraries, do not mock the database. The SQL running against real Postgres is the test. Mock the `Repository` interface at the application boundary, not inside the library.
- Use `t.TempDir()` for filesystem tests, never write outside of it.
- No third-party assertion libraries. Standard `testing` package only.

## Publishing

After build, this module will be:
1. Pushed to `github.com/benaskins/{name}`
2. Tagged with a version
3. Added to the axon catalogue in `luthier/catalogues/axon.yaml`
4. Available for composition in future scaffolds

---

# CLAUDE.md

## What This Is

Create the Go module at the repo root: `go.mod` declaring module path (e.g. `github.com/org/axon-code`), import axon-loop, axon-talk, axon-tool. Define the local `Agent` interface in `agent.go` with `Implement(projectDir string, step plan.Step, feedback string) (string, error)`. Define `plan.Step` struct locally (Title, Description string fields to match maestro's shape). Define the `Coder` struct with constructor `New(client talk.LLMClient, opts ...Option) *Coder`. Add a `Config` struct and `Option` functional-options type covering: `MaxIterations` (default 50), `Timeout` (default 15 min), `SystemPromptPrefix` string, `Verbose io.Writer`. Verify: `go build ./...` passes with no implementation bodies yet (stub returns).

## Module

- Module path: `github.com/benaskins/axon-code`
- Project type: library

## Build & Run

```bash
just build     # builds to bin/axon-code
just install   # installs to ~/.local/bin/axon-code
just test      # run tests
just vet       # lint
```

## Constraints

These constraints are extracted from the PRD. Follow them strictly during implementation.

- Go library only — no main package, no HTTP server, no CLI entry point. Do not import axon (the HTTP platform).
- Must NOT import github.com/benaskins/maestro. Define a local compatible Agent interface and rely on Go structural typing for interface satisfaction.
- All file-system tool implementations must resolve paths relative to the project directory. Path traversal via ../ must be detected and rejected before any OS call.
- The bash tool must set cwd to the project directory and must not allow commands that escape the sandbox.
- Tool definitions must follow axon-tool conventions: ToolDef with ParameterSchema and Execute function. No ad-hoc tool wiring.
- Tests must not write outside t.TempDir(). All test file operations must be scoped to a temporary directory.
- Depends only on axon-loop, axon-talk, axon-tool, and the Go standard library. No additional third-party dependencies.
## Plan

See `plans/` for commit-sized implementation steps.

## Framework: Axon/Lamina (go 1.26)

### Components in Use

- **axon-loop**: Core conversation loop that orchestrates LLM turns: receives system prompt + step description, drives tool calls, and iterates until the agent calls the 'done' tool or hits max iterations.
- **axon-talk**: Provider-agnostic LLM adapter. The library accepts a talk.LLMClient at construction time so callers can use Anthropic, OpenRouter/Qwen, Ollama, or Cloudflare AI Gateway without code changes.
- **axon-tool**: Defines and executes all coding agent tools: read_file, write_file, edit_file, list_dir, grep, glob, bash, and done. Each tool is a ToolDef with ParameterSchema and Execute function following axon-tool conventions.

### Patterns

- **HTTP service**: axon.ListenAndServe + axon.MustLoadConfig
- **CLI tool**: main.go with os.Args or flag parsing. No axon import needed.
- **LLM conversation**: axon-loop + axon-talk + axon-tool (all three required). The loop orchestrates turns, talk connects to the LLM provider, tool defines the structured actions the model can take. Selecting axon-loop without axon-tool means the model has no tools to call and cannot produce structured output.
- **Async/background work**: axon-task + axon-fact; never block HTTP handlers
- **Authentication**: axon-auth (WebAuthn/passkeys)
- **Event audit trail / replay**: axon-fact projectors
- **Cross-session memory**: axon-memo
- **Cross-instance fan-out**: axon-nats
- **Process supervision**: aurelia service YAML
- **Deterministic logic**: Go code, no LLM needed
- **Non-deterministic logic**: axon-loop, never raw LLM calls

### File Conventions

- `main.go`: Entry point. HTTP services: imports axon, calls axon.ListenAndServe. CLI tools: parses args, wires deps, runs pipeline.
- `justfile`: build, install, test targets using just
- `AGENTS.md`: Architecture, module selections, boundaries, dep graph
- `CLAUDE.md`: Working instructions for Claude Code
- `README.md`: What it is, how to run it
- `plans/YYYY-MM-DD-initial-build.md`: Commit-sized plan steps

### Boundary Notes

The boundary between a caller and axon-loop is always non-det.
The boundary between axon-loop and axon-talk is det (provider selection is deterministic).
The boundary between axon-tool and its tool implementations depends on what the tools do.


## Practice

Execute the plan one step at a time. Each step is a TDD cycle that ends with a clean commit.

1. Read the plan. Pick up the next incomplete step.
2. Write a failing test first, then make it pass, then clean up. Run the full test suite before committing.
3. Wire new code into the entrypoint immediately. Every step should produce a program that builds, runs, and does something observable end-to-end. Do not defer integration to later steps.
4. Review your change for reuse, quality, and efficiency before committing.
5. Run `git status`. Only stage files related to this step.
6. One commit per plan step. Use conventional commit messages (feat/fix/refactor/test/infra/config prefix).
7. Move to the next step.

Stop if:
- A step reveals a design question the plan did not anticipate
- Tests are failing for reasons unrelated to the current step
