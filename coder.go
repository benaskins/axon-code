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
	SessionID          string
	Verbose            io.Writer
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
		MaxIterations: 100,
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
func (c *Coder) Implement(projectDir string, step plan.Step, feedback string) (*ImplementResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.cfg.Timeout)
	defer cancel()

	// Build tools bound to projectDir.
	toolDefs, signal, err := internaltools.Build(projectDir, internaltools.Config{})
	if err != nil {
		return nil, fmt.Errorf("implement: build tools: %w", err)
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

	profile := prompt.ForModel(c.cfg.Model)
	systemPrompt := prompt.Build(c.cfg.SystemPromptPrefix, step, feedback, profile)

	// Filter tools based on profile (e.g. Gemini excludes edit_file, rewrite).
	if len(profile.ExcludeTools) > 0 {
		exclude := make(map[string]bool, len(profile.ExcludeTools))
		for _, name := range profile.ExcludeTools {
			exclude[name] = true
		}
		for name := range toolMap {
			if exclude[name] {
				delete(toolMap, name)
			}
		}
	}

	opts := map[string]any{}
	if c.cfg.SessionID != "" {
		opts["session_id"] = c.cfg.SessionID
	}
	// Apply profile overrides (e.g. Gemini needs temp 1.0 and tool_choice auto).
	if profile.Temperature != nil {
		opts["temperature"] = *profile.Temperature
	}
	if profile.ToolChoice != "" {
		opts["tool_choice"] = profile.ToolChoice
	}

	req := &loop.Request{
		Model: c.cfg.Model,
		Messages: []loop.Message{
			{Role: loop.RoleSystem, Content: systemPrompt},
			{Role: loop.RoleUser, Content: step.Title + ": " + step.Description},
		},
		MaxIterations: c.cfg.MaxIterations,
		Options:       opts,
	}

	var cb loop.Callbacks
	if c.cfg.Verbose != nil {
		w := c.cfg.Verbose
		var thinkBuf []byte
		flushThinking := func() {
			if len(thinkBuf) == 0 {
				return
			}
			s := string(thinkBuf)
			thinkBuf = thinkBuf[:0]
			if len(s) > 200 {
				s = s[:200] + "..."
			}
			fmt.Fprintf(w, "thinking: %s\n", s)
		}
		var contentBuf []byte
		flushContent := func() {
			if len(contentBuf) == 0 {
				return
			}
			s := string(contentBuf)
			contentBuf = contentBuf[:0]
			if len(s) > 300 {
				s = s[:300] + "..."
			}
			fmt.Fprintf(w, "reasoning: %s\n", s)
		}
		cb.OnThinking = func(token string) {
			thinkBuf = append(thinkBuf, token...)
		}
		cb.OnToken = func(token string) {
			contentBuf = append(contentBuf, token...)
		}
		cb.OnToolUse = func(name string, args map[string]any) {
			flushThinking()
			flushContent()
			switch name {
			case "read_file", "write_file", "edit_file", "ast", "rewrite":
				path, _ := args["path"].(string)
				if op, ok := args["operation"].(string); ok {
					fmt.Fprintf(w, "tool: %s %s %s\n", name, op, path)
				} else {
					fmt.Fprintf(w, "tool: %s %s\n", name, path)
				}
			case "go_cmd", "git_cmd":
				a, _ := args["args"].(string)
				fmt.Fprintf(w, "tool: %s %s\n", name, a)
			case "grep":
				pattern, _ := args["pattern"].(string)
				fmt.Fprintf(w, "tool: %s %q\n", name, pattern)
			case "glob":
				pattern, _ := args["pattern"].(string)
				fmt.Fprintf(w, "tool: %s %s\n", name, pattern)
			case "inspect_project":
				path, _ := args["path"].(string)
				fmt.Fprintf(w, "tool: %s %s\n", name, path)
			case "lsp_diagnostics":
				path, _ := args["path"].(string)
				fmt.Fprintf(w, "tool: %s %s\n", name, path)
			case "lsp_definition":
				path, _ := args["path"].(string)
				line, _ := args["line"].(float64)
				col, _ := args["column"].(float64)
				fmt.Fprintf(w, "tool: %s %s:%d:%d\n", name, path, int(line), int(col))
			case "done":
				summary, _ := args["summary"].(string)
				if len(summary) > 80 {
					summary = summary[:80] + "..."
				}
				fmt.Fprintf(w, "tool: %s %s\n", name, summary)
			default:
				fmt.Fprintf(w, "tool: %s\n", name)
			}
		}
	}

	loopResult, loopErr := loop.Run(ctx, loop.RunConfig{
		Client:    c.client,
		Request:   req,
		Tools:     toolMap,
		ToolCtx:   &tool.ToolContext{Ctx: ctx},
		Callbacks: cb,
	})

	// If the done tool was called, return the summary regardless of loop error
	// (context cancellation is expected when done exits the loop early).
	if signal.Done {
		result := &ImplementResult{
			Summary: signal.Summary,
			Files:   signal.Files,
		}
		if loopResult != nil {
			result.Usage = loopResult.Usage
		}
		return result, nil
	}

	if loopErr != nil {
		return nil, fmt.Errorf("implement: %w", loopErr)
	}

	return nil, fmt.Errorf("implement: loop completed without done signal")
}
