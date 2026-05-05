// parseSelection converts a DOM Range inside the .passage container
// into a (start_verse, start_offset, end_verse, end_offset) tuple
// suitable for POST /api/highlights, using ESV's include-verse-anchors
// elements (e.g. <a name="v043003016">) as verse landmarks.
//
// Offsets are character positions within the textContent that follows
// each verse anchor up to (but not including) the next verse anchor.
// This matches the server-side range-storage convention in
// specs/highlights.md.

export type RangeTuple = {
  start_verse: number;
  start_offset: number;
  end_verse: number;
  end_offset: number;
};

// ESV's include-verse-anchors emits <a class="va" rel="vBCCCVVV">
// before each verse, where B is the book number (1-2 digits) and CCC,
// VVV are zero-padded chapter and verse. We only need the verse for
// offset math; book and chapter come from the URL on the read side.
//
// Reality is messier than the docs imply: most verses have multiple
// .va anchors (esv_01 in section headings, esv_06 in paragraph
// starts, esv_12 inside woc spans, esv_17 in continuation paragraphs).
// We dedupe by verse, keeping the LAST occurrence in document order
// — that's the one immediately preceding the verse text body.
const verseAnchorPattern = /^v\d+(\d{3})$/;

type AnchorEntry = { verse: number; anchor: Element };

export function listVerseAnchors(container: Element): AnchorEntry[] {
  const byVerse = new Map<number, Element>();
  container.querySelectorAll("a.va[rel]").forEach((a) => {
    const rel = a.getAttribute("rel") ?? "";
    const m = rel.match(verseAnchorPattern);
    if (!m) return;
    byVerse.set(parseInt(m[1], 10), a); // last write wins
  });
  return Array.from(byVerse.entries())
    .sort((a, b) => a[0] - b[0])
    .map(([verse, anchor]) => ({ verse, anchor }));
}

// rangeToTuple resolves a DOM Range into the verse/offset tuple.
// Returns null when the range doesn't fall under any verse anchor
// (e.g. the user selected a section heading or passage reference).
export function rangeToTuple(
  range: Range,
  container: Element,
  anchors: AnchorEntry[] = listVerseAnchors(container),
): RangeTuple | null {
  if (anchors.length === 0) return null;
  // Browsers sometimes hand back a Range whose start/end container is
  // an Element (with offset = a child index) instead of a Text node —
  // happens with triple-click, shift-click across element boundaries,
  // and some Safari touch gestures. Normalize to text-node boundaries
  // so the offset math has something to count from.
  const startBoundary = toTextBoundary(
    range.startContainer,
    range.startOffset,
    true,
  );
  const endBoundary = toTextBoundary(
    range.endContainer,
    range.endOffset,
    false,
  );
  if (!startBoundary || !endBoundary) return null;
  const start = locate(
    startBoundary.node,
    startBoundary.offset,
    container,
    anchors,
  );
  if (!start) return null;
  const end = locate(endBoundary.node, endBoundary.offset, container, anchors);
  if (!end) return null;
  return {
    start_verse: start.verse,
    start_offset: start.offset,
    end_verse: end.verse,
    end_offset: end.offset,
  };
}

// toTextBoundary normalizes (container, offset) to a (Text, offset)
// pair. If container is already a Text node, returns it unchanged.
// If it's an Element, looks up the relevant child boundary and walks
// into the first/last Text descendant. `forward=true` for range start,
// false for range end.
function toTextBoundary(
  container: Node,
  offset: number,
  forward: boolean,
): { node: Text; offset: number } | null {
  if (container.nodeType === Node.TEXT_NODE) {
    return { node: container as Text, offset };
  }
  if (container.nodeType !== Node.ELEMENT_NODE) return null;
  const children = container.childNodes;
  if (forward) {
    for (let i = offset; i < children.length; i++) {
      const t = firstTextDescendant(children[i]);
      if (t) return { node: t, offset: 0 };
    }
    for (let i = Math.min(offset, children.length) - 1; i >= 0; i--) {
      const t = lastTextDescendant(children[i]);
      if (t) return { node: t, offset: t.length };
    }
    return null;
  }
  for (let i = offset - 1; i >= 0; i--) {
    const t = lastTextDescendant(children[i]);
    if (t) return { node: t, offset: t.length };
  }
  for (let i = offset; i < children.length; i++) {
    const t = firstTextDescendant(children[i]);
    if (t) return { node: t, offset: 0 };
  }
  return null;
}

function firstTextDescendant(node: Node): Text | null {
  if (node.nodeType === Node.TEXT_NODE) return node as Text;
  if (node.nodeType !== Node.ELEMENT_NODE) return null;
  const walker = document.createTreeWalker(node, NodeFilter.SHOW_TEXT);
  return (walker.nextNode() as Text | null) ?? null;
}

