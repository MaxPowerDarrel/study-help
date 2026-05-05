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

export type NotesState = {
  notes: Note[];
  loading: boolean;
};

export type NotesActions = {
  create: (input: CreateInput) => Promise<CreateResult>;
  update: (id: number, body: string) => Promise<UpdateResult>;
  remove: (id: number) => Promise<DeleteResult>;
};

// useNotes owns the per-passage notes cache. It re-fetches on
// (book, chapter, enabled) change and after every successful mutation;
// no optimistic updates at v1. When `enabled` is false (guest), the
// hook stays empty and never hits the network.
export function useNotes(
  book: string | null,
  chapter: number | null,
  enabled: boolean,
): NotesState & NotesActions {
  const [notes, setNotes] = useState<Note[]>([]);
  const [loading, setLoading] = useState(false);

  const fetchAll = useCallback(async () => {
    if (!enabled || !book || !chapter) {
      setNotes([]);
      return;
    }
    setLoading(true);
    const res = await listNotes(book, chapter);
    setLoading(false);
    if (res.kind === "ok") {
      setNotes(res.notes);
    } else {
      setNotes([]);
    }
  }, [book, chapter, enabled]);

  useEffect(() => {
    let cancelled = false;
    void (async () => {
      if (!enabled || !book || !chapter) {
        if (!cancelled) setNotes([]);
        return;
      }
      setLoading(true);
      const res = await listNotes(book, chapter);
      if (cancelled) return;
      setLoading(false);
      setNotes(res.kind === "ok" ? res.notes : []);
    })();
    return () => {
      cancelled = true;
    };
  }, [book, chapter, enabled]);

  const create = useCallback(
    async (input: CreateInput): Promise<CreateResult> => {
      const res = await createNote(input);
      if (res.kind === "ok") await fetchAll();
      return res;
    },
    [fetchAll],
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
