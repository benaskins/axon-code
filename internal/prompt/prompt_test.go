package prompt_test

import (
	"strings"
	"testing"

	code "github.com/benaskins/axon-code"
	"github.com/benaskins/axon-code/internal/prompt"
	"github.com/benaskins/axon-code/plan"
)

func TestBuild_StepAlwaysPresent(t *testing.T) {
	cfg := code.Config{}
	step := plan.Step{Title: "My Title", Description: "Do the thing."}
	out := prompt.Build(cfg, step, "")

	if !strings.Contains(out, "My Title") {
		t.Errorf("expected step title in output, got:\n%s", out)
	}
	if !strings.Contains(out, "Do the thing.") {
		t.Errorf("expected step description in output, got:\n%s", out)
	}
}

func TestBuild_PrefixPrependedWhenSet(t *testing.T) {
	cfg := code.Config{SystemPromptPrefix: "CUSTOM PREFIX"}
	step := plan.Step{Title: "T", Description: "D"}
	out := prompt.Build(cfg, step, "")

	if !strings.HasPrefix(out, "CUSTOM PREFIX") {
		t.Errorf("expected output to start with prefix, got:\n%s", out)
	}
}

func TestBuild_NoPrefixWhenEmpty(t *testing.T) {
	cfg := code.Config{}
	step := plan.Step{Title: "T", Description: "D"}
	out := prompt.Build(cfg, step, "")

	if strings.HasPrefix(out, "\n") {
		t.Errorf("expected no leading newline when prefix is empty, got:\n%s", out)
	}
}

func TestBuild_FeedbackAppearsOnlyWhenNonEmpty(t *testing.T) {
	cfg := code.Config{}
	step := plan.Step{Title: "T", Description: "D"}

	withFeedback := prompt.Build(cfg, step, "something went wrong")
	withoutFeedback := prompt.Build(cfg, step, "")

	if !strings.Contains(withFeedback, "something went wrong") {
		t.Errorf("expected feedback in output, got:\n%s", withFeedback)
	}
	if strings.Contains(withoutFeedback, "something went wrong") {
		t.Errorf("expected no feedback in output, got:\n%s", withoutFeedback)
	}
}

func TestBuild_Deterministic(t *testing.T) {
	cfg := code.Config{SystemPromptPrefix: "PREFIX"}
	step := plan.Step{Title: "T", Description: "D"}
	feedback := "prior error"

	a := prompt.Build(cfg, step, feedback)
	b := prompt.Build(cfg, step, feedback)

	if a != b {
		t.Errorf("Build is not deterministic:\na=%s\nb=%s", a, b)
	}
}
