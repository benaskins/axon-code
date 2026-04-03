package code_test

import (
	"testing"
	"time"

	code "github.com/benaskins/axon-code"
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
