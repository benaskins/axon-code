# axon-code — Initial Build Plan
# 2026-04-03

Each step is commit-sized. Execute via `/iterate`.

## Step 1 — Scaffold library module and core interfaces

Create the Go module at the repo root: `go.mod` declaring module path (e.g. `github.com/org/axon-code`), import axon-loop, axon-talk, axon-tool. Define the local `Agent` interface in `agent.go` with `Implement(projectDir string, step plan.Step, feedback string) (string, error)`. Define `plan.Step` struct locally (Title, Description string fields to match maestro's shape). Define the `Coder` struct with constructor `New(client talk.LLMClient, opts ...Option) *Coder`. Add a `Config` struct and `Option` functional-options type covering: `MaxIterations` (default 50), `Timeout` (default 15 min), `SystemPromptPrefix` string, `Verbose io.Writer`. Verify: `go build ./...` passes with no implementation bodies yet (stub returns).

Commit: `feat: scaffold module, Agent interface, Coder struct, and Option types`

## Step 2 — Implement sandbox path resolver

Create `internal/sandbox/sandbox.go`. Implement `Resolve(root, relPath string) (string, error)`: calls `filepath.Clean`, then `filepath.Abs`, then checks the result shares the `root` prefix using `strings.HasPrefix` on the cleaned absolute path. Returns an error if the resolved path escapes root or if relPath is empty. Write unit tests in `internal/sandbox/sandbox_test.go` covering: normal relative path, nested path, path with `../` traversal (must error), absolute path input (must error or treat as relative), empty path (must error). All tests use only in-memory string operations, no disk writes.

Commit: `feat: implement sandbox path resolver with traversal rejection`

## Step 3 — Implement file system tools

Create `internal/tools/fs.go`. Implement four axon-tool `ToolDef`s using `sandbox.Resolve` for every path argument before any OS call: `read_file` (params: path string, offset int optional, limit int optional — read lines offset..offset+limit), `write_file` (params: path string, content string — creates parent dirs if needed), `edit_file` (params: path string, old_string string, new_string string — strings.Replace first occurrence, error if old_string not found), `list_dir` (params: path string — returns names+type of immediate children). Write unit tests in `internal/tools/fs_test.go` using `t.TempDir()` for all disk operations. Test: read round-trips with write_file; edit_file replaces text; edit_file errors on missing old_string; list_dir returns correct entries; read_file with offset/limit returns correct lines; traversal paths error on all tools.

Commit: `feat: implement file system tools (read_file, write_file, edit_file, list_dir)`

## Step 4 — Implement search tools (grep, glob)

Create `internal/tools/search.go`. Implement two axon-tool `ToolDef`s: `grep` (params: pattern string, path string optional default ".", file_type string optional — walks the directory, compiles `regexp.MustCompile(pattern)`, filters by extension if file_type given, returns matching lines with filename:linenum prefix), `glob` (params: pattern string — calls `filepath.Glob` rooted at project dir via sandbox, returns matching paths relative to project root). Both tools resolve their path argument through `sandbox.Resolve`. Write unit tests in `internal/tools/search_test.go` using `t.TempDir()`. Test: grep finds pattern in nested files; grep with file_type filters; grep returns empty on no match; glob matches expected files; both reject traversal paths.

Commit: `feat: implement search tools (grep, glob)`

## Step 5 — Implement bash tool with timeout and cwd

Create `internal/tools/bash.go`. Implement one axon-tool `ToolDef`: `bash` (params: command string, timeout_seconds int optional default from config). Uses `exec.CommandContext` with a derived context capped at the configured timeout. Sets `Cmd.Dir` to the project directory. Captures stdout and stderr separately. Returns a JSON-serialisable struct with fields: stdout, stderr, exit_code. The tool does not parse or interpret the command — it executes via `sh -c`. Write unit tests in `internal/tools/bash_test.go` using `t.TempDir()` as project dir. Test: simple echo command returns stdout; failing command returns non-zero exit_code with stderr; timeout cancels the command and returns an error result; cwd is set to project dir (verify with `pwd`).

Commit: `feat: implement bash tool with configurable timeout and cwd sandboxing`

## Step 6 — Implement done tool and tool registry builder

Create `internal/tools/done.go`. Implement the `done` axon-tool `ToolDef` (params: summary string). The Execute function stores the summary and signals loop exit — implement via a sentinel error or a shared completion state that the loop runner checks. Create `internal/tools/registry.go` with `Build(projectDir string, cfg Config) ([]tool.ToolDef, *DoneSignal, error)` that instantiates all tools bound to `projectDir` and returns the slice plus a `DoneSignal` value the caller can inspect after the loop. Write a unit test verifying that the done tool's Execute sets the summary on DoneSignal correctly.

Commit: `feat: implement done tool and tool registry builder`

## Step 7 — Implement system prompt builder

Create `internal/prompt/prompt.go`. Implement `Build(cfg Config, step plan.Step, feedback string) string` that assembles the full system prompt: optional SystemPromptPrefix prepended if non-empty, then the standard coding instructions (role, task, tool usage guidance, sandboxing reminder), then the step title and description as the task, then — if feedback is non-empty — a clearly labelled section describing what went wrong in the previous attempt. Write unit tests verifying: prefix is prepended when set; feedback section appears only when feedback is non-empty; step title and description always appear; output is deterministic given the same inputs.

Commit: `feat: implement system prompt builder`

## Step 8 — Wire Coder.Implement with axon-loop and tools

Implement `Coder.Implement` in `coder.go`. Steps: (1) build the tool registry via `tools.Build(projectDir, cfg)`, (2) build the system prompt via `prompt.Build(cfg, step, feedback)`, (3) construct an axon-loop instance with the talk.LLMClient, system prompt, tool list, and MaxIterations limit, (4) run the loop — on each iteration check if DoneSignal is set and exit early, (5) if Verbose writer is set, stream tool call names and results to it, (6) apply the step-level timeout via context.WithTimeout wrapping the loop run, (7) return DoneSignal.Summary on success, or an error if max iterations hit without done being called or the timeout expired. Write an integration-style unit test using a fake/mock talk.LLMClient that returns a scripted sequence: one tool call (write_file), then done. Verify Implement returns the expected summary and that the file was written to t.TempDir().

Commit: `feat: implement Coder.Implement wiring axon-loop, tools, and prompt`

## Step 9 — Add Option constructors and interface compliance check

Add exported Option constructors in `options.go`: `WithMaxIterations(n int) Option`, `WithTimeout(d time.Duration) Option`, `WithSystemPromptPrefix(s string) Option`, `WithVerbose(w io.Writer) Option`. Apply defaults in `New()`: MaxIterations=50, Timeout=15*time.Minute. Add a compile-time interface check in `coder.go`: `var _ Agent = (*Coder)(nil)`. Write a test that constructs a Coder with all options and verifies Config fields are set correctly. Verify `go vet ./...` and `go build ./...` pass cleanly.

Commit: `feat: add Option constructors and validate Coder satisfies Agent interface`

## Step 10 — Add justfile, AGENTS.md, CLAUDE.md, README

Add `justfile` with targets: `build` (`go build ./...`), `test` (`go test ./...`), `lint` (`go vet ./...`). Add `AGENTS.md` documenting: module selections (axon-loop, axon-talk, axon-tool), boundary classification table, dependency graph, sandbox invariant. Add `CLAUDE.md` with working instructions: module layout, how to run tests, constraint reminders (no maestro import, no writes outside TempDir, sandbox all paths). Add `README.md` describing what axon-code is, how to import it, and a minimal usage example showing `coder.New(client)` and `coder.Implement(dir, step, "")`. No functional code changes; verify `just test` still passes.

Commit: `infra: add justfile, AGENTS.md, CLAUDE.md, and README`

