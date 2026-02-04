CREATE TABLE IF NOT EXISTS media (
                                     id uuid PRIMARY KEY,
                                     status text NOT NULL,
                                     type text NOT NULL,
                                     source text NOT NULL,
                                     created_at timestamptz NOT NULL,
                                     updated_at timestamptz NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_media_status ON media(status);

-- ============================================================
-- Ingest Service Tables
-- ============================================================

CREATE TABLE IF NOT EXISTS upload_sessions (
    id              UUID PRIMARY KEY,
    saga_id         UUID NOT NULL,
    asset_id        UUID NOT NULL,
    user_id         TEXT NOT NULL,
    token           TEXT NOT NULL UNIQUE,
    status          TEXT NOT NULL DEFAULT 'pending',
    file_name       TEXT,
    file_size       BIGINT,
    content_type    TEXT,
    storage_path    TEXT,
    expires_at      TIMESTAMPTZ NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_upload_sessions_token ON upload_sessions(token);
CREATE INDEX IF NOT EXISTS idx_upload_sessions_saga_id ON upload_sessions(saga_id);
CREATE INDEX IF NOT EXISTS idx_upload_sessions_asset_id ON upload_sessions(asset_id);

CREATE TABLE IF NOT EXISTS ingest_outbox (
    id              BIGSERIAL PRIMARY KEY,
    event_id        TEXT NOT NULL,
    event_type      TEXT NOT NULL,
    aggregate_id    TEXT NOT NULL,
    saga_id         UUID,
    payload         JSONB NOT NULL,
    occurred_at     TIMESTAMPTZ NOT NULL,
    processed_at    TIMESTAMPTZ
);

-- ============================================================
-- Processing Service Tables
-- ============================================================

CREATE TABLE IF NOT EXISTS processing_tasks (
    id              UUID PRIMARY KEY,
    saga_id         UUID NOT NULL,
    asset_id        UUID NOT NULL,
    status          TEXT NOT NULL DEFAULT 'pending',
    input_path      TEXT NOT NULL,
    output_paths    JSONB,
    metadata        JSONB,
    error_message   TEXT,
    started_at      TIMESTAMPTZ,
    completed_at    TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_processing_tasks_saga_id ON processing_tasks(saga_id);
CREATE INDEX IF NOT EXISTS idx_processing_tasks_asset_id ON processing_tasks(asset_id);
CREATE INDEX IF NOT EXISTS idx_processing_tasks_status ON processing_tasks(status);

CREATE TABLE IF NOT EXISTS processing_outbox (
    id              BIGSERIAL PRIMARY KEY,
    event_id        TEXT NOT NULL,
    event_type      TEXT NOT NULL,
    aggregate_id    TEXT NOT NULL,
    saga_id         UUID,
    payload         JSONB NOT NULL,
    occurred_at     TIMESTAMPTZ NOT NULL,
    processed_at    TIMESTAMPTZ
);

-- ============================================================
-- Publish Service Tables
-- ============================================================

CREATE TABLE IF NOT EXISTS publications (
    id              UUID PRIMARY KEY,
    saga_id         UUID NOT NULL,
    asset_id        UUID NOT NULL,
    status          TEXT NOT NULL DEFAULT 'pending',
    source_paths    JSONB NOT NULL,
    published_paths JSONB,
    public_urls     JSONB,
    error_message   TEXT,
    published_at    TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_publications_saga_id ON publications(saga_id);
CREATE INDEX IF NOT EXISTS idx_publications_asset_id ON publications(asset_id);
CREATE INDEX IF NOT EXISTS idx_publications_status ON publications(status);

CREATE TABLE IF NOT EXISTS publish_outbox (
    id              BIGSERIAL PRIMARY KEY,
    event_id        TEXT NOT NULL,
    event_type      TEXT NOT NULL,
    aggregate_id    TEXT NOT NULL,
    saga_id         UUID,
    payload         JSONB NOT NULL,
    occurred_at     TIMESTAMPTZ NOT NULL,
    processed_at    TIMESTAMPTZ
);

-- ============================================================
-- Orchestrator Service Tables
-- ============================================================

CREATE TABLE IF NOT EXISTS sagas (
    id              UUID PRIMARY KEY,
    asset_id        UUID NOT NULL,
    user_id         TEXT NOT NULL,
    status          TEXT NOT NULL DEFAULT 'created',
    current_step    TEXT NOT NULL DEFAULT 'PREPARE_INGEST',
    payload         JSONB,
    error_message   TEXT,
    started_at      TIMESTAMPTZ,
    completed_at    TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_sagas_asset_id ON sagas(asset_id);
CREATE INDEX IF NOT EXISTS idx_sagas_status ON sagas(status);
CREATE INDEX IF NOT EXISTS idx_sagas_user_id ON sagas(user_id);

CREATE TABLE IF NOT EXISTS orchestrator_outbox (
    id              BIGSERIAL PRIMARY KEY,
    event_id        TEXT NOT NULL,
    event_type      TEXT NOT NULL,
    aggregate_id    TEXT NOT NULL,
    saga_id         UUID,
    payload         JSONB NOT NULL,
    occurred_at     TIMESTAMPTZ NOT NULL,
    processed_at    TIMESTAMPTZ
);
