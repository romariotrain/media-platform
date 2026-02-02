# Media Platform - Event-Driven Microservices

## О проекте

Это учебная микросервисная платформа для обработки и публикации медиа-файлов (видео, аудио, изображения). Проект демонстрирует:

- **Event-Driven Architecture** через Apache Kafka
- **Saga Pattern (Orchestration)** для координации долгих процессов
- **Outbox Pattern** для надёжной публикации событий
- **Идемпотентность** обработки команд и событий
- **Distributed Transactions** через компенсации

## Бизнес-процесс

```
Пользователь загружает файл
    ↓
Система обрабатывает файл (транскодирование, превью)
    ↓
Система публикует готовый контент
```

Весь процесс разбит на микросервисы, которые общаются через события.

---

## Архитектура

### Orchestrator Service (координатор)
**Роль**: Мозг системы. Управляет всем процессом публикации медиа.

**Что делает**:
- Хранит состояние саги (где мы сейчас в процессе)
- Отправляет команды другим сервисам
- Реагирует на события и решает, что делать дальше
- Обрабатывает ошибки и запускает компенсации (откат)
- Следит за таймаутами

**Пример саги** (последовательность шагов):
1. Создать запись о медиа → `commands.asset.create`
2. Зарезервировать квоту → `commands.quota.reserve`
3. Подготовить загрузку → `commands.ingest.prepare`
4. Дождаться файла (timeout 30 мин)
5. Обработать файл → `commands.processing.start`
6. Опубликовать → `commands.publish.finalize`
7. Пометить как готово → `commands.asset.mark_published`

**Компенсации** (если что-то пошло не так):
- Освободить квоту
- Удалить временные файлы
- Пометить медиа как failed

---

### Media Service (источник истины)
**Роль**: Реестр всех медиа-файлов. База данных знаний о каждом файле.

**Что делает**:
- Хранит метаданные: ID, статус, владелец, тип (video/audio/image)
- Меняет статусы: `uploaded → processing → ready → published`
- Отвечает на запросы "где этот файл?" и "в каком он статусе?"
- Публикует события о изменениях (`MediaCreated`, `MediaStatusChanged`)

**Важно**: НЕ обрабатывает файлы! Только хранит информацию о них.

**База данных**:
```
medias table:
- id (UUID)
- status (uploaded/processing/ready/failed)
- type (video/audio/file)
- source (ссылка на файл в storage)
- owner_id
- created_at, updated_at
```

---

### Quota Service (контроль лимитов)
**Роль**: Бухгалтер. Следит чтобы пользователи не превысили лимиты.

**Что делает**:
- Хранит лимиты пользователей (напр. 100GB хранилища, 1000 файлов)
- Резервирует квоту перед загрузкой ("забронировать 2GB")
- Освобождает квоту при отмене или ошибке
- Отвечает "можно или нельзя загружать"

**Сценарии**:
- ✅ Пользователь загружает 1GB файл, у него 5GB свободно → OK
- ❌ Пользователь загружает 10GB файл, у него 3GB свободно → REJECT
- 🔄 Загрузка отменена → вернуть зарезервированную квоту

**База данных**:
```
user_quotas table:
- user_id
- type (storage/count)
- limit_amount (100GB)
- used_amount (45GB)

quotas table (история резерваций):
- id
- user_id
- asset_id
- amount (2GB)
- status (reserved/released)
- expires_at (автоосвобождение через 1 час)
```

---

### Ingest Service (приёмка файлов)
**Роль**: Швейцар. Принимает файлы от пользователей.

**Что делает**:
- Генерирует upload token (временная ссылка для загрузки)
- Принимает файлы по HTTP
- Валидирует формат, размер, checksum
- Сохраняет сырой файл в Object Storage (S3/MinIO)
- Публикует событие `IngestUploaded` (файл получен)

**Важно**:
- Идемпотентность: повторная загрузка того же файла не создаст дубликат
- Может принимать файлы частями (chunked upload)

**Процесс**:
```
1. GET /upload-token → {token: "abc123", expires: "2026-01-27T12:00:00Z"}
2. POST /upload?token=abc123 + file → {upload_id: "u1", status: "uploaded"}
3. Event: IngestUploaded {upload_id: "u1", storage_path: "s3://raw/u1.mp4"}
```

---

### Processing Service (обработчик)
**Роль**: Кухня. Готовит файлы к потреблению.

**Что делает**:
- Транскодирует видео (напр. 4K → 1080p, 720p, 480p)
- Генерирует превью/thumbnails
- Извлекает метаданные (длительность, разрешение, codec)
- Нормализует аудио
- Оптимизирует изображения

**Процесс**:
```
1. Слушает: ProcessingStartCommand
2. Скачивает raw файл из storage
3. Обрабатывает (может занять минуты/часы)
4. Загружает результаты обратно в storage
5. Публикует: ProcessingSucceeded или ProcessingFailed
```

**Реализация** (для MVP может быть заглушкой):
- Реальная: FFmpeg, ImageMagick
- MVP: просто sleep(5s) + success event

---

### Publish Service (публикация)
**Роль**: Издатель. Делает файлы доступными миру.

**Что делает**:
- Проверяет что всё готово (все форматы обработаны)
- Генерирует публичные ссылки (CDN URLs)
- Настраивает правила доступа (public/private)
- Обновляет метаданные для быстрого доступа
- Публикует событие `MediaPublished`

