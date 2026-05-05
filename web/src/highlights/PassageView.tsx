import { useCallback, useEffect, useRef, useState } from "react";
import {
  defaultSelectionAdapter,
  type SelectionAdapter,
} from "../platform/SelectionAdapter";
import { applyHighlights, findHighlightId } from "./applyHighlights";
import { HighlightToolbar, type ToolbarMode } from "./HighlightToolbar";
import { rangeToTuple } from "./parseSelection";
import { useHighlights } from "./useHighlights";

type Props = {
  html: string;
  book: string;
  chapter: number;
  isSignedIn: boolean;
  onGuestSignin: () => void;
  selectionAdapter?: SelectionAdapter;
};

// PassageView owns the rendered ESV HTML, the highlights overlay, the
// floating toolbar, and the selection event wiring. Extracted from
// App.tsx so the highlight machinery has a stable container ref to
// attach listeners to.
export function PassageView({
  html,
  book,
  chapter,
  isSignedIn,
  onGuestSignin,
  selectionAdapter = defaultSelectionAdapter,
}: Props) {
  const articleRef = useRef<HTMLElement | null>(null);
  const { highlights, create, remove } = useHighlights(
    book,
    chapter,
    isSignedIn,
  );
  const [mode, setMode] = useState<ToolbarMode>({ kind: "idle" });

  // Re-apply highlight marks whenever the HTML or the highlights change.
  useEffect(() => {
    const article = articleRef.current;
    if (!article) return;
    applyHighlights(article, highlights);
  }, [html, highlights]);

  // Dismiss the toolbar on book/chapter change.
  useEffect(() => {
    setMode({ kind: "idle" });
  }, [book, chapter]);

  // Dismiss on window resize — the toolbar's anchor rect is fragile.
  useEffect(() => {
    function onResize() {
      setMode((m) => (m.kind === "idle" ? m : { kind: "idle" }));
    }
    window.addEventListener("resize", onResize);
    return () => window.removeEventListener("resize", onResize);
  }, []);

  // Show the toolbar once the user finishes a selection gesture
  // (mouseup on desktop, touchend on iPad/iOS). selectionchange is
  // intentionally NOT used to *show* the toolbar — it fires on every
  // pixel of the drag, and re-rendering mid-drag confuses the
  // browser's selection tracking. selectionchange is still used below
  // to *dismiss* the toolbar when the selection is cleared (e.g., the
  // user clicks elsewhere in the article).
  useEffect(() => {
    const article = articleRef.current;
    if (!article) return;

    function showIfSelection() {
      const article = articleRef.current;
      if (!article) return;
      const info = selectionAdapter.current(article);
      if (!info) return;
      if (!isSignedIn) {
        setMode({ kind: "guest", rect: info.rect });
        return;
      }
      // Resolve to verse/offset tuple at capture time. Storing
      // primitives in mode means later DOM mutations can't invalidate
      // the highlight target the way they do a live Range.
      const tuple = rangeToTuple(info.range, article);
      if (!tuple) {
        setMode({
          kind: "error",
          rect: info.rect,
          message: "Selection isn't inside a verse.",
        });
        return;
      }
      setMode({
        kind: "selection",
        rect: info.rect,
        tuple: {
          start_verse: tuple.start_verse,
          start_offset: tuple.start_offset,
          end_verse: tuple.end_verse,
          end_offset: tuple.end_offset,
        },
      });
    }

    document.addEventListener("mouseup", showIfSelection);
    article.addEventListener("touchend", showIfSelection);
    return () => {
      document.removeEventListener("mouseup", showIfSelection);
      article.removeEventListener("touchend", showIfSelection);
    };
  }, [isSignedIn, selectionAdapter]);

  // Dismiss the toolbar when the selection becomes empty (without
  // disturbing other modes like "existing" or "error").
  useEffect(() => {
    function check() {
      const article = articleRef.current;
      if (!article) return;
      const info = selectionAdapter.current(article);
      if (info) return;
      setMode((m) =>
        m.kind === "selection" || m.kind === "guest" ? { kind: "idle" } : m,
      );
    }
    document.addEventListener("selectionchange", check);
    return () => document.removeEventListener("selectionchange", check);
  }, [selectionAdapter]);

  // Tap-on-existing-highlight: when the user clicks inside a <mark>
  // without an active selection, surface the "Remove highlight" toolbar.
  // A drag-selection that overlaps a highlight is a different gesture
  // (selection wins, per spec Decisions 2026-05-04).
  useEffect(() => {
    const article = articleRef.current;
    if (!article) return;
    function onClick(e: MouseEvent) {
      const sel = window.getSelection();
      if (sel && !sel.isCollapsed) return; // selection takes precedence
      const id = findHighlightId(e.target as Node | null);
      if (id == null) return;
      const target = e.target as Element;
      const rect = target.getBoundingClientRect();
      setMode({ kind: "existing", rect, highlightId: id });
    }
    article.addEventListener("click", onClick);
    return () => article.removeEventListener("click", onClick);
  }, []);

  const handleHighlight = useCallback(async () => {
    if (mode.kind !== "selection") return;
    const { rect, tuple } = mode;
    const res = await create({
      book,
      chapter,
      start_verse: tuple.start_verse,
      start_char_offset: tuple.start_offset,
      end_verse: tuple.end_verse,
      end_char_offset: tuple.end_offset,
    });
    if (res.kind === "ok") {
      selectionAdapter.clear();
      setMode({ kind: "idle" });
      return;
    }
    if (res.kind === "overlap") {
      setMode({
        kind: "error",
        rect,
        message: "Overlaps an existing highlight.",
      });
      return;
    }
    if (res.kind === "guest") {
      setMode({ kind: "guest", rect });
      return;
    }
    setMode({
      kind: "error",
      rect,
      message: "Couldn't save highlight.",
    });
  }, [book, chapter, create, mode, selectionAdapter]);

  const handleRemove = useCallback(
    async (id: number) => {
      const res = await remove(id);
      if (res.kind === "ok" || res.kind === "not_found") {
        setMode({ kind: "idle" });
      }
    },
    [remove],
  );

  return (
    <>
      <article
        ref={articleRef}
        className="passage"
        dangerouslySetInnerHTML={{ __html: html }}
      />
      <HighlightToolbar
        mode={mode}
        onHighlight={handleHighlight}
        onRemove={handleRemove}
        onSignin={onGuestSignin}
        onDismiss={() => setMode({ kind: "idle" })}
      />
    </>
  );
}
