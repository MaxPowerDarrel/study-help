import { beforeEach, describe, expect, it } from "vitest";
import {
  applyHighlights,
  findHighlightId,
  removeAllHighlights,
} from "./applyHighlights";
import type { Highlight } from "./api";

// Mirror the real ESV include-verse-anchors output:
// <a class="va" alt="esv_06" rel="vBCCCVVV"></a> precedes each verse.
// John = book 43. Verse 16 textContent: "For God so loved the world."
// (27 chars); verse 17: "For God did not send his Son to condemn."
// (40 chars). A separate test below exercises the more-realistic
// shape with a verse-num element preceding the verse text.
function makePassage(): HTMLElement {
  const article = document.createElement("article");
  article.className = "passage";
  article.innerHTML = [
    `<a class="va" alt="esv_06" rel="v43003016"></a>`,
    `For God so loved the world.`,
    `<a class="va" alt="esv_06" rel="v43003017"></a>`,
    `For God did not send his Son to condemn.`,
  ].join("");
  document.body.appendChild(article);
  return article;
}

const baseHighlight: Omit<
  Highlight,
  "start_verse" | "start_offset" | "end_verse" | "end_offset" | "id"
> = {
  book: "John",
  chapter: 3,
  created_at: "2026-05-04T00:00:00Z",
};

beforeEach(() => {
  document.body.innerHTML = "";
});

describe("applyHighlights", () => {
  it("wraps a single-verse range in one <mark> with the highlight id", () => {
    const article = makePassage();
    const h: Highlight = {
      ...baseHighlight,
      id: 7,
      start_verse: 16,
      start_offset: 0,
      end_verse: 16,
      end_offset: 11, // "For God so " (11 chars)
    };
    applyHighlights(article, [h]);
    const marks = article.querySelectorAll("mark.highlight");
    expect(marks.length).toBe(1);
    expect(marks[0].getAttribute("data-highlight-id")).toBe("7");
    expect(marks[0].textContent).toBe("For God so ");
  });

  it("produces multiple <mark> spans sharing one data-highlight-id for cross-verse ranges", () => {
    const article = makePassage();
    const h: Highlight = {
      ...baseHighlight,
      id: 9,
      start_verse: 16,
      start_offset: 11, // "loved the world." starts at offset 11 of v16
      end_verse: 17,
      end_offset: 12, // "For God did " is 12 chars from v17 start
    };
    applyHighlights(article, [h]);
    const marks = article.querySelectorAll("mark.highlight");
    expect(marks.length).toBeGreaterThan(1);
    for (const m of marks) {
      expect(m.getAttribute("data-highlight-id")).toBe("9");
    }
    const joined = Array.from(marks)
      .map((m) => m.textContent)
      .join("");
    expect(joined).toContain("loved the world.");
    expect(joined).toContain("For God did ");
  });

  it("is idempotent — applying twice yields the same DOM", () => {
    const article = makePassage();
    const h: Highlight = {
      ...baseHighlight,
      id: 1,
      start_verse: 16,
      start_offset: 0,
      end_verse: 16,
      end_offset: 7,
    };
    applyHighlights(article, [h]);
    const first = article.innerHTML;
    applyHighlights(article, [h]);
    expect(article.innerHTML).toBe(first);
  });

  it("handles real ESV layout: <b class='verse-num'>N </b> precedes verse text, anchor precedes verse-num", () => {
    // Real ESV: <b verse-num id="v..."> sits BEFORE the .va anchor for
    // some structural reasons; what matters for offsets is that the
    // verse text starts after the .va anchor.
    // Verse 16 textContent walked from .va onward = "For God so loved
    // the world." (the <b class="verse-num"> sits before .va).
    const article = document.createElement("article");
    article.className = "passage";
    article.innerHTML = [
      `<b class="verse-num" id="v43003016-1">16&nbsp;</b>`,
      `<a class="va" alt="esv_06" rel="v43003016"></a>`,
      `For God so loved the world.`,
    ].join("");
    document.body.appendChild(article);
    const h: Highlight = {
      ...baseHighlight,
      id: 5,
      start_verse: 16,
      start_offset: 0,
      end_verse: 16,
      end_offset: 11,
    };
    applyHighlights(article, [h]);
    const marks = article.querySelectorAll("mark.highlight");
    expect(marks.length).toBe(1);
    expect(marks[0].textContent).toBe("For God so ");
  });

  it("removes existing marks when called with an empty list", () => {
    const article = makePassage();
    const h: Highlight = {
      ...baseHighlight,
      id: 1,
      start_verse: 16,
      start_offset: 0,
      end_verse: 16,
      end_offset: 7,
    };
    applyHighlights(article, [h]);
    expect(article.querySelectorAll("mark.highlight").length).toBeGreaterThan(
      0,
    );
    applyHighlights(article, []);
    expect(article.querySelectorAll("mark.highlight").length).toBe(0);
  });
});

describe("removeAllHighlights", () => {
  it("strips marks and merges adjacent text", () => {
    const article = makePassage();
    applyHighlights(article, [
      {
        ...baseHighlight,
        id: 3,
        start_verse: 16,
        start_offset: 4,
        end_verse: 16,
        end_offset: 7,
      },
    ]);
    expect(article.querySelectorAll("mark.highlight").length).toBe(1);
    removeAllHighlights(article);
    expect(article.querySelectorAll("mark.highlight").length).toBe(0);
    // Verse text should be intact.
    expect(article.textContent).toContain("For God so loved the world.");
  });
});

describe("findHighlightId", () => {
  it("returns the id of the nearest <mark> ancestor", () => {
    const article = makePassage();
    applyHighlights(article, [
      {
        ...baseHighlight,
        id: 42,
        start_verse: 16,
        start_offset: 0,
        end_verse: 16,
        end_offset: 5,
      },
    ]);
    const mark = article.querySelector("mark.highlight")!;
    expect(findHighlightId(mark.firstChild)).toBe(42);
    expect(findHighlightId(mark)).toBe(42);
  });
  it("returns null for nodes outside any mark", () => {
    const article = makePassage();
    expect(findHighlightId(article.firstChild)).toBeNull();
  });
});
