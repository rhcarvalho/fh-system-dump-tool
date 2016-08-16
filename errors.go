package main

// IgnorableError is an error that has an extra method to tell whether it should
// be ignored. Ignored errors may be omitted from standard program output.
type IgnorableError interface {
	error
	Ignore() bool // Is the error a timeout?
}

// ignoredError implements IgnorableError, always returning true.
type ignoredError struct {
	err error
}

func (e *ignoredError) Error() string {
	if e.err == nil {
		return ""
	}
	return e.err.Error()
}

// Ignore implements IgnorableError.
func (e *ignoredError) Ignore() bool {
	return true
}

// Assert that ignoredError implements the IgnorableError interface.
var _ IgnorableError = (*ignoredError)(nil)

// MarkErrorAsIgnorable marks the original error as ignored if non-nil.
func MarkErrorAsIgnorable(err error) error {
	if err == nil {
		// Keeps nil errors nil, to avoid the confusion of having a
		// non-nil type and a nil value in an interface value.
		return nil
	}
	return &ignoredError{err}
}
