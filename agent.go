package code

import "github.com/benaskins/axon-code/plan"

// Agent implements a plan step in a given project directory.
// The interface is structurally compatible with maestro's Agent interface.
type Agent interface {
	Implement(projectDir string, step plan.Step, feedback string) (string, error)
}
