package models

import "errors"

var (
	ErrNotFound        = errors.New("not found")
	ErrInvalidArgument = errors.New("invalid argument")
	ErrConflict        = errors.New("conflict")
	ErrTokenExpired    = errors.New("upload token expired")
	ErrTokenUsed       = errors.New("upload token already used")
	ErrFileTooLarge    = errors.New("file too large")
	ErrUnsupportedType = errors.New("unsupported file type")
)
