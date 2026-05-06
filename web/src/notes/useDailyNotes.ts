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
import type { NotesActions, NotesState } from "./useNotes";
import { useResource } from "../platform/useResource";
import type { TranslationID } from "../translations/catalog";

const EMPTY: Note[] = [];

// useDailyNotes aggregates listNotes across multiple chapters for the
// Daily tab. Surface matches useNotes so the drawer can be tab-agnostic.
// chaptersKey is the dependency anchor — a stable string lets us pass a
// fresh array literal per render without re-fetching.
export function useDailyNotes(
  book: string | null,
  chapters: number[],
  translation: TranslationID,
  enabled: boolean,
): NotesState & NotesActions {
  const chaptersKey = chapters.join(",");

  const fetcher =
    enabled && book && chapters.length > 0
      ? async (signal: AbortSignal) => {
          const results = await Promise.all(
            chapters.map((c) => listNotes(book, c, translation, signal)),
          );
          const merged: Note[] = [];
          for (const r of results) {
            if (r.kind === "ok") merged.push(...r.notes);
          }
          merged.sort((a, b) => {
            if (a.chapter !== b.chapter) return a.chapter - b.chapter;
            return a.created_at < b.created_at ? -1 : 1;
          });
          return merged;
        }
      : null;

  const { data, loading, refetch } = useResource(fetcher, EMPTY, [
    book,
    chaptersKey,
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
