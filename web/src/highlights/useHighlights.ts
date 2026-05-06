import { useCallback } from "react";
import {
  type CreateInput,
  type CreateResult,
  type DeleteResult,
  type Highlight,
  createHighlight,
  deleteHighlight,
  listHighlights,
} from "./api";
import { useResource } from "../platform/useResource";
import type { TranslationID } from "../translations/catalog";

export type HighlightsState = {
  highlights: Highlight[];
  loading: boolean;
};

export type HighlightsActions = {
  create: (input: CreateInput) => Promise<CreateResult>;
  remove: (id: number) => Promise<DeleteResult>;
};

const EMPTY: Highlight[] = [];

// useHighlights owns the per-passage highlight cache. It re-fetches on
// (book, chapter, translation) change and after every successful mutation;
// no optimistic updates at v1 (specs/highlights.md Decisions, 2026-05-04).
// When `enabled` is false (guest), the hook stays empty and never hits
// the network.
export function useHighlights(
  book: string | null,
  chapter: number | null,
  translation: TranslationID,
  enabled: boolean,
): HighlightsState & HighlightsActions {
  const fetcher =
    enabled && book && chapter
      ? async (signal: AbortSignal) => {
          const res = await listHighlights(book, chapter, translation, signal);
          return res.kind === "ok" ? res.highlights : null;
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
      const res = await createHighlight(input, translation);
      if (res.kind === "ok") await refetch();
      return res;
    },
    [refetch, translation],
  );

  const remove = useCallback(
    async (id: number): Promise<DeleteResult> => {
      const res = await deleteHighlight(id);
      if (res.kind === "ok") await refetch();
      return res;
    },
    [refetch],
  );

  return { highlights: data, loading, create, remove };
}
