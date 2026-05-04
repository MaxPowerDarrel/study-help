-- +goose Up
CREATE TABLE highlights (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id      INTEGER   NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    book         TEXT      NOT NULL,
    chapter      INTEGER   NOT NULL,
    start_verse  INTEGER   NOT NULL,
    start_offset INTEGER   NOT NULL,
    end_verse    INTEGER   NOT NULL,
    end_offset   INTEGER   NOT NULL,
    created_at   TIMESTAMP NOT NULL
);
CREATE INDEX highlights_user_passage ON highlights(user_id, book, chapter);

-- +goose Down
DROP INDEX IF EXISTS highlights_user_passage;
DROP TABLE IF EXISTS highlights;
