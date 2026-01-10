# Media Platform (Go + Kafka) — Saga Orchestrator Training Project

Учебный микросервисный проект на Go для практики:
- event-driven архитектуры на Kafka
- Saga (orchestration) для долгих процессов
- идемпотентности/дедупликации (Redis)
- хранения состояния и аудита (Postgres)

Домен выбран технический: “публикация медиа-ассета” как цепочка асинхронных шагов. Реальная обработка медиа упрощена; фокус — на корректной межсервисной координации, ретраях и компенсациях.

---

## Цель MVP

Реализовать первый сквозной цикл (часть саги):
1) Orchestrator публикует `commands.asset.create`
2) Media Service обрабатывает команду и публикует `events.asset.created`
3) Orchestrator реагирует на событие и продолжает сагу (следующий шаг будет `commands.quota.reserve`)

---

## Сервисы

- **orchestrator**
    - хранит состояние саги (Postgres)
    - отправляет команды в Kafka
    - слушает события и двигает workflow
    - отвечает за ретраи/таймауты/компенсации

- **media**
    - реестр ассетов и их статусов (Postgres)
    - обрабатывает команды Create/Fail/MarkPublished и публикует события

- **quota**
    - резервирует/освобождает квоту (MVP: упрощённо; Redis/PG — решим по мере роста)

- **ingest**
    - подготовка к загрузке (token)
    - приём загрузки по HTTP (позже)
    - публикует `events.ingest.uploaded`

- **processing**
    - “обработка” (MVP: имитация)
    - публикует `events.processing.succeeded/failed`

- **publish**
    - финализация публикации
    - публикует `events.publish.succeeded/failed`

---

## Saga: PublishMediaAsset (план)

Happy path:
1) CreateAsset
2) ReserveQuota
3) PrepareIngest
4) WaitFileUploaded (timeout)
5) ProcessMedia (retries)
6) Publish (retries)
7) MarkPublished

Компенсации (в обратном порядке, идемпотентны):
- ReleaseQuota
- Cleanup temp artifacts (ingest/processing/publish)
- MarkAssetFailed

---

## Kafka Topics

#### Команды (commands.*):
- commands.asset.create
- commands.asset.mark_failed
- commands.quota.reserve
- commands.quota.release
- commands.ingest.prepare
- commands.processing.start
- commands.publish.finalize

#### События (events.*):
- events.asset.created
- events.asset.failed
- events.quota.reserved
- events.quota.failed
- events.quota.released
- events.ingest.ready
- events.ingest.uploaded
- events.ingest.failed
- events.processing.succeeded
- events.processing.failed
- events.publish.succeeded
- events.publish.failed

---

## Message Envelope (контракт)

Все сообщения в Kafka передаются в едином “конверте” (JSON):

- `message_id` — UUID сообщения (идемпотентность)
- `saga_id` — UUID саги (корреляция)
- `type` — тип команды/события
- `step` — шаг саги (для логов/аудита)
- `created_at` — RFC3339 timestamp
- `payload` — объект с данными домена

Пример:
```json
{
  "message_id": "m1",
  "saga_id": "s1",
  "type": "AssetCreate",
  "step": "CREATE_ASSET",
  "created_at": "2026-01-10T12:00:00Z",
  "payload": {
    "asset_id": "a1"
  }
}
```

## Local Infrastructure (Docker Compose)

Используется:
- Kafka + ZooKeeper
- Postgres
- Redis
- Kafka UI (Provectus)

Файл:
deploy/docker-compose.yml

### Запуск:
```bash
docker compose -f deploy/docker-compose.yml up -d
docker compose -f deploy/docker-compose.yml ps
```

### Kafka UI:
```arduino
http://localhost:8080
```

### Running Services
Каждый сервис запускается отдельно:

```bash

go run ./cmd/orchestrator
go run ./cmd/media
go run ./cmd/quota
go run ./cmd/ingest
go run ./cmd/processing
go run ./cmd/publish
```
## Development Notes
- Состояние саги хранится в Postgres (orchestrator).

- Kafka доставляет сообщения “как минимум один раз”, обработчики должны быть идемпотентными.

- Redis используется для дедупликации сообщений (TTL).

- В начале реализуется минимальный сквозной цикл command → event → command.

## Repo Structure

```text
cmd/            # entrypoints сервисов
internal/       # общий код (bootstrap, kafka, saga, идемпотентность)
deploy/         # docker-compose
```

## Quick Start

### Требования

- Go 1.22+
- Docker + Docker Compose
- Свободные порты: 9092 (Kafka), 5432 (Postgres), 6379 (Redis), 8080 (Kafka UI)

### Шаги установки

#### 1. Клонировать репозиторий

```bash
git clone 
cd media-platform
```

#### 2. Поднять инфраструктуру

```bash
docker compose -f deploy/docker-compose.yml up -d
```

Проверить, что контейнеры запущены:

```bash
docker ps
```

#### 3. Проверить Kafka UI

Открыть в браузере:

```
http://localhost:8080
```

Должны быть видны топики `commands.*` и `events.*`.

#### 4. Запустить сервисы

В отдельных терминалах:

```bash
# Терминал 1
go run ./cmd/orchestrator

# Терминал 2
go run ./cmd/media
```

> **Примечание**: Остальные сервисы будут добавляться по мере реализации саги.

#### 5. Smoke test (Kafka)

Отправить тестовое сообщение:

```bash
echo '{"message_id":"m1","saga_id":"s1","type":"AssetCreate","step":"CREATE_ASSET","created_at":"2026-01-10T12:00:00Z","payload":{"asset_id":"a1"}}' \
  | docker exec -i mp_kafka kafka-console-producer \
    --bootstrap-server localhost:9092 \
    --topic commands.asset.create
```

Сообщение должно появиться:
- в Kafka UI
- в логах `media` (после реализации consumer)
