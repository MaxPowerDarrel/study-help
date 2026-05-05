// SelectionAdapter is the platform abstraction for the browser
// Selection API (PROJECT_CONSTITUTION.md §4 — platform features behind
// an abstraction; specs/highlights.md Decisions, 2026-05-04).
//
// A future native shell can substitute its own implementation, and
// tests can inject a fake without touching the DOM.

export type SelectionInfo = {
  range: Range;
  rect: DOMRect;
};

export interface SelectionAdapter {
  // current returns the active selection if it's a non-empty range
  // contained within `container`, else null. iOS Safari occasionally
  // hands back a selection whose anchorNode is outside the container
  // mid-gesture; that case is treated as no-selection.
  current(container: Element): SelectionInfo | null;
  clear(): void;
}

class WindowSelectionAdapter implements SelectionAdapter {
  current(container: Element): SelectionInfo | null {
    const sel = window.getSelection();
    if (!sel || sel.rangeCount === 0) return null;
    if (sel.isCollapsed) return null;
    const range = sel.getRangeAt(0);
    if (!container.contains(range.startContainer)) return null;
    if (!container.contains(range.endContainer)) return null;
    const rect = range.getBoundingClientRect();
    if (rect.width === 0 && rect.height === 0) return null;
    return { range, rect };
  }
  clear(): void {
    const sel = window.getSelection();
    if (sel) sel.removeAllRanges();
  }
}

export const defaultSelectionAdapter: SelectionAdapter =
  new WindowSelectionAdapter();
