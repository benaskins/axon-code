package prompt

import (
	"strings"

	"github.com/benaskins/axon-code/plan"
)

const coreGuidelines = `- Read existing code before modifying it.
- For Go files, use the ast tool instead of read_file. It returns the full structure (package, imports, declarations, signatures, bodies) in one call. Only fall back to read_file for non-Go files or when you need raw text.
- Use the available tools to read, write, search, and run commands in the project directory.
- All file paths are relative to the project directory. Do not attempt to access paths outside it.
- Be concise. Do not explain what you are about to do -- just do it.
- Do NOT create standalone main() programs in tmp/ to test your code. Write _test.go files instead. Tests are the verification mechanism, not scratch scripts. If you need to verify integration between packages, write an integration test in internal/integration/ or alongside the package.
- If tmp/ files exist from a previous step, ignore them. They are gitignored.
- Build and vet failures automatically include gopls diagnostics with exact line:column positions. Read the hooks section of the result for details.
- Use lsp_definition to look up the type signature and documentation of any symbol at a specific file position. This is how you discover dependency APIs.
- Do NOT refactor, rename, or reorganise code from previous steps. Work with the existing structure. If a package name collides with a stdlib package, use an import alias instead of renaming the package. Refactoring cascades across files and burns iterations.`

const doneInstruction = `IMPORTANT: You MUST call the done tool when you have finished the task. Pass a brief summary of what you accomplished. If you do not call done, your work will be lost.`

// qwenInstructions includes rewrite tool guidance since Qwen handles it well.
const qwenInstructions = `You are a coding agent. Your job is to implement a plan step precisely and completely.

Guidelines:
` + coreGuidelines + `
- To modify Go functions, prefer the rewrite tool (replace_body, rename) over edit_file. It operates on AST nodes so it cannot produce malformed code.

` + doneInstruction

// geminiInstructions omits edit_file and rewrite references since those tools
// are filtered out for Gemini. Uses task-first structure (PTCF pattern).
const geminiInstructions = `You are a coding agent. Your job is to implement the task below.

## Constraints
- Read existing Go code with the ast tool before writing changes.
- Use write_file to create new files or replace entire files.
- Use edit_file for targeted fixes to existing files (prefer this over rewriting the whole file).
- Use go_cmd and git_cmd for all Go toolchain and git operations.
- All file paths are relative to the project directory. Do not access paths outside it.
- Do NOT create standalone main() programs in tmp/. Write _test.go files instead.
- Be concise. Do not explain what you are about to do -- just do it.
- If tests fail after 3 attempts on the same file, simplify the implementation rather than continuing to tweak the same approach.
- Build and vet failures automatically include detailed gopls diagnostics in the result. Read the hooks section for exact error positions before attempting fixes.
- When you encounter an unknown type or function from a dependency, use lsp_definition at the symbol position to see its signature and documentation. Do not guess at APIs.`

const geminiClosing = `
## Completion
When you have finished ALL work for this task, you MUST call the done tool with a brief summary. Do not end your turn without calling done. If you do not call done, your work will be lost and the build will fail.`

// anthropicInstructions uses the standard structure that works well with Claude.
const anthropicInstructions = `You are a coding agent. Your job is to implement a plan step precisely and completely.

Guidelines:
` + coreGuidelines + `
- To modify Go functions, prefer the rewrite tool (replace_body, rename) over edit_file. It operates on AST nodes so it cannot produce malformed code.

` + doneInstruction

// Build assembles the full system prompt for a coding agent turn.
// The section ordering and content depend on the model family profile.
func Build(systemPromptPrefix string, step plan.Step, feedback string, profile Profile) string {
	switch profile.Family {
	case FamilyGemini:
		return buildGemini(systemPromptPrefix, step, feedback)
	case FamilyAnthropic:
		return buildAnthropic(systemPromptPrefix, step, feedback)
	default:
		return buildQwen(systemPromptPrefix, step, feedback)
	}
}

// buildQwen: expertise → instructions → task → feedback
func buildQwen(prefix string, step plan.Step, feedback string) string {
	var b strings.Builder
	if prefix != "" {
		b.WriteString(prefix)
		b.WriteString("\n\n")
	}
	b.WriteString(qwenInstructions)
	b.WriteString("\n\n")
	writeTask(&b, step)
	writeFeedback(&b, feedback)
	return b.String()
}

// buildGemini: task → constraints → expertise as reference → closing (done enforcement)
func buildGemini(prefix string, step plan.Step, feedback string) string {
	var b strings.Builder
	b.WriteString(geminiInstructions)
	b.WriteString("\n\n")
	writeTask(&b, step)
	if prefix != "" {
		b.WriteString("\n\n## Reference\n\n")
		b.WriteString(prefix)
	}
	writeFeedback(&b, feedback)
	b.WriteString(geminiClosing)
	return b.String()
}

// buildAnthropic: instructions → task → expertise as reference → feedback
func buildAnthropic(prefix string, step plan.Step, feedback string) string {
	var b strings.Builder
	b.WriteString(anthropicInstructions)
	b.WriteString("\n\n")
	writeTask(&b, step)
	if prefix != "" {
		b.WriteString("\n\n## Reference\n\n")
		b.WriteString(prefix)
	}
	writeFeedback(&b, feedback)
	return b.String()
}

func writeTask(b *strings.Builder, step plan.Step) {
	b.WriteString("## Task\n\n")
	b.WriteString("**")
	b.WriteString(step.Title)
	b.WriteString("**\n\n")
	b.WriteString(step.Description)
}

func writeFeedback(b *strings.Builder, feedback string) {
	if feedback != "" {
		b.WriteString("\n\n## Previous Attempt Failed\n\n")
		b.WriteString("The previous attempt did not succeed. Here is what went wrong:\n\n")
		b.WriteString(feedback)
	}
}
