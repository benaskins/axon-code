package code

import "github.com/benaskins/axon-code/plan"

// ImplementResult captures the outcome of implementing a plan step.
type ImplementResult struct {
	Summary string   // what the agent accomplished
	Files   []string // paths modified (relative to project dir)
}

// Agent implements a plan step in a given project directory.
type Agent interface {
	Implement(projectDir string, step plan.Step, feedback string) (*ImplementResult, error)
}
