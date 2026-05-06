import { useCallback } from "react";
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
import { useResource } from "../platform/useResource";
import type { TranslationID } from "../translations/catalog";

export type NotesState = {
  notes: Note[];
  loading: boolean;
};

export type NotesActions = {
  create: (input: CreateInput) => Promise<CreateResult>;
  update: (id: number, body: string) => Promise<UpdateResult>;
  remove: (id: number) => Promise<DeleteResult>;
};

const EMPTY: Note[] = [];

// useNotes owns the per-passage notes cache. It re-fetches on
// (book, chapter, translation, enabled) change and after every successful
// mutation; no optimistic updates at v1. When `enabled` is false (guest),
// the hook stays empty and never hits the network.
export function useNotes(
  book: string | null,
  chapter: number | null,
  translation: TranslationID,
  enabled: boolean,
): NotesState & NotesActions {
  const fetcher =
    enabled && book && chapter
      ? async (signal: AbortSignal) => {
          const res = await listNotes(book, chapter, translation, signal);
          return res.kind === "ok" ? res.notes : null;
        }
      : null;

  const { data, loading, refetch } = useResource(fetcher, EMPTY, [
    book,
    chapter,
    translation,
    enabled,
  ]);

  const create = useCallback(
    async (input: CreateInput): Promise<CreateResult> => {
      const res = await createNote(input, translation);
      if (res.kind === "ok") await refetch();
      return res;
    },
    [refetch, translation],
  );

  const update = useCallback(
    async (id: number, body: string): Promise<UpdateResult> => {
      const res = await updateNote(id, body);
      if (res.kind === "ok") await refetch();
      return res;
    },
    [refetch],
  );

  const remove = useCallback(
    async (id: number): Promise<DeleteResult> => {
      const res = await deleteNote(id);
      if (res.kind === "ok") await refetch();
      return res;
    },
    [refetch],
  );

  return { notes: data, loading, create, update, remove };
}
