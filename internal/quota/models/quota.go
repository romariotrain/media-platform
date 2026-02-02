package models

import (
	"time"

	"github.com/google/uuid"
)

// QuotaStatus представляет статус резервации квоты
type QuotaStatus string

const (
	QuotaReserved QuotaStatus = "reserved"
	QuotaReleased QuotaStatus = "released"
	QuotaExpired  QuotaStatus = "expired"
)

// QuotaType тип квоты
type QuotaType string

const (
	StorageQuota QuotaType = "storage" // квота на хранилище
	CountQuota   QuotaType = "count"   // квота на количество медиа
)

// Quota представляет резервацию квоты
type Quota struct {
	ID         uuid.UUID   `db:"id"`
	UserID     string      `db:"user_id"`     // идентификатор пользователя/аккаунта
	AssetID    uuid.UUID   `db:"asset_id"`    // ID медиа-ассета
	SagaID     uuid.UUID   `db:"saga_id"`     // ID саги для корреляции
	Type       QuotaType   `db:"type"`        // тип квоты
	Amount     int64       `db:"amount"`      // размер квоты (байты для storage, штуки для count)
	Status     QuotaStatus `db:"status"`      // статус резервации
	ReservedAt time.Time   `db:"reserved_at"` // время резервации
	ReleasedAt *time.Time  `db:"released_at"` // время освобождения (nullable)
	ExpiresAt  time.Time   `db:"expires_at"`  // время автоматического истечения
	CreatedAt  time.Time   `db:"created_at"`
	UpdatedAt  time.Time   `db:"updated_at"`
}

// UserQuota представляет текущее состояние квоты пользователя
type UserQuota struct {
	UserID      string    `db:"user_id"`
	Type        QuotaType `db:"type"`
	LimitAmount int64     `db:"limit_amount"` // максимальная квота
	UsedAmount  int64     `db:"used_amount"`  // использовано
	UpdatedAt   time.Time `db:"updated_at"`
}

// AvailableAmount возвращает доступную квоту
func (uq *UserQuota) AvailableAmount() int64 {
	available := uq.LimitAmount - uq.UsedAmount
	if available < 0 {
		return 0
	}
	return available
}

// CanReserve проверяет, можно ли зарезервировать указанное количество
func (uq *UserQuota) CanReserve(amount int64) bool {
	return uq.AvailableAmount() >= amount
}
