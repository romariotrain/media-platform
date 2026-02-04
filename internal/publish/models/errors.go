package models

import "errors"

var (
	ErrNotFound        = errors.New("publication not found")
	ErrInvalidArgument = errors.New("invalid argument")
	ErrAlreadyExists   = errors.New("publication already exists")
)
