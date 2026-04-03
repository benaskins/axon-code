package tools

import (
	"testing"

	tool "github.com/benaskins/axon-tool"
)

func TestDoneTool_SetsSummaryOnSignal(t *testing.T) {
	signal := &DoneSignal{}
	doneTool := NewDoneTool(signal)

	ctx := &tool.ToolContext{}
	result := doneTool.Execute(ctx, map[string]any{
		"summary": "all tasks complete",
	})

	if !signal.Done {
		t.Error("expected DoneSignal.Done to be true after Execute")
	}
	if signal.Summary != "all tasks complete" {
		t.Errorf("expected summary %q, got %q", "all tasks complete", signal.Summary)
	}
	if result.Content == "" {
		t.Error("expected non-empty result content")
	}
}

func TestDoneTool_MissingSummary(t *testing.T) {
	signal := &DoneSignal{}
	doneTool := NewDoneTool(signal)

	ctx := &tool.ToolContext{}
	result := doneTool.Execute(ctx, map[string]any{})

	if signal.Done {
		t.Error("expected DoneSignal.Done to remain false when summary arg is missing")
	}
	if result.Content == "" {
		t.Error("expected non-empty error result content")
	}
}
