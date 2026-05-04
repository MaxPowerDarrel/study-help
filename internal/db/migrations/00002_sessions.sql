-- +goose Up
CREATE TABLE sessions (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id       INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash    BLOB    NOT NULL,
    created_at    TIMESTAMP NOT NULL,
    last_seen_at  TIMESTAMP NOT NULL,
    expires_at    TIMESTAMP NOT NULL
);
CREATE UNIQUE INDEX sessions_token_hash_unique ON sessions(token_hash);
CREATE INDEX sessions_user_id ON sessions(user_id);

-- +goose Down
DROP INDEX IF EXISTS sessions_user_id;
DROP INDEX IF EXISTS sessions_token_hash_unique;
DROP TABLE IF EXISTS sessions;
