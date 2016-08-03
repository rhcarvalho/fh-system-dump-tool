package main

import "strings"

// An errorList accumulates multiple errors and implements error.
type errorList []error

func (e errorList) Error() string {
	switch len(e) {
	case 0:
		return ""
	case 1:
		return e[0].Error()
	}
	var msgs = make([]string, len(e))
	for i := range e {
		msgs[i] = e[i].Error()
	}
	return "multiple errors:\n" + strings.Join(msgs, "\n")
}
