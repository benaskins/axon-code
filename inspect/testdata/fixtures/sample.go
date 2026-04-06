// Package sample provides sample types and interfaces for testing.
package sample

// SampleInterface is a sample interface for testing.
type SampleInterface interface {
	// DoSomething performs an action.
	DoSomething() error
	// GetValue returns a value.
	GetValue() int
}

// AnotherInterface is another sample interface.
type AnotherInterface interface {
	// Process processes data.
	Process(data string) string
	// Reset resets the state.
	Reset()
}

// EmbeddedInterface embeds another interface.
type EmbeddedInterface interface {
	SampleInterface
	// ExtendedMethod adds extended functionality.
	ExtendedMethod() bool
}

// sampleUnexported is an unexported interface (should not appear).
type sampleUnexported interface {
	UnexportedMethod()
}
