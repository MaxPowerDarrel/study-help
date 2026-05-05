-- +goose Up
-- Add per-user active translation; existing accounts default to ESV
-- (the only provider that existed before this migration).
ALTER TABLE users      ADD COLUMN translation TEXT NOT NULL DEFAULT 'ESV';

-- Scope highlights and notes by translation. Existing rows were
-- authored against ESV, so the literal default is correct. Replace the
-- (user_id, book, chapter) index with a (user_id, translation, book,
-- chapter) index — every list query filters by translation now.
ALTER TABLE highlights ADD COLUMN translation TEXT NOT NULL DEFAULT 'ESV';
DROP INDEX IF EXISTS highlights_user_passage;
CREATE INDEX highlights_user_passage
    ON highlights(user_id, translation, book, chapter);

ALTER TABLE notes      ADD COLUMN translation TEXT NOT NULL DEFAULT 'ESV';
DROP INDEX IF EXISTS notes_user_passage;
CREATE INDEX notes_user_passage
    ON notes(user_id, translation, book, chapter);

-- +goose Down
DROP INDEX IF EXISTS notes_user_passage;
CREATE INDEX notes_user_passage ON notes(user_id, book, chapter);
ALTER TABLE notes DROP COLUMN translation;

DROP INDEX IF EXISTS highlights_user_passage;
CREATE INDEX highlights_user_passage ON highlights(user_id, book, chapter);
ALTER TABLE highlights DROP COLUMN translation;

ALTER TABLE users DROP COLUMN translation;
