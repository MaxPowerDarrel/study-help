import { useEffect, useRef, useState } from "react";
import { fetchCrossref } from "./api";
import styles from "./PassageArticle.module.css";

type Props = {
  html: string;
  showWoc: boolean;
};

type Popover =
  | { state: "loading"; anchor: DOMRect }
  | { state: "ok"; anchor: DOMRect; html: string }
  | { state: "error"; anchor: DOMRect; message: string };

// PassageArticle renders a provider-rendered passage HTML blob and turns
// ESV cross-reference markers (<a class="cf">) into an inline popover
// showing the referenced verse text. Shared by the Read tab (App) and
// the Daily tab (DailyPanel) so the cross-ref behavior lives in one place.
//
// The marker's href is a bare reference string (ESV emits it that way —
// see internal/esv/client.go), not a navigable URL, so clicks are
// intercepted: we look the references up via /api/crossref and render
// them. NIV passages carry no .cf markers, so this is a no-op there.
export function PassageArticle({ html, showWoc }: Props) {
  const [popover, setPopover] = useState<Popover | null>(null);
  // Guards against a stale lookup landing after a newer click or a close.
  const reqId = useRef(0);

  const close = () => {
    reqId.current++;
    setPopover(null);
  };

  const onClick = (e: React.MouseEvent<HTMLElement>) => {
    const marker = (e.target as HTMLElement).closest<HTMLAnchorElement>("a.cf");
    if (!marker) return;
    e.preventDefault();
    const q = (marker.getAttribute("href") ?? "").replace(/\/+$/, "").trim();
    if (!q) return;
    const anchor = marker.getBoundingClientRect();
    const id = ++reqId.current;
    setPopover({ state: "loading", anchor });
    fetchCrossref(q).then((res) => {
      if (id !== reqId.current) return;
      if (res.kind === "ok") {
        setPopover({ state: "ok", anchor, html: res.html });
      } else {
        const message =
          res.kind === "rate_limited"
            ? "Service is busy, try again in a moment"
            : "Could not load cross-reference";
        setPopover({ state: "error", anchor, message });
      }
    });
  };

  // Dismiss on Escape, outside pointer-down, or any scroll (so the
  // popover never drifts away from its marker).
  useEffect(() => {
    if (!popover) return;
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") close();
    };
    const onPointerDown = (e: PointerEvent) => {
      const el = e.target as HTMLElement;
      if (!el.closest(`.${styles.popover}`)) close();
    };
    document.addEventListener("keydown", onKey);
    document.addEventListener("pointerdown", onPointerDown, true);
    window.addEventListener("scroll", close, true);
    return () => {
      document.removeEventListener("keydown", onKey);
      document.removeEventListener("pointerdown", onPointerDown, true);
      window.removeEventListener("scroll", close, true);
    };
  }, [popover]);

  return (
    <>
      <article
        className={showWoc ? "passage" : "passage no-woc"}
        dangerouslySetInnerHTML={{ __html: html }}
        onClick={onClick}
      />
      {popover && (
        <div
          className={styles.popover}
          role="dialog"
          aria-label="Cross reference"
          style={popoverPosition(popover.anchor)}
        >
          <button
            type="button"
            className={styles.close}
            aria-label="Close cross-reference"
            onClick={close}
          >
            ×
          </button>
          {popover.state === "loading" && (
            <div className={styles.status} aria-label="loading">
              Loading…
            </div>
          )}
          {popover.state === "error" && (
            <div className={styles.status} role="alert">
              {popover.message}
            </div>
          )}
          {popover.state === "ok" && (
            <div
              className={`passage ${styles.body}`}
              dangerouslySetInnerHTML={{ __html: popover.html }}
            />
          )}
        </div>
      )}
    </>
  );
}

// popoverPosition anchors the popover just below the marker, clamped to
// the viewport. Fixed positioning uses the marker's viewport-relative
// rect; the popover closes on scroll so it never separates from it.
function popoverPosition(anchor: DOMRect): React.CSSProperties {
  const margin = 8;
  const width = Math.min(360, window.innerWidth - margin * 2);
  const left = Math.max(
    margin,
    Math.min(anchor.left, window.innerWidth - width - margin),
  );
  return { position: "fixed", top: anchor.bottom + 6, left, width };
}
