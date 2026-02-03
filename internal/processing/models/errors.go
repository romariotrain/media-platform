package models

import "errors"

var (
	ErrNotFound         = errors.New("not found")
	ErrInvalidArgument  = errors.New("invalid argument")
	ErrAlreadyProcessed = errors.New("already processed")
	ErrProcessingFailed = errors.New("processing failed")
	ErrFFmpegFailed     = errors.New("ffmpeg execution failed")
)
