package prompt

import "strings"

// Family identifies an LLM model family for prompt composition.
type Family int

const (
	FamilyQwen Family = iota
	FamilyGemini
	FamilyAnthropic
)

// Profile controls how the system prompt is assembled and what
// tools/settings are appropriate for a given model family.
type Profile struct {
	Family       Family
	Temperature  *float64 // nil = use caller's setting, non-nil = override
	ExcludeTools []string // tool names to filter out
	ToolChoice   string   // "auto", "required", or "" to omit
	Stream       *bool    // nil = use caller's setting, non-nil = override
}

// ForModel returns the appropriate profile for the given model string.
func ForModel(model string) Profile {
	m := strings.ToLower(model)
	switch {
	case strings.Contains(m, "gemini"):
		temp := 1.0
		return Profile{
			Family:      FamilyGemini,
			Temperature: &temp,
			// Gemini thrashes when rewrite (AST mutation) competes with
			// write_file and edit_file. Keep edit_file for targeted fixes.
			ExcludeTools: []string{"rewrite"},
			// Gemini won't call tools unless explicitly told to.
			ToolChoice: "auto",
		}
	case strings.Contains(m, "claude") || strings.Contains(m, "anthropic"):
		return Profile{Family: FamilyAnthropic}
	default:
		return Profile{Family: FamilyQwen}
	}
}
