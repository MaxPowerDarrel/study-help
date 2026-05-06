import { useState } from "react";
import type { ToolbarTuple } from "../highlights/HighlightToolbar";
import {
  type CreateResult,
  type DeleteResult,
  type Note,
  type UpdateResult,
} from "./api";
import styles from "./NotesDrawer.module.css";

type Props = {
  open: boolean;
  onClose: () => void;
  title: string;
  notes: Note[];
  loading: boolean;
  pendingTuple: ToolbarTuple | null;
  onCancelPending: () => void;
  onCreate: (body: string) => Promise<CreateResult>;
  onUpdate: (id: number, body: string) => Promise<UpdateResult>;
  onRemove: (id: number) => Promise<DeleteResult>;
  onScrollToAnchor: (n: Note) => void;
  onError: (message: string) => void;
  // When true (Daily aggregated view), entry refs include book + chapter
  // since the title alone can't disambiguate the spanning chapters.
  showChapter?: boolean;
};

export function NotesDrawer({
  open,
  onClose,
  title,
  notes,
  loading,
  pendingTuple,
  onCancelPending,
  onCreate,
  onUpdate,
  onRemove,
  onScrollToAnchor,
  onError,
  showChapter = false,
}: Props) {
  if (!open) return null;
  return (
    <div className={styles.overlay} role="dialog" aria-label="Notes">
      <div className={styles.pane}>
        <header className={styles.header}>
          <h2>Notes — {title}</h2>
          <button
            type="button"
            className={styles.close}
            onClick={onClose}
            aria-label="Close notes"
          >
            ×
          </button>
        </header>

        {pendingTuple && (
          <Composer
            tuple={pendingTuple}
            onCancel={onCancelPending}
            onCreate={onCreate}
            onError={onError}
          />
        )}

        {loading && <div className={styles.spinner} aria-label="Loading" />}

        {!loading && (
          <ul className={styles.entryList}>
            {notes.map((note) => (
              <Entry
                key={note.id}
                note={note}
                showChapter={showChapter}
                onUpdate={onUpdate}
                onRemove={onRemove}
                onScrollToAnchor={onScrollToAnchor}
                onError={onError}
              />
            ))}
          </ul>
        )}
      </div>
    </div>
  );
}

function Composer({
  tuple,
  onCancel,
  onCreate,
  onError,
}: {
  tuple: ToolbarTuple;
  onCancel: () => void;
  onCreate: (body: string) => Promise<CreateResult>;
  onError: (message: string) => void;
}) {
  const [body, setBody] = useState("");
  const [invalid, setInvalid] = useState(false);
  const [saving, setSaving] = useState(false);

  async function handleSave() {
    if (saving) return;
    setSaving(true);
    setInvalid(false);
    const res = await onCreate(body);
    setSaving(false);
    if (res.kind === "ok") return;
    if (res.kind === "invalid") {
      setInvalid(true);
      return;
    }
    if (res.kind === "guest") {
      onError("Sign in to save notes.");
      return;
    }
    onError("Something went wrong, try again");
  }

  return (
    <div className={styles.composer}>
      <span className={styles.composerRef}>{formatRef(tuple)}</span>
      <textarea
        className={styles.composerTextarea}
        value={body}
        onChange={(e) => setBody(e.target.value)}
        placeholder="Write a note…"
        autoFocus
      />
      {invalid && (
        <span className={styles.errorMessage} role="alert">
          Note is too long.
        </span>
      )}
      <div className={styles.composerActions}>
        <button
          type="button"
          className={styles.cancelButton}
          onClick={onCancel}
        >
          Cancel
        </button>
        <button
          type="button"
          className={styles.saveButton}
          onClick={handleSave}
          disabled={saving}
        >
          Save
        </button>
      </div>
    </div>
  );
}

function Entry({
  note,
  showChapter,
  onUpdate,
  onRemove,
  onScrollToAnchor,
  onError,
}: {
  note: Note;
  showChapter: boolean;
  onUpdate: (id: number, body: string) => Promise<UpdateResult>;
  onRemove: (id: number) => Promise<DeleteResult>;
  onScrollToAnchor: (n: Note) => void;
  onError: (message: string) => void;
}) {
  const [editing, setEditing] = useState(false);
  const [draft, setDraft] = useState(note.body);
  const [invalid, setInvalid] = useState(false);
  const [saving, setSaving] = useState(false);

  const edited = note.updated_at !== note.created_at;

  async function handleSave() {
    if (saving) return;
    setSaving(true);
    setInvalid(false);
    const res = await onUpdate(note.id, draft);
    setSaving(false);
    if (res.kind === "ok") {
      setEditing(false);
      return;
    }
    if (res.kind === "invalid") {
      setInvalid(true);
      return;
    }
    if (res.kind === "not_found") {
      onError("That note no longer exists.");
      setEditing(false);
      return;
    }
    onError("Something went wrong, try again");
  }

  async function handleDelete() {
    const res = await onRemove(note.id);
    if (res.kind === "ok" || res.kind === "not_found") return;
    onError("Something went wrong, try again");
  }

  return (
    <li className={styles.entry}>
      <span className={styles.entryRef}>
        {showChapter
          ? formatChapterRef(note)
          : formatRef({
              start_verse: note.start_verse,
              start_offset: note.start_offset,
              end_verse: note.end_verse,
              end_offset: note.end_offset,
            })}
      </span>

      {editing ? (
        <>
          <textarea
            className={styles.editorTextarea}
            value={draft}
            onChange={(e) => setDraft(e.target.value)}
            autoFocus
          />
          {invalid && (
            <span className={styles.errorMessage} role="alert">
              Note is too long.
            </span>
          )}
          <div className={styles.editorActions}>
            <button
              type="button"
              className={styles.cancelButton}
              onClick={() => {
                setEditing(false);
                setDraft(note.body);
                setInvalid(false);
              }}
            >
              Cancel
            </button>
            <button
              type="button"
              className={styles.saveButton}
              onClick={handleSave}
              disabled={saving}
            >
              Save
            </button>
          </div>
        </>
      ) : (
        <>
          <p
            className={styles.entryBody}
            onClick={() => onScrollToAnchor(note)}
          >
            {note.body}
          </p>
          <div className={styles.entryMeta}>
            <span>{formatDate(note.created_at)}</span>
            {edited && <span className={styles.editedTag}>edited</span>}
          </div>
          <div className={styles.entryControls}>
            <button
              type="button"
              className={styles.editButton}
              onClick={() => setEditing(true)}
            >
              Edit
            </button>
            <button
              type="button"
              className={styles.deleteButton}
              onClick={handleDelete}
            >
              Delete
            </button>
          </div>
        </>
      )}
    </li>
  );
}

function formatRef(t: {
  start_verse: number;
  start_offset: number;
  end_verse: number;
  end_offset: number;
}): string {
  if (t.start_verse === t.end_verse) return `Verse ${t.start_verse}`;
  return `Verses ${t.start_verse}–${t.end_verse}`;
}

function formatChapterRef(n: Note): string {
  if (n.start_verse === n.end_verse) {
    return `${n.book} ${n.chapter}:${n.start_verse}`;
  }
  return `${n.book} ${n.chapter}:${n.start_verse}–${n.end_verse}`;
}

function formatDate(iso: string): string {
  const d = new Date(iso);
  if (isNaN(d.getTime())) return iso;
  return d.toLocaleDateString(undefined, {
    year: "numeric",
    month: "short",
    day: "numeric",
  });
}
