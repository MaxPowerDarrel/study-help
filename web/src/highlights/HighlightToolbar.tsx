import { useEffect, useRef } from "react";
import styles from "./HighlightToolbar.module.css";

// The verse-tuple resolved at the moment the selection completes —
// stored as primitives so that DOM mutations between capture and use
// (toolbar render, applyHighlights running, React re-mount) can't
// corrupt it the way a live Range would.
export type ToolbarTuple = {
  start_verse: number;
  start_offset: number;
  end_verse: number;
  end_offset: number;
};

export type ToolbarMode =
  | { kind: "idle" }
  | { kind: "selection"; rect: DOMRect; tuple: ToolbarTuple }
  | { kind: "existing"; rect: DOMRect; highlightId: number }
  | { kind: "guest"; rect: DOMRect }
  | { kind: "error"; rect: DOMRect; message: string };

type Props = {
  mode: ToolbarMode;
  onHighlight: () => void;
  onRemove: (id: number) => void;
  onSignin: () => void;
  onDismiss: () => void;
};

const TOOLBAR_HEIGHT = 44;
const GAP = 8;

export function HighlightToolbar({
  mode,
  onHighlight,
  onRemove,
  onSignin,
  onDismiss,
}: Props) {
  const ref = useRef<HTMLDivElement | null>(null);

  useEffect(() => {
    if (mode.kind === "idle") return;
    function handleKey(e: KeyboardEvent) {
      if (e.key === "Escape") onDismiss();
    }
    function handleDown(e: MouseEvent | TouchEvent) {
      const target = e.target as Node | null;
      if (target && ref.current && ref.current.contains(target)) return;
      onDismiss();
    }
    window.addEventListener("keydown", handleKey);
    document.addEventListener("mousedown", handleDown);
    document.addEventListener("touchstart", handleDown);
    return () => {
      window.removeEventListener("keydown", handleKey);
      document.removeEventListener("mousedown", handleDown);
      document.removeEventListener("touchstart", handleDown);
    };
  }, [mode.kind, onDismiss]);

  if (mode.kind === "idle") return null;

  const pos = computePosition(mode.rect);

  return (
    <div
      ref={ref}
      className={styles.toolbar}
      style={{ top: pos.top, left: pos.left }}
      role="toolbar"
      aria-label="Highlight"
      // preventDefault on mousedown so clicking the toolbar doesn't
      // clear the user's text selection in the article — without this,
      // by the time the button's onClick fires the selection is gone.
      onMouseDown={(e) => e.preventDefault()}
    >
      {mode.kind === "selection" && (
        <button type="button" className={styles.button} onClick={onHighlight}>
          Highlight
        </button>
      )}
      {mode.kind === "existing" && (
        <button
          type="button"
          className={styles.button}
          onClick={() => onRemove(mode.highlightId)}
        >
          Remove highlight
        </button>
      )}
      {mode.kind === "guest" && (
        <button type="button" className={styles.button} onClick={onSignin}>
          Sign in to highlight
        </button>
      )}
      {mode.kind === "error" && (
        <span className={styles.error} role="alert">
          {mode.message}
        </span>
      )}
    </div>
  );
}

function computePosition(rect: DOMRect): { top: number; left: number } {
  const above = rect.top - TOOLBAR_HEIGHT - GAP;
  const top = above >= 0 ? above : rect.bottom + GAP;
  const left = Math.max(GAP, rect.left);
  return { top, left };
}
