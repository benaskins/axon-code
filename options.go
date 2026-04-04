package code

import (
	"io"
	"time"
)

// WithModel sets the model name for the LLM request.
func WithModel(m string) Option {
	return func(c *Config) { c.Model = m }
}

// WithMaxIterations sets the maximum number of loop iterations.
func WithMaxIterations(n int) Option {
	return func(c *Config) { c.MaxIterations = n }
}

// WithTimeout sets the timeout for a single Implement call.
func WithTimeout(d time.Duration) Option {
	return func(c *Config) { c.Timeout = d }
}

// WithSystemPromptPrefix prepends a custom string to the system prompt.
func WithSystemPromptPrefix(s string) Option {
	return func(c *Config) { c.SystemPromptPrefix = s }
}

// WithVerbose sets the writer for verbose tool-use logging.
func WithVerbose(w io.Writer) Option {
	return func(c *Config) { c.Verbose = w }
}

// WithGoAST enables AST-aware tools for Go source files.
func WithGoAST() Option {
	return func(c *Config) { c.GoAST = true }
}
