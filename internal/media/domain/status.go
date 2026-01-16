package domain

import "fmt"

type Status string

const (
	Uploaded   Status = "uploaded"
	Processing Status = "processing"
	Ready      Status = "ready"
	Failed     Status = "failed"
)

func CanTransition(from, to Status) bool {
	switch from {
	case Uploaded:
		return to == Processing || to == Failed
	case Processing:
		return to == Ready || to == Failed
	case Ready:
		return false
	case Failed:
		return false
	default:
		return false
	}
}

func ValidateTransition(from, to Status) error {
	if from == to {
		return nil
	}
	if !CanTransition(from, to) {
		return fmt.Errorf("invalid transition: %s -> %s", from, to)
	}
	return nil
}