**После публикации**:
```json
{
  "media_id": "m1",
  "public_urls": {
    "1080p": "https://cdn.example.com/m1_1080p.mp4",
    "720p": "https://cdn.example.com/m1_720p.mp4",
    "thumbnail": "https://cdn.example.com/m1_thumb.jpg"
  },
  "status": "published",
  "published_at": "2026-01-27T12:00:00Z"
}
```

---

## Коммуникация между сервисами

### Kafka Topics

**Команды** (команды от orchestrator к сервисам):
```
commands.asset.create           → Media Service
commands.asset.mark_published   → Media Service
commands.quota.reserve          → Quota Service
commands.quota.release          → Quota Service
commands.ingest.prepare         → Ingest Service
commands.processing.start       → Processing Service
commands.publish.finalize       → Publish Service
```

**События** (уведомления от сервисов к orchestrator):
```
events.asset.created            ← Media Service
events.asset.status_changed     ← Media Service
events.quota.reserved           ← Quota Service
events.quota.failed             ← Quota Service
events.ingest.ready             ← Ingest Service
events.ingest.uploaded          ← Ingest Service
events.processing.succeeded     ← Processing Service
events.processing.failed        ← Processing Service
events.publish.succeeded        ← Publish Service
```

### Message Format (конверт)

Все сообщения идут в едином формате:
```json
{
  "message_id": "msg-uuid-123",      // для дедупликации
  "saga_id": "saga-uuid-456",        // корреляция шагов саги
  "type": "QuotaReserve",            // тип команды/события
  "step": "RESERVE_QUOTA",           // шаг саги
  "created_at": "2026-01-27T12:00:00Z",
  "payload": {                       // данные специфичные для команды
    "user_id": "user123",
    "amount": 2048576,               // 2MB
    "asset_id": "asset-uuid-789"
  }
}
```

---

## Ключевые паттерны

### 1. Outbox Pattern
Проблема: как гарантировать что событие будет опубликовано после изменения БД?

Решение:
```sql
BEGIN TRANSACTION;
  UPDATE medias SET status = 'ready' WHERE id = 'a1';
  INSERT INTO outbox (event_type, payload) VALUES ('MediaReady', '...');
COMMIT;

-- Отдельный publisher читает outbox и шлёт в Kafka
```

### 2. Idempotency (идемпотентность)
Проблема: Kafka может доставить сообщение дважды.

Решение:
```go
// Проверяем message_id перед обработкой
if redis.Exists(ctx, msg.MessageID) {
    return nil // уже обработано
}

// Обрабатываем
process(msg)

// Помечаем как обработанное (TTL 24h)
redis.Set(ctx, msg.MessageID, "processed", 24*time.Hour)
```

### 3. Saga Compensation
Проблема: шаг 5 из 7 упал, нужно откатить шаги 1-4.

Решение:
```
Forward actions:        Compensations:
1. CreateAsset       →  7. MarkAssetFailed
2. ReserveQuota      →  6. ReleaseQuota
3. PrepareIngest     →  5. CleanupIngest
4. ProcessMedia      →  4. CleanupProcessing
[ERROR HERE]         →  3. CleanupPublish
                        2. ReleaseQuota
                        1. MarkAssetFailed
```

---

## Инфраструктура (Docker Compose)

```yaml
services:
  kafka:       # Брокер сообщений
  postgres:    # База данных (media, quotas, saga state)
  redis:       # Кэш (идемпотентность, deduplication)
  kafka-ui:    # Web UI для просмотра топиков
  minio:       # Object Storage (S3-compatible)
```

---

## Начало работы

```bash
# 1. Поднять инфру
docker compose -f deploy/docker-compose.yml up -d

# 2. Создать БД схемы
psql -h localhost -U postgres -f sql/schema.sql

# 3. Запустить сервисы (в отдельных терминалах)
go run ./cmd/orchestrator
go run ./cmd/media
go run ./cmd/quota
go run ./cmd/ingest
go run ./cmd/processing
go run ./cmd/publish

# 4. Kafka UI
open http://localhost:8080
```

---

## Roadmap

### MVP (минимум для работы):
- ✅ Media Service (create, get, change status)
- 🔄 Quota Service (reserve, release)
- ⏳ Ingest Service (upload token, accept file)
- ⏳ Processing Service (mock processing)
- ⏳ Publish Service (generate URLs)
- ⏳ Orchestrator (базовая сага без компенсаций)

### Phase 2:
- Компенсации в orchestrator
- Retry политики
- Dead Letter Queue
- Мониторинг и метрики

### Phase 3:
- Реальная обработка (FFmpeg)
- CDN интеграция
- Горизонтальное масштабирование
- Distributed tracing

---

## Вопросы для изучения

1. **Почему нельзя использовать REST между сервисами?**
    - Что если сервис недоступен?
    - Как гарантировать порядок операций?

2. **Зачем нужен Orchestrator?**
    - Можно ли сделать choreography (каждый сервис знает что делать дальше)?
    - Какие плюсы/минусы?

3. **Как тестировать распределённые саги?**
    - Unit тесты для каждого сервиса
    - Integration тесты для саги
    - Chaos engineering (имитация падений)

4. **Что делать если Kafka недоступна?**
    - Outbox pattern спасает запись в БД
    - Но отправка события застрянет
    - Нужен retry механизм в publisher
