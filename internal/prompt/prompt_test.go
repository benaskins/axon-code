package prompt_test

import (
	"strings"
	"testing"

	"github.com/benaskins/axon-code/internal/prompt"
	"github.com/benaskins/axon-code/plan"
)

var testStep = plan.Step{Title: "My Title", Description: "Do the thing."}

// --- Profile selection ---

func TestForModel_Gemini(t *testing.T) {
	p := prompt.ForModel("google/gemini-2.5-flash")
	if p.Family != prompt.FamilyGemini {
		t.Errorf("expected FamilyGemini, got %v", p.Family)
	}
	if p.Temperature == nil || *p.Temperature != 1.0 {
		t.Errorf("expected temperature 1.0, got %v", p.Temperature)
	}
	if len(p.ExcludeTools) == 0 {
		t.Error("expected ExcludeTools to be non-empty for Gemini")
	}
}

func TestForModel_Qwen(t *testing.T) {
	p := prompt.ForModel("qwen/qwen3.5-122b-a10b")
	if p.Family != prompt.FamilyQwen {
		t.Errorf("expected FamilyQwen, got %v", p.Family)
	}
	if p.Temperature != nil {
		t.Errorf("expected nil temperature for Qwen, got %v", *p.Temperature)
	}
}

func TestForModel_Anthropic(t *testing.T) {
	p := prompt.ForModel("anthropic/claude-sonnet-4-6")
	if p.Family != prompt.FamilyAnthropic {
		t.Errorf("expected FamilyAnthropic, got %v", p.Family)
	}
}

func TestForModel_DefaultIsQwen(t *testing.T) {
	p := prompt.ForModel("deepseek/deepseek-v3")
	if p.Family != prompt.FamilyQwen {
		t.Errorf("expected FamilyQwen as default, got %v", p.Family)
	}
}

// --- Prompt composition: all families ---

func TestBuild_StepAlwaysPresent(t *testing.T) {
	for _, family := range []string{"qwen/test", "google/gemini-2.5-flash", "anthropic/claude"} {
		t.Run(family, func(t *testing.T) {
			p := prompt.ForModel(family)
			out := prompt.Build("", testStep, "", p)
			if !strings.Contains(out, "My Title") {
				t.Errorf("missing step title")
			}
			if !strings.Contains(out, "Do the thing.") {
				t.Errorf("missing step description")
			}
		})
	}
}

func TestBuild_FeedbackAppearsOnlyWhenNonEmpty(t *testing.T) {
	for _, family := range []string{"qwen/test", "google/gemini-2.5-flash", "anthropic/claude"} {
		t.Run(family, func(t *testing.T) {
			p := prompt.ForModel(family)
			with := prompt.Build("", testStep, "something went wrong", p)
			without := prompt.Build("", testStep, "", p)
			if !strings.Contains(with, "something went wrong") {
				t.Errorf("expected feedback in output")
			}
			if strings.Contains(without, "something went wrong") {
				t.Errorf("unexpected feedback in output")
			}
		})
	}
}

func TestBuild_Deterministic(t *testing.T) {
	p := prompt.ForModel("qwen/test")
	a := prompt.Build("PREFIX", testStep, "err", p)
	b := prompt.Build("PREFIX", testStep, "err", p)
	if a != b {
		t.Error("Build is not deterministic")
	}
}

// --- Qwen-specific ---

func TestBuild_Qwen_PrefixFirst(t *testing.T) {
	p := prompt.ForModel("qwen/test")
	out := prompt.Build("EXPERTISE BLOCK", testStep, "", p)
	if !strings.HasPrefix(out, "EXPERTISE BLOCK") {
		t.Errorf("Qwen: expected prefix at start, got:\n%.100s", out)
	}
}

func TestBuild_Qwen_HasRewriteGuidance(t *testing.T) {
	p := prompt.ForModel("qwen/test")
	out := prompt.Build("", testStep, "", p)
	if !strings.Contains(out, "rewrite") {
		t.Error("Qwen prompt should mention rewrite tool")
	}
}

