package models

import "errors"

var (
	ErrNotFound          = errors.New("saga not found")
	ErrInvalidArgument   = errors.New("invalid argument")
	ErrInvalidTransition = errors.New("invalid saga step transition")
	ErrSagaCompleted     = errors.New("saga already completed")
)
