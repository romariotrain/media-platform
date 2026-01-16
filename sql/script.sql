CREATE TABLE IF NOT EXISTS media (
                                     id uuid PRIMARY KEY,
                                     status text NOT NULL,
                                     type text NOT NULL,
                                     source text NOT NULL,
                                     created_at timestamptz NOT NULL,
                                     updated_at timestamptz NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_media_status ON media(status);
