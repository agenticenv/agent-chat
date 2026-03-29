package store

import "errors"

// ErrNotFound is returned when a row targeted by ID does not exist.
var ErrNotFound = errors.New("not found")