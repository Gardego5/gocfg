package utils

import "errors"

var (
	// ErrCircularDependency is returned when a circular dependency is detected
	// in the configuration.
	//
	// Consider this example:
	//
	//	type Config struct {
	//	    A string `cfg:"@B"`
	//	    B string `cfg:"@A"`
	//	}
	//
	// Because A depends on B and B depends on A, there is no way to resolve the
	// configuration.
	ErrCircularDependency = errors.New("circular dependency detected")

	// ErrUnboundVariable is returned when a field references an undefined field.
	ErrUnboundVariable = errors.New("reference to undefined field")

	// ErrMissingRequired is returned when a required field is not set.
	ErrMissingRequired = errors.New("required value not set")
)
