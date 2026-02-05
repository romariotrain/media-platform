# Ingest Service

Приём и хранение файлов. Two-phase upload: сначала получает токен от оркестратора, затем принимает файл от клиента.

## Порт

`:8082`

## Процесс загрузки

```
1. Orchestrator → [commands.ingest.prepare] → Ingest
   Ingest создаёт upload session + token
   Ingest → [events.ingest.ready]

2. Клиент → GET /upload-token?token=abc123
   Получает информацию о сессии

3. Клиент → POST /upload?token=abc123 + file (multipart)
   Ingest сохраняет файл на диск
   Ingest → [events.ingest.uploaded]
```

## HTTP API

| Метод | Endpoint | Описание |
|-------|----------|----------|
| `GET` | `/upload-token?token=<string>` | Информация о токене |
| `POST` | `/upload?token=<string>` | Загрузка файла (multipart/form-data, field `file`) |
| `GET` | `/health` | Health check |

## Kafka

**Слушает:**
- `commands.ingest.prepare` — команда подготовить загрузку

**Отправляет:**
- `events.ingest.ready` — токен создан, ждём файл
- `events.ingest.uploaded` — файл загружен

## Хранилище файлов

Файлы сохраняются в локальную FS. Путь задаётся переменной `UPLOAD_DIR` (по умолчанию `uploads/`).

## БД таблицы

- `upload_sessions` — сессии загрузки (token, status, file_name, storage_path)
- `ingest_outbox` — outbox событий

## Ключевые файлы

- `service/prepare.go` — создание upload session с токеном
- `service/upload.go` — приём файла, валидация, сохранение
- `storage/fs/file_store.go` — запись файлов на диск
- `domain/status.go` — state machine статусов сессии
