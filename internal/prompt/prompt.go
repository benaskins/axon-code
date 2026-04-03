package prompt

import (
	"strings"

	code "github.com/benaskins/axon-code"
	"github.com/benaskins/axon-code/plan"
)

const standardInstructions = `You are a coding agent. Your job is to implement a plan step precisely and completely.

Guidelines:
- Read existing code before modifying it.
- Use the available tools to read, write, edit, search, and run commands in the project directory.
- All file paths are relative to the project directory. Do not attempt to access paths outside it.
- After completing the work, call the done tool with a summary of what you did.
- Be concise. Do not explain what you are about to do — just do it.`

// Build assembles the full system prompt for a coding agent turn.
func Build(cfg code.Config, step plan.Step, feedback string) string {
	var b strings.Builder

	if cfg.SystemPromptPrefix != "" {
		b.WriteString(cfg.SystemPromptPrefix)
		b.WriteString("\n\n")
	}

	b.WriteString(standardInstructions)
	b.WriteString("\n\n")

	b.WriteString("## Task\n\n")
	b.WriteString("**")
	b.WriteString(step.Title)
	b.WriteString("**\n\n")
	b.WriteString(step.Description)

	if feedback != "" {
		b.WriteString("\n\n## Previous Attempt Failed\n\n")
		b.WriteString("The previous attempt did not succeed. Here is what went wrong:\n\n")
		b.WriteString(feedback)
	}

	return b.String()
}
