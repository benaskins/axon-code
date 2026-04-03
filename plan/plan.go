package plan

// Step is a single unit of work in an implementation plan.
// Fields match the shape used by maestro for structural compatibility.
type Step struct {
	Title       string
	Description string
}
