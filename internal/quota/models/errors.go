package models

import "errors"

var (
	// ErrNotFound возвращается когда квота не найдена
	ErrNotFound = errors.New("quota not found")

	// ErrInvalidArgument возвращается при невалидных параметрах
	ErrInvalidArgument = errors.New("invalid argument")

	// ErrInsufficientQuota возвращается когда недостаточно квоты
	ErrInsufficientQuota = errors.New("insufficient quota")

	// ErrAlreadyReserved возвращается при попытке повторно зарезервировать квоту
	ErrAlreadyReserved = errors.New("quota already reserved")

	// ErrAlreadyReleased возвращается при попытке освободить уже освобожденную квоту
	ErrAlreadyReleased = errors.New("quota already released")

	// ErrExpired возвращается при работе с истекшей квотой
	ErrExpired = errors.New("quota expired")
)
