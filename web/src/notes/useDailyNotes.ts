import { useCallback, useEffect, useState } from "react";
import {
  type CreateInput,
  type CreateResult,
  type DeleteResult,
  type Note,
  type UpdateResult,
  createNote,
  deleteNote,
  listNotes,
  updateNote,
} from "./api";
import type { NotesActions, NotesState } from "./useNotes";
import type { TranslationID } from "../translations/catalog";

// Aggregates listNotes across multiple chapters for the Daily tab. Surface
// matches useNotes so the drawer can be tab-agnostic. chaptersKey is the
// dependency anchor — a stable string lets us pass a fresh array literal
// per render without re-fetching.
export function useDailyNotes(
  book: string | null,
  chapters: number[],
  translation: TranslationID,
  enabled: boolean,
): NotesState & NotesActions {
  const [notes, setNotes] = useState<Note[]>([]);
  const [loading, setLoading] = useState(false);
  const chaptersKey = chapters.join(",");

  const fetchAll = useCallback(async () => {
    if (!enabled || !book || chapters.length === 0) {
      setNotes([]);
      return;
    }
    setLoading(true);
    const results = await Promise.all(
      chapters.map((c) => listNotes(book, c, translation)),
    );
    const merged: Note[] = [];
    for (const r of results) {
      if (r.kind === "ok") merged.push(...r.notes);
    }
    merged.sort((a, b) => {
      if (a.chapter !== b.chapter) return a.chapter - b.chapter;
      return a.created_at < b.created_at ? -1 : 1;
    });
    setLoading(false);
    setNotes(merged);
    // chaptersKey is the canonical dependency for `chapters`.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [book, chaptersKey, translation, enabled]);

  useEffect(() => {
    let cancelled = false;
    void (async () => {
      if (!enabled || !book || chapters.length === 0) {
        if (!cancelled) setNotes([]);
        return;
      }
      setLoading(true);
      const results = await Promise.all(
        chapters.map((c) => listNotes(book, c, translation)),
      );
      if (cancelled) return;
      const merged: Note[] = [];
      for (const r of results) {
        if (r.kind === "ok") merged.push(...r.notes);
      }
      merged.sort((a, b) => {
        if (a.chapter !== b.chapter) return a.chapter - b.chapter;
        return a.created_at < b.created_at ? -1 : 1;
      });
      setLoading(false);
      setNotes(merged);
    })();
    return () => {
      cancelled = true;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [book, chaptersKey, translation, enabled]);

  const create = useCallback(
    async (input: CreateInput): Promise<CreateResult> => {
      const res = await createNote(input, translation);
      if (res.kind === "ok") await fetchAll();
      return res;
    },
    [fetchAll, translation],
  );

  const update = useCallback(
    async (id: number, body: string): Promise<UpdateResult> => {
      const res = await updateNote(id, body);
      if (res.kind === "ok") await fetchAll();
      return res;
    },
    [fetchAll],
  );

  const remove = useCallback(
    async (id: number): Promise<DeleteResult> => {
      const res = await deleteNote(id);
      if (res.kind === "ok") await fetchAll();
      return res;
    },
    [fetchAll],
  );

  return { notes, loading, create, update, remove };
}