function lastTextDescendant(node: Node): Text | null {
  if (node.nodeType === Node.TEXT_NODE) return node as Text;
  if (node.nodeType !== Node.ELEMENT_NODE) return null;
  const walker = document.createTreeWalker(node, NodeFilter.SHOW_TEXT);
  let last: Text | null = null;
  let current = walker.nextNode();
  while (current) {
    last = current as Text;
    current = walker.nextNode();
  }
  return last;
}

// tupleToRange is the inverse — used by applyHighlights to build a
// Range that can then be wrapped in <mark> spans.
export function tupleToRange(
  tuple: {
    start_verse: number;
    start_offset: number;
    end_verse: number;
    end_offset: number;
  },
  container: Element,
  anchors: AnchorEntry[] = listVerseAnchors(container),
): Range | null {
  const startAnchor = anchors.find((a) => a.verse === tuple.start_verse);
  const endAnchor = anchors.find((a) => a.verse === tuple.end_verse);
  if (!startAnchor || !endAnchor) return null;
  const nextAfterStart = anchorAfter(startAnchor, anchors);
  const nextAfterEnd = anchorAfter(endAnchor, anchors);
  const start = textNodeAtOffset(
    startAnchor.anchor,
    nextAfterStart?.anchor ?? null,
    tuple.start_offset,
    container,
  );
  const end = textNodeAtOffset(
    endAnchor.anchor,
    nextAfterEnd?.anchor ?? null,
    tuple.end_offset,
    container,
  );
  if (!start || !end) return null;
  const range = document.createRange();
  range.setStart(start.node, start.offset);
  range.setEnd(end.node, end.offset);
  return range;
}

function anchorAfter(
  cur: AnchorEntry,
  anchors: AnchorEntry[],
): AnchorEntry | null {
  const idx = anchors.indexOf(cur);
  if (idx < 0) return null;
  return anchors[idx + 1] ?? null;
}

function locate(
  node: Node,
  nodeOffset: number,
  container: Element,
  anchors: AnchorEntry[],
): { verse: number; offset: number } | null {
  // Find the verse anchor that comes at-or-before `node` in document order.
  let owning: AnchorEntry | null = null;
  for (const entry of anchors) {
    if (precedesOrIs(entry.anchor, node)) {
      owning = entry;
    } else {
      break;
    }
  }
  if (!owning) return null;
  const next = anchorAfter(owning, anchors);
  const offset = offsetWithin(
    owning.anchor,
    next?.anchor ?? null,
    node,
    nodeOffset,
    container,
  );
  if (offset < 0) return null;
  return { verse: owning.verse, offset };
}

function precedesOrIs(anchor: Element, node: Node): boolean {
  if (anchor === node) return true;
  const cmp = anchor.compareDocumentPosition(node);
  // Bit set when `node` follows `anchor`, OR when `node` is contained
  // by `anchor` (an empty-anchor case that doesn't normally happen
  // here but stays robust).
  return Boolean(
    cmp &
    (Node.DOCUMENT_POSITION_FOLLOWING | Node.DOCUMENT_POSITION_CONTAINED_BY),
  );
}

// offsetWithin walks text nodes from `from` (exclusive) toward `target`,
// summing textContent lengths until it reaches `target`, then adds
// `targetOffset`. Stops early if it reaches `until` (exclusive) without
// finding the target.
function offsetWithin(
  from: Element,
  until: Element | null,
  target: Node,
  targetOffset: number,
  container: Element,
): number {
  const walker = document.createTreeWalker(container, NodeFilter.SHOW_TEXT);
  let total = 0;
  let started = false;
  let current = walker.nextNode();
  while (current) {
    if (until && precedesOrIs(until, current)) {
      // Walked off the end of this verse without finding the target.
      return -1;
    }
    if (!started) {
      if (precedesOrIs(from, current) && current !== from) started = true;
    }
    if (started) {
      if (current === target) return total + targetOffset;
      total += current.textContent?.length ?? 0;
    }
    current = walker.nextNode();
  }
  return -1;
}

function textNodeAtOffset(
  from: Element,
  until: Element | null,
  offset: number,
  container: Element,
): { node: Text; offset: number } | null {
  const walker = document.createTreeWalker(container, NodeFilter.SHOW_TEXT);
  let started = false;
  let consumed = 0;
  let current = walker.nextNode();
  let lastNode: Text | null = null;
  while (current) {
    if (until && precedesOrIs(until, current)) break;
    if (!started) {
      if (precedesOrIs(from, current) && current !== from) started = true;
    }
    if (started) {
      const text = current as Text;
      const len = text.length;
      lastNode = text;
      if (offset <= consumed + len) {
        return { node: text, offset: offset - consumed };
      }
      consumed += len;
    }
    current = walker.nextNode();
  }
  // Allow exact-end placement at the tail of the last text node in this
  // verse (closed-end clamp keeps end-of-verse highlights renderable).
  if (lastNode && offset === consumed) {
    return { node: lastNode, offset: lastNode.length };
  }
  return null;
}