// --- Gemini-specific ---

func TestBuild_Gemini_TaskBeforeExpertise(t *testing.T) {
	p := prompt.ForModel("google/gemini-2.5-flash")
	out := prompt.Build("EXPERTISE BLOCK", testStep, "", p)
	taskIdx := strings.Index(out, "## Task")
	refIdx := strings.Index(out, "## Reference")
	if taskIdx < 0 || refIdx < 0 {
		t.Fatalf("missing Task or Reference section")
	}
	if taskIdx > refIdx {
		t.Error("Gemini: Task should come before Reference")
	}
}

func TestBuild_Gemini_NoRewriteGuidance(t *testing.T) {
	p := prompt.ForModel("google/gemini-2.5-flash")
	out := prompt.Build("", testStep, "", p)
	if strings.Contains(out, "rewrite tool") {
		t.Error("Gemini prompt should not mention rewrite tool")
	}
}

func TestBuild_Gemini_DoneEnforcedAtEnd(t *testing.T) {
	p := prompt.ForModel("google/gemini-2.5-flash")
	out := prompt.Build("", testStep, "", p)
	if !strings.Contains(out, "MUST call the done tool") {
		t.Error("Gemini prompt should enforce done tool")
	}
	// done instruction should be near the end
	doneIdx := strings.LastIndex(out, "MUST call the done tool")
	if doneIdx < len(out)/2 {
		t.Error("Gemini: done enforcement should be in second half of prompt")
	}
}

func TestForModel_Gemini_ToolChoiceAuto(t *testing.T) {
	p := prompt.ForModel("google/gemini-2.5-flash")
	if p.ToolChoice != "auto" {
		t.Errorf("expected ToolChoice auto, got %q", p.ToolChoice)
	}
}

func TestBuild_Gemini_ExcludesRewriteOnly(t *testing.T) {
	p := prompt.ForModel("google/gemini-2.5-flash")
	found := map[string]bool{}
	for _, name := range p.ExcludeTools {
		found[name] = true
	}
	if found["edit_file"] {
		t.Error("Gemini should NOT exclude edit_file")
	}
	if !found["rewrite"] {
		t.Error("Gemini should exclude rewrite")
	}
}

func TestBuild_Gemini_WriteFileGuidance(t *testing.T) {
	p := prompt.ForModel("google/gemini-2.5-flash")
	out := prompt.Build("", testStep, "", p)
	if !strings.Contains(out, "write_file") {
		t.Error("Gemini prompt should guide toward write_file")
	}
}

func TestBuild_Gemini_LSPGuidance(t *testing.T) {
	p := prompt.ForModel("google/gemini-2.5-flash")
	out := prompt.Build("", testStep, "", p)
	if !strings.Contains(out, "lsp_diagnostics") {
		t.Error("Gemini prompt should mention lsp_diagnostics")
	}
	if !strings.Contains(out, "lsp_definition") {
		t.Error("Gemini prompt should mention lsp_definition")
	}
}

func TestBuild_AllFamilies_LSPGuidance(t *testing.T) {
	for _, family := range []string{"qwen/test", "google/gemini-2.5-flash", "anthropic/claude"} {
		t.Run(family, func(t *testing.T) {
			p := prompt.ForModel(family)
			out := prompt.Build("", testStep, "", p)
			if !strings.Contains(out, "lsp_diagnostics") {
				t.Errorf("prompt should mention lsp_diagnostics")
			}
		})
	}
}

// --- Anthropic-specific ---

func TestBuild_Anthropic_InstructionsFirst(t *testing.T) {
	p := prompt.ForModel("anthropic/claude")
	out := prompt.Build("EXPERTISE BLOCK", testStep, "", p)
	instrIdx := strings.Index(out, "You are a coding agent")
	taskIdx := strings.Index(out, "## Task")
	if instrIdx < 0 || taskIdx < 0 {
		t.Fatalf("missing instructions or task")
	}
	if instrIdx > taskIdx {
		t.Error("Anthropic: instructions should come before task")
	}
}
