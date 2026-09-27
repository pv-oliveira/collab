BEGIN;

CREATE TABLE documents (
    id         TEXT        PRIMARY KEY,
    user_id    TEXT        NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    title      TEXT        NOT NULL DEFAULT '',
    content    TEXT        NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Toda consulta de documentos filtra pelo dono.
CREATE INDEX documents_user_id_idx ON documents (user_id);

COMMIT;
