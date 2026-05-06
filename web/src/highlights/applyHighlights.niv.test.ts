import { beforeEach, describe, expect, it } from "vitest";
import { applyHighlights, removeAllHighlights } from "./applyHighlights";
import { listVerseAnchors, rangeToTuple, tupleToRange } from "./parseSelection";
import type { Highlight } from "./api";

// YouVersion's passage HTML emits an empty boundary span before each
// verse, plus a separate verse-label span:
//   <span class="yv-v" v="16"></span><span class="yv-vlbl">16</span>For God so loved...
// Source of truth: youversion/platform-sdk-react PR #152. The boundary
// span is what listYouVersionAnchors uses; offsets count the verse
// label as part of the verse body, but rangeToTuple/tupleToRange round-
// trip consistently because they share the same anchor.
function makePassage(): HTMLElement {
  const article = document.createElement("article");
  article.className = "passage";
  article.innerHTML = [
    `<span class="yv-v" v="16"></span>`,
    `<span class="yv-vlbl">16</span>`,
    `For God so loved the world.`,
    `<span class="yv-v" v="17"></span>`,
    `<span class="yv-vlbl">17</span>`,
    `For God did not send his Son to condemn.`,
  ].join("");
  document.body.appendChild(article);
  return article;
}

const baseHighlight: Omit<
  Highlight,
  "start_verse" | "start_offset" | "end_verse" | "end_offset" | "id"
> = {
  translation: "NIV",
  book: "John",
  chapter: 3,
  created_at: "2026-05-05T00:00:00Z",
};

beforeEach(() => {
  document.body.innerHTML = "";
});

describe("listVerseAnchors (NIV)", () => {
  it("collects verse boundary spans by verse number", () => {
    const article = makePassage();
    const anchors = listVerseAnchors(article, "NIV");
    expect(anchors.map((a) => a.verse)).toEqual([16, 17]);
    for (const a of anchors) {
      expect((a.anchor as Element).matches("span.yv-v[v]")).toBe(true);
    }
  });

  it("returns an empty list when the container has no NIV anchors", () => {
    const article = document.createElement("article");
    article.innerHTML = `<p>plain text without anchors</p>`;
    document.body.appendChild(article);
    expect(listVerseAnchors(article, "NIV")).toEqual([]);
  });

  it("dispatches independently from ESV (NIV anchors don't show up under ESV lookup)", () => {
    const article = makePassage();
    expect(listVerseAnchors(article, "ESV")).toEqual([]);
    expect(listVerseAnchors(article, "NIV")).toHaveLength(2);
  });
});

describe("rangeToTuple / tupleToRange round-trip (NIV)", () => {
  it("round-trips a single-verse range", () => {
    const article = makePassage();
    // Find the text node "For God so loved the world." and select
    // characters 0-3 ("For").
    const text = findTextStartingWith(article, "For God so loved");
    expect(text).not.toBeNull();
    const range = document.createRange();
    range.setStart(text!, 0);
    range.setEnd(text!, 3);

    const tuple = rangeToTuple(range, article, "NIV");
    expect(tuple).not.toBeNull();
    expect(tuple!.start_verse).toBe(16);
    expect(tuple!.end_verse).toBe(16);
    // Offset counts the verse label "16" before the verse text body
    // because the boundary anchor sits before yv-vlbl.
    expect(tuple!.end_offset - tuple!.start_offset).toBe(3);

    const round = tupleToRange(tuple!, article, "NIV");
    expect(round).not.toBeNull();
    expect(round!.toString()).toBe("For");
  });

  it("round-trips a cross-verse range", () => {
    const article = makePassage();
    const v16 = findTextStartingWith(article, "For God so loved");
    const v17 = findTextStartingWith(article, "For God did not");
    expect(v16).not.toBeNull();
    expect(v17).not.toBeNull();
    const range = document.createRange();
    range.setStart(v16!, "For God so ".length);
    range.setEnd(v17!, "For God ".length);

    const tuple = rangeToTuple(range, article, "NIV");
    expect(tuple).not.toBeNull();
    expect(tuple!.start_verse).toBe(16);
    expect(tuple!.end_verse).toBe(17);

    const round = tupleToRange(tuple!, article, "NIV");
    expect(round).not.toBeNull();
    expect(round!.toString()).toContain("loved the world.");
    expect(round!.toString()).toContain("For God ");
  });
});

describe("applyHighlights (NIV)", () => {
  it("wraps a single-verse range in one <mark>", () => {
    const article = makePassage();
    // Build a tuple via rangeToTuple so offsets line up with the NIV
    // anchor convention without hard-coding label/whitespace counts.
    const text = findTextStartingWith(article, "For God so loved");
    const range = document.createRange();
    range.setStart(text!, 0);
    range.setEnd(text!, "For God".length);
    const tuple = rangeToTuple(range, article, "NIV")!;

    const h: Highlight = {
      ...baseHighlight,
      id: 7,
      ...tuple,
    };
    applyHighlights(article, [h], "NIV");
    const marks = article.querySelectorAll("mark.highlight");
    expect(marks.length).toBe(1);
    expect(marks[0].getAttribute("data-highlight-id")).toBe("7");
    expect(marks[0].textContent).toBe("For God");
  });

  it("idempotent across re-applies", () => {
    const article = makePassage();
    const text = findTextStartingWith(article, "For God so loved");
    const range = document.createRange();
    range.setStart(text!, 0);
    range.setEnd(text!, "For".length);
    const tuple = rangeToTuple(range, article, "NIV")!;
    const h: Highlight = { ...baseHighlight, id: 1, ...tuple };

    applyHighlights(article, [h], "NIV");
    const first = article.innerHTML;
    applyHighlights(article, [h], "NIV");
    expect(article.innerHTML).toBe(first);

    removeAllHighlights(article);
    expect(article.querySelectorAll("mark.highlight").length).toBe(0);
  });
});

function findTextStartingWith(root: Element, prefix: string): Text | null {
  const walker = document.createTreeWalker(root, NodeFilter.SHOW_TEXT);
  let cur = walker.nextNode();
  while (cur) {
    if ((cur as Text).data.startsWith(prefix)) return cur as Text;
    cur = walker.nextNode();
  }
  return null;
}
