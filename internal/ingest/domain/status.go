package domain

import "fmt"

// UploadStatus представляет статус загрузки в доменном слое
type UploadStatus string

const (
	Pending  UploadStatus = "pending"
	Ready    UploadStatus = "ready"
	Uploaded UploadStatus = "uploaded"
	Expired  UploadStatus = "expired"
	Failed   UploadStatus = "failed"
)

// Машина состояний:
// pending  -> ready     (подготовка завершена)
// pending  -> failed    (ошибка подготовки)
// ready    -> uploaded  (файл получен)
// ready    -> expired   (токен протух)
// ready    -> failed    (ошибка загрузки)

func CanTransition(from, to UploadStatus) bool {
	switch from {
	case Pending:
		return to == Ready || to == Failed
	case Ready:
		return to == Uploaded || to == Expired || to == Failed
	case Uploaded, Expired, Failed:
		return false
	default:
		return false
	}
}

func ValidateTransition(from, to UploadStatus) error {
	if from == to {
		return nil
	}
	if !CanTransition(from, to) {
		return fmt.Errorf("invalid upload status transition: %s -> %s", from, to)
	}
	return nil
}
