-- +goose Up
CREATE TABLE notes (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id      INTEGER   NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    book         TEXT      NOT NULL,
    chapter      INTEGER   NOT NULL,
    start_verse  INTEGER   NOT NULL,
    start_offset INTEGER   NOT NULL,
    end_verse    INTEGER   NOT NULL,
    end_offset   INTEGER   NOT NULL,
    body         TEXT      NOT NULL,
    created_at   TIMESTAMP NOT NULL,
    updated_at   TIMESTAMP NOT NULL
);
CREATE INDEX notes_user_passage ON notes(user_id, book, chapter);

-- +goose Down
DROP INDEX IF EXISTS notes_user_passage;
DROP TABLE IF EXISTS notes;
