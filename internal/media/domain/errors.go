package domain

import "errors"

var (
	ErrNotFound          = errors.New("not found")
	ErrInvalidTransition = errors.New("invalid transition")
	ErrConflict          = errors.New("conflict") // под optimistic lock / version mismatch
)
