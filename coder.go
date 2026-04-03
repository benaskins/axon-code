package code

import (
	"io"
	"time"

	loop "github.com/benaskins/axon-loop"
	talk "github.com/benaskins/axon-talk"

	"github.com/benaskins/axon-code/plan"
)

// Config holds runtime configuration for a Coder.
type Config struct {
	MaxIterations      int
	Timeout            time.Duration
	SystemPromptPrefix string
	Verbose            io.Writer
}

// Option applies a configuration change to a Config.
type Option func(*Config)

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
// Wired with axon-loop in step 8; stub returns empty string until then.
func (c *Coder) Implement(projectDir string, step plan.Step, feedback string) (string, error) {
	_ = loop.RunConfig{} // placeholder; fully wired in step 8
	return "", nil
}
