# Media Platform

Микросервисная платформа для загрузки, обработки и публикации медиафайлов.
Event-Driven Architecture на Go + Apache Kafka + PostgreSQL.

## Архитектура

```
                         ┌──────────────────┐
                         │   HTTP-клиент    │
                         └──┬───────────┬───┘
                            │           │
                       POST /sagas   POST /upload
                            │           │
                   ┌────────▼──┐   ┌────▼──────┐
                   │Orchestrator│   │  Ingest   │
                   │   :8084    │   │   :8082   │
                   └──┬──┬──┬──┘   └───────────┘
                      │  │  │           ▲
             commands │  │  │  events   │
             (Kafka)  │  │  │  (Kafka)  │
                      │  │  │           │
          ┌───────────┘  │  └─────────┐ │
          ▼              ▼            ▼ │
    ┌──────────┐  ┌───────────┐  ┌─────┴───┐
    │Processing│  │  Publish   │  │ Ingest  │
    │ (worker) │  │   :8083    │  │  :8082  │
    └──────────┘  └───────────┘  └─────────┘
```

## Сервисы

| Сервис | Порт | Роль |
|--------|------|------|
| [Orchestrator](cmd/orchestrator/) | 8084 | Координатор Saga-пайплайна |
| [Ingest](cmd/ingest/) | 8082 | Приём и хранение файлов |
| [Processing](cmd/processing/) | — | Обработка через FFmpeg |
| [Publish](cmd/publish/) | 8083 | Публикация и формирование URL |
| [Media](cmd/media/) | 8081 | CRUD метаданных медиа |
| [Quota](cmd/quota/) | — | Управление квотами (частично) |

## Пайплайн

```
POST /sagas → PREPARE_INGEST → WAIT_UPLOAD → PROCESS_MEDIA → PUBLISH_MEDIA → DONE
```

1. Клиент создаёт сагу через `POST /sagas` (Orchestrator)
2. Orchestrator отправляет команду Ingest → подготовить загрузку
3. Клиент загружает файл через `POST /upload?token=...` (Ingest)
4. Orchestrator отправляет команду Processing → обработать файл
5. FFmpeg транскодирует видео (720p, 1080p, thumbnail, метаданные)
6. Orchestrator отправляет команду Publish → опубликовать
7. Publish копирует файлы и формирует публичные URL
8. Saga завершена

## Kafka Topics

| Topic | Направление | Описание |
|-------|------------|----------|
| `commands.ingest.prepare` | Orchestrator → Ingest | Подготовить загрузку |
| `commands.processing.start` | Orchestrator → Processing | Обработать файл |
| `commands.publish.start` | Orchestrator → Publish | Опубликовать файлы |
| `events.ingest.ready` | Ingest → Orchestrator | Токен готов |
| `events.ingest.uploaded` | Ingest → Orchestrator | Файл загружен |
| `events.processing.succeeded` | Processing → Orchestrator | Обработка завершена |
| `events.processing.failed` | Processing → Orchestrator | Ошибка обработки |
| `events.publish.succeeded` | Publish → Orchestrator | Публикация завершена |
| `events.publish.failed` | Publish → Orchestrator | Ошибка публикации |

## Паттерны

- **Saga (Orchestration)** — Orchestrator хранит state machine, последовательно проводит файл через шаги
- **Transactional Outbox** — события записываются в `*_outbox` таблицу в одной транзакции с данными, фоновый publisher отправляет их в Kafka
- **Idempotency** — все сервисы проверяют `saga_id` перед выполнением, повторная команда не создаёт дубль

## Структура проекта

```
cmd/                        # Точки входа сервисов
  ├── orchestrator/
  ├── ingest/
  ├── processing/
  ├── publish/
  ├── media/
  └── quota/
internal/                   # Бизнес-логика
  ├── cli/                  # Общий graceful shutdown
  └── <service>/
      ├── models/           # Сущности, ошибки, команды, события
      ├── repository/       # Интерфейс репозитория
      ├── storage/postgres/ # SQL-реализация
      ├── storage/fs/       # Файловое хранилище
      ├── service/          # Бизнес-логика
      ├── kafka/            # Kafka consumer
      ├── outbox/           # Outbox publisher
      └── httpapi/          # HTTP API
sql/
  └── script.sql            # Все миграции
frontend/
  └── index.html            # Web UI (vanilla HTML/JS)
```

## Запуск

### Зависимости

- Go 1.25+
- PostgreSQL
- Apache Kafka (broker на localhost:9092)
- FFmpeg (для Processing сервиса)

### База данных

```bash
createdb media_platform
psql -d media_platform -f sql/script.sql
```

### Переменные окружения

```bash
export DATABASE_URL="postgres://user:pass@localhost:5432/media_platform?sslmode=disable"
export UPLOAD_DIR="uploads"        # для Ingest
export PUBLISH_DIR="published"     # для Publish
export PUBLISH_BASE_URL="http://localhost:8083/static"
```

### Запуск сервисов

```bash
# Каждый в отдельном терминале
go run ./cmd/orchestrator
go run ./cmd/ingest
go run ./cmd/processing
go run ./cmd/publish
go run ./cmd/media
```

### Web UI

Открыть `frontend/index.html` в браузере. Настроить base URL сервисов в верхней панели.

## БД таблицы

| Таблица | Сервис | Назначение |
|---------|--------|------------|
| `media` | Media | Метаданные медиа |
| `upload_sessions` | Ingest | Сессии загрузки |
| `processing_tasks` | Processing | Задачи обработки |
| `publications` | Publish | Записи публикаций |
| `sagas` | Orchestrator | State machine саг |
| `*_outbox` | Все | Transactional outbox |

## Технологии

- **Go** — язык
- **PostgreSQL** (pgx/sqlx) — хранилище
- **Apache Kafka** (segmentio/kafka-go) — шина событий
- **FFmpeg** — обработка видео
- **zerolog** — логирование
