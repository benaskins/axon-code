package code_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	code "github.com/benaskins/axon-code"
	"github.com/benaskins/axon-code/plan"
	loop "github.com/benaskins/axon-loop"
)

func TestNew_defaults(t *testing.T) {
	c := code.New(nil)
	cfg := c.Config()
	if cfg.MaxIterations != 50 {
		t.Errorf("MaxIterations = %d, want 50", cfg.MaxIterations)
	}
	if cfg.Timeout != 15*time.Minute {
		t.Errorf("Timeout = %v, want 15m", cfg.Timeout)
	}
	if cfg.SystemPromptPrefix != "" {
		t.Errorf("SystemPromptPrefix = %q, want empty", cfg.SystemPromptPrefix)
	}
	if cfg.Verbose != nil {
		t.Errorf("Verbose = %v, want nil", cfg.Verbose)
	}
}

// fakeClient is a multi-turn LLM client for testing.
// Each element in turns is the list of responses for that Chat call.
// It respects context cancellation so the done-tool early-exit works.
type fakeClient struct {
	turns [][]loop.Response
	call  int
}

func (f *fakeClient) Chat(ctx context.Context, req *loop.Request, fn func(loop.Response) error) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if f.call >= len(f.turns) {
		return nil
	}
	responses := f.turns[f.call]
	f.call++
	for _, resp := range responses {
		if err := fn(resp); err != nil {
			return err
		}
	}
	return nil
}

func TestImplement_WritesFileAndReturnsSummary(t *testing.T) {
	dir := t.TempDir()

	client := &fakeClient{
		turns: [][]loop.Response{
			// Turn 1: write_file tool call
			{
				{
					ToolCalls: []loop.ToolCall{
						{
							ID:   "call_1",
							Name: "write_file",
							Arguments: map[string]any{
								"path":    "hello.txt",
								"content": "hello world",
							},
						},
					},
					Done: true,
				},
			},
			// Turn 2: done tool call
			{
				{
					ToolCalls: []loop.ToolCall{
						{
							ID:   "call_2",
							Name: "done",
							Arguments: map[string]any{
								"summary": "wrote hello.txt with greeting",
							},
						},
					},
					Done: true,
				},
			},
			// Turn 3: final response — should not be reached because done cancels context
			{
				{Content: "unreachable", Done: true},
			},
		},
	}

	coder := code.New(client)
	step := plan.Step{
		Title:       "Write greeting",
		Description: "Write hello world into hello.txt.",
	}

	summary, err := coder.Implement(dir, step, "")
	if err != nil {
		t.Fatalf("Implement returned error: %v", err)
	}
	if summary != "wrote hello.txt with greeting" {
		t.Errorf("summary = %q, want %q", summary, "wrote hello.txt with greeting")
	}

	data, readErr := os.ReadFile(filepath.Join(dir, "hello.txt"))
	if readErr != nil {
		t.Fatalf("expected hello.txt to exist: %v", readErr)
	}
	if string(data) != "hello world" {
		t.Errorf("file content = %q, want %q", string(data), "hello world")
	}
}
