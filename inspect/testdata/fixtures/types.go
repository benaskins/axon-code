// Package sample provides sample types and interfaces for testing.
package sample

// SampleStruct is a sample struct for testing.
type SampleStruct struct {
	// Name is the name field.
	Name string
	// Value is the value field.
	Value int
	// Private is a private field.
	private string
}

// SampleType is a type alias.
type SampleType string

// SamplePointer is a pointer type.
type SamplePointer *SampleStruct

// SampleSlice is a slice type.
type SampleSlice []int

// SampleMap is a map type.
type SampleMap map[string]int

// SampleChannel is a channel type.
type SampleChannel chan string

// SampleFunc is a function type.
type SampleFunc func(int) string

// UnexportedStruct is an unexported struct (should not appear).
type UnexportedStruct struct {
	Field string
}
