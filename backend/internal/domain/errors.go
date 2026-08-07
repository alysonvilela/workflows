package domain

import "errors"

// ErrNotFound is returned by store lookups when the requested row does not
// exist (or, for uuid columns, when the id could not even be parsed as one).
var ErrNotFound = errors.New("not found")

// ValidationError carries one message per invalid field, so a handler can
// surface it as the inline form errors the frontend renders.
type ValidationError struct {
	Fields map[string]string
}

func (e *ValidationError) Error() string {
	return "validation failed"
}
