package code

import (
	"context"
	"fmt"
	"io"
	"time"

	loop "github.com/benaskins/axon-loop"
	tool "github.com/benaskins/axon-tool"
	talk "github.com/benaskins/axon-talk"

	"github.com/benaskins/axon-code/internal/prompt"
	internaltools "github.com/benaskins/axon-code/internal/tools"
	"github.com/benaskins/axon-code/plan"
)

// Config holds runtime configuration for a Coder.
type Config struct {
	Model              string
	MaxIterations      int
	Timeout            time.Duration
	SystemPromptPrefix string
	Verbose            io.Writer
	GoAST              bool
}

// Option applies a configuration change to a Config.
type Option func(*Config)

// Compile-time check that *Coder satisfies Agent.
var _ Agent = (*Coder)(nil)

// Coder implements the Agent interface using axon-loop, axon-talk, and axon-tool.
type Coder struct {
	client talk.LLMClient
	cfg    Config
}

// New constructs a Coder with the given LLM client and options.
// Default: MaxIterations=50, Timeout=15min.
func New(client talk.LLMClient, opts ...Option) *Coder {
	cfg := Config{
		MaxIterations: 50,
		Timeout:       15 * time.Minute,
	}
	for _, opt := range opts {
		opt(&cfg)
	}
	return &Coder{client: client, cfg: cfg}
}

// Config returns the current configuration.
func (c *Coder) Config() Config {
	return c.cfg
}

// Implement runs the coding agent loop for the given plan step.
// It builds the tool registry, assembles the system prompt, runs axon-loop,
// and returns the done summary on success.
func (c *Coder) Implement(projectDir string, step plan.Step, feedback string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.cfg.Timeout)
	defer cancel()

	// Build tools bound to projectDir.
	toolDefs, signal, err := internaltools.Build(projectDir, internaltools.Config{
		GoAST: c.cfg.GoAST,
	})
	if err != nil {
		return "", fmt.Errorf("implement: build tools: %w", err)
	}

	// Convert slice to map as required by loop.RunConfig.
	toolMap := make(map[string]tool.ToolDef, len(toolDefs))
	for _, td := range toolDefs {
		toolMap[td.Name] = td
	}

	// Wrap the done tool to cancel the context immediately when called,
	// so the loop exits without making an extra LLM round trip.
	if doneTool, ok := toolMap["done"]; ok {
		origExec := doneTool.Execute
		doneTool.Execute = func(tc *tool.ToolContext, args map[string]any) tool.ToolResult {
			res := origExec(tc, args)
			cancel()
			return res
		}
		toolMap["done"] = doneTool
	}

	systemPrompt := prompt.Build(c.cfg.SystemPromptPrefix, step, feedback)

	req := &loop.Request{
		Model: c.cfg.Model,
		Messages: []loop.Message{
			{Role: loop.RoleSystem, Content: systemPrompt},
			{Role: loop.RoleUser, Content: step.Title + ": " + step.Description},
		},
		MaxIterations: c.cfg.MaxIterations,
	}

	var cb loop.Callbacks
	if c.cfg.Verbose != nil {
		w := c.cfg.Verbose
		cb.OnToolUse = func(name string, args map[string]any) {
			fmt.Fprintf(w, "tool: %s\n", name)
		}
	}

	_, loopErr := loop.Run(ctx, loop.RunConfig{
		Client:    c.client,
		Request:   req,
		Tools:     toolMap,
		ToolCtx:   &tool.ToolContext{Ctx: ctx},
		Callbacks: cb,
	})

	// If the done tool was called, return the summary regardless of loop error
	// (context cancellation is expected when done exits the loop early).
	if signal.Done {
		return signal.Summary, nil
	}

	if loopErr != nil {
		return "", fmt.Errorf("implement: %w", loopErr)
	}

	return "", fmt.Errorf("implement: loop completed without done signal")
}
