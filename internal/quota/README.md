# Quota Service - Реализация завершена! ✅

## Что сделано

### 1. Domain Layer
- ✅ `models/quota.go` - модели Quota, UserQuota с бизнес-логикой
- ✅ `models/errors.go` - domain ошибки
- ✅ `models/events.go` - события (QuotaReserved, QuotaReleased, QuotaFailed)
- ✅ `models/commands.go` - команды (ReserveQuota, ReleaseQuota)

### 2. Repository Layer
- ✅ `repository/repository.go` - интерфейс QuotaRepository
- ✅ `repository/memory.go` - in-memory реализация для тестов
- ✅ `storage/postgres/quota_repo.go` - Postgres реализация
- ✅ `storage/postgres/outbox_repo.go` - Outbox Pattern для надёжной публикации
- ✅ `storage/postgres/db.go` - подключение к БД

### 3. Service Layer
- ✅ `service/service.go` - структура сервиса с outbox
- ✅ `service/reserve.go` - резервирование квоты с идемпотентностью
- ✅ `service/release.go` - освобождение квоты с идемпотентностью
- ✅ `service/service_test.go` - unit тесты

### 4. Kafka Integration
- ✅ `outbox/publisher.go` - фоновый процесс публикации событий из outbox
- ✅ `kafka/consumer.go` - обработчик команд из Kafka

### 5. Entry Point
- ✅ `cmd/quota/main.go` - точка входа
- ✅ `cmd/quota/run.go` - инициализация и запуск

### 6. Database
- ✅ `sql/quota_schema.sql` - миграции для таблиц quotas, user_quotas, quota_outbox

## Архитектурные решения

### Идемпотентность
Проверка по `asset_id`: один asset = одна резервация квоты.
Повторные команды с тем же `asset_id` возвращают существующую квоту.

### Транзакции (SELECT FOR UPDATE)
```sql
BEGIN;
  SELECT * FROM user_quotas WHERE user_id = ? FOR UPDATE; -- блокируем
  UPDATE user_quotas SET used_amount = used_amount + ?;
  INSERT INTO quotas (...);
COMMIT;
```
Защита от race conditions при параллельных резервациях.

### Outbox Pattern
```
BEGIN TRANSACTION;
  UPDATE quotas SET status = 'reserved';
  INSERT INTO quota_outbox (event_type, payload);
COMMIT;

Отдельный publisher:
  - Читает quota_outbox
  - Публикует в Kafka
  - Помечает как processed
```

Гарантия: событие будет опубликовано если транзакция успешна.

### Обработка ошибок
- `ErrInsufficientQuota` → Consumer возвращает success (команда обработана)
  - TODO: публиковать событие `QuotaFailed` напрямую в Kafka
- `ErrNotFound` при release → не критично (возможно резервации не было)

## Запуск

### 1. Применить миграции
```bash
psql -h localhost -U postgres -d your_db -f sql/quota_schema.sql
```

### 2. Настроить окружение
```bash
# .env файл
DATABASE_URL=postgres://user:pass@localhost:5432/your_db
```

### 3. Создать тестовые квоты пользователям
```sql
INSERT INTO user_quotas (user_id, type, limit_amount, used_amount, updated_at)
VALUES 
    ('user123', 'storage', 10737418240, 0, NOW()), -- 10GB
    ('user123', 'count', 1000, 0, NOW());
```

### 4. Запустить сервис
```bash
go run ./cmd/quota
```

## Kafka Topics

**Входящие команды** (слушает):
- `commands.quota.reserve` - резервирование квоты
- `commands.quota.release` - освобождение квоты

**Исходящие события** (публикует):
- `events.quota.reserved` - квота успешно зарезервирована
- `events.quota.released` - квота освобождена
- `events.quota.failed` - не хватило квоты (TODO)

## Формат сообщений

### Команда ReserveQuota
```json
{
  "message_id": "msg-uuid-123",
  "saga_id": "saga-uuid-456",
  "type": "ReserveQuota",
  "step": "RESERVE_QUOTA",
  "created_at": "2026-01-28T12:00:00Z",
  "payload": {
    "user_id": "user123",
    "asset_id": "asset-uuid-789",
    "type": "storage",
    "amount": 2097152
  }
}
```

### Событие QuotaReserved
```json
{
  "message_id": "outbox-42",
  "saga_id": "saga-uuid-456",
  "type": "QuotaReserved",
  "created_at": "2026-01-28T12:00:01Z",
  "payload": {
    "quota_id": "quota-uuid-111",
    "asset_id": "asset-uuid-789",
    "user_id": "user123",
    "type": "storage",
    "amount": 2097152,
    "expires_at": "2026-01-28T13:00:01Z"
  }
}
```

## TODO / Improvements

### Phase 2
- [ ] Идемпотентность через Redis (проверка `message_id`)
- [ ] Публикация `QuotaFailed` напрямую в Kafka (без outbox)
- [ ] Background job для очистки expired квот
- [ ] Метрики (Prometheus)
- [ ] Distributed tracing (OpenTelemetry)

### Phase 3
- [ ] HTTP API для проверки квот (GET /quotas/{user_id})
- [ ] Горизонтальное масштабирование (Kafka consumer groups)
- [ ] Graceful shutdown для корректной обработки in-flight сообщений

## Тестирование

```bash
# Unit тесты
go test ./internal/quota/service/...

# Integration тесты (требуется running Postgres + Kafka)
go test ./internal/quota/... -tags=integration
```

## Примеры использования

### Отправка тестовой команды в Kafka
```bash
echo '{
  "message_id": "test-1",
  "saga_id": "00000000-0000-0000-0000-000000000001",
  "type": "ReserveQuota",
  "step": "RESERVE_QUOTA",
  "created_at": "2026-01-28T12:00:00Z",
  "payload": {
    "user_id": "user123",
    "asset_id": "00000000-0000-0000-0000-000000000002",
    "type": "storage",
    "amount": 1048576
  }
}' | docker exec -i mp_kafka kafka-console-producer \
    --bootstrap-server localhost:9092 \
    --topic commands.quota.reserve
```

### Проверка событий в outbox
```sql
SELECT * FROM quota_outbox WHERE processed_at IS NULL;
```

### Проверка квот пользователя
```sql
SELECT * FROM user_quotas WHERE user_id = 'user123';
SELECT * FROM quotas WHERE user_id = 'user123';
```

---

**Статус**: ✅ Quota Service полностью реализован и готов к интеграции с Orchestrator!
