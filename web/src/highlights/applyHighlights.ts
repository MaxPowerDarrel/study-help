// applyHighlights wraps each highlight's character range in one or
// more <mark class="highlight" data-highlight-id="<id>"> spans inside
// the .passage container. Cross-verse highlights produce multiple
// <mark> elements that share the same data-highlight-id and are
// treated as one logical highlight. Idempotent: calling repeatedly
// with the same input yields the same DOM.

import type { Highlight } from "./api";
import { listVerseAnchors, tupleToRange } from "./parseSelection";

const HIGHLIGHT_ATTR = "data-highlight-id";

export function applyHighlights(
  container: Element,
  highlights: Highlight[],
): void {
  removeAllHighlights(container);
  if (highlights.length === 0) return;
  const anchors = listVerseAnchors(container);
  for (const h of highlights) {
    const range = tupleToRange(
      {
        start_verse: h.start_verse,
        start_offset: h.start_offset,
        end_verse: h.end_verse,
        end_offset: h.end_offset,
      },
      container,
      anchors,
    );
    if (!range) continue;
    wrapRange(range, h.id);
  }
}

export function removeAllHighlights(container: Element): void {
  const existing = container.querySelectorAll(
    `mark.highlight[${HIGHLIGHT_ATTR}]`,
  );
  existing.forEach(unwrap);
}

// findHighlightId returns the data-highlight-id of the nearest <mark>
// ancestor of `node`, or null if none.
export function findHighlightId(node: Node | null): number | null {
  let cur: Node | null = node;
  while (cur) {
    if (cur instanceof Element) {
      if (cur.tagName === "MARK" && cur.classList.contains("highlight")) {
        const raw = cur.getAttribute(HIGHLIGHT_ATTR);
        if (raw) {
          const n = Number(raw);
          if (Number.isFinite(n)) return n;
        }
      }
    }
    cur = cur.parentNode;
  }
  return null;
}

function wrapRange(range: Range, id: number): void {
  // Collect all text nodes the range intersects up front, so wrapping
  // (which mutates the DOM) doesn't affect iteration.
  const targets = collectIntersectingTextNodes(range);
  for (const t of targets) {
    const from = t.node === range.startContainer ? range.startOffset : 0;
    const to = t.node === range.endContainer ? range.endOffset : t.node.length;
    if (to <= from) continue;
    wrapTextSlice(t.node, from, to, id);
  }
}

type TextTarget = { node: Text };

function collectIntersectingTextNodes(range: Range): TextTarget[] {
  const root = range.commonAncestorContainer;
  const out: TextTarget[] = [];
  // commonAncestorContainer can itself be a Text node when start and
  // end live in the same text node.
  if (root.nodeType === Node.TEXT_NODE) {
    out.push({ node: root as Text });
    return out;
  }
  const walker = document.createTreeWalker(root, NodeFilter.SHOW_TEXT, {
    acceptNode: (node) =>
      range.intersectsNode(node)
        ? NodeFilter.FILTER_ACCEPT
        : NodeFilter.FILTER_REJECT,
  });
  let current = walker.nextNode();
  while (current) {
    out.push({ node: current as Text });
    current = walker.nextNode();
  }
  return out;
}

function wrapTextSlice(text: Text, from: number, to: number, id: number): void {
  // splitText returns the new node containing the tail. Splitting twice
  // gives us the middle slice.
  const middle = from === 0 ? text : text.splitText(from);
  if (to - from < middle.length) {
    middle.splitText(to - from);
  }
  const mark = document.createElement("mark");
  mark.className = "highlight";
  mark.setAttribute(HIGHLIGHT_ATTR, String(id));
  middle.parentNode?.insertBefore(mark, middle);
  mark.appendChild(middle);
}

function unwrap(el: Element): void {
  const parent = el.parentNode;
  if (!parent) return;
  while (el.firstChild) {
    parent.insertBefore(el.firstChild, el);
  }
  parent.removeChild(el);
  // Merge adjacent text nodes so subsequent offset math stays sane.
  parent.normalize();
}
