import { useCallback, useEffect, useState } from "react";
import {
  type CreateInput,
  type CreateResult,
  type DeleteResult,
  type Highlight,
  createHighlight,
  deleteHighlight,
  listHighlights,
} from "./api";

export type HighlightsState = {
  highlights: Highlight[];
  loading: boolean;
};

export type HighlightsActions = {
  create: (input: CreateInput) => Promise<CreateResult>;
  remove: (id: number) => Promise<DeleteResult>;
};

// useHighlights owns the per-passage highlight cache. It re-fetches on
// (book, chapter) change and after every successful mutation; no
// optimistic updates at v1 (specs/highlights.md Decisions, 2026-05-04).
// When `enabled` is false (guest), the hook stays empty and never hits
// the network.
export function useHighlights(
  book: string | null,
  chapter: number | null,
  enabled: boolean,
): HighlightsState & HighlightsActions {
  const [highlights, setHighlights] = useState<Highlight[]>([]);
  const [loading, setLoading] = useState(false);

  const fetchAll = useCallback(async () => {
    if (!enabled || !book || !chapter) {
      setHighlights([]);
      return;
    }
    setLoading(true);
    const res = await listHighlights(book, chapter);
    setLoading(false);
    if (res.kind === "ok") {
      setHighlights(res.highlights);
    } else {
      setHighlights([]);
    }
  }, [book, chapter, enabled]);

  useEffect(() => {
    let cancelled = false;
    void (async () => {
      if (!enabled || !book || !chapter) {
        if (!cancelled) setHighlights([]);
        return;
      }
      setLoading(true);
      const res = await listHighlights(book, chapter);
      if (cancelled) return;
      setLoading(false);
      setHighlights(res.kind === "ok" ? res.highlights : []);
    })();
    return () => {
      cancelled = true;
    };
  }, [book, chapter, enabled]);

  const create = useCallback(
    async (input: CreateInput): Promise<CreateResult> => {
      const res = await createHighlight(input);
      if (res.kind === "ok") await fetchAll();
      return res;
    },
    [fetchAll],
  );

  const remove = useCallback(
    async (id: number): Promise<DeleteResult> => {
      const res = await deleteHighlight(id);
      if (res.kind === "ok") await fetchAll();
      return res;
    },
    [fetchAll],
  );

  return { highlights, loading, create, remove };
}
