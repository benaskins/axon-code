package prompt

import (
	"strings"

	"github.com/benaskins/axon-code/plan"
)

const standardInstructions = `You are a coding agent. Your job is to implement a plan step precisely and completely.

Guidelines:
- Read existing code before modifying it.
- Use the available tools to read, write, edit, search, and run commands in the project directory.
- All file paths are relative to the project directory. Do not attempt to access paths outside it.
- Be concise. Do not explain what you are about to do -- just do it.

IMPORTANT: You MUST call the done tool when you have finished the task. Pass a brief summary of what you accomplished. If you do not call done, your work will be lost.`

// Build assembles the full system prompt for a coding agent turn.
func Build(systemPromptPrefix string, step plan.Step, feedback string) string {
	var b strings.Builder

	if systemPromptPrefix != "" {
		b.WriteString(systemPromptPrefix)
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
