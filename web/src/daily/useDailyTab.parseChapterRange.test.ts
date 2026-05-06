import { describe, expect, it } from "vitest";
import { parseChapterRange } from "./useDailyTab";

describe("parseChapterRange", () => {
  it("returns a single chapter for a bare number", () => {
    expect(parseChapterRange("119")).toEqual([119]);
  });

  it("expands hyphen ranges", () => {
    expect(parseChapterRange("1-3")).toEqual([1, 2, 3]);
  });

  it("handles comma-separated chapters", () => {
    expect(parseChapterRange("1,3")).toEqual([1, 3]);
  });

  it("collapses verse-range refs to the chapter", () => {
    // Hope-plan Psalms come through with a chapter+verse spec like
    // "119:33-40"; the daily panel renders chapter blocks, so the
    // verse range collapses to the bare chapter for fetching.
    expect(parseChapterRange("119:33-40")).toEqual([119]);
    expect(parseChapterRange("18:1-15")).toEqual([18]);
  });

  it("ignores garbage", () => {
    expect(parseChapterRange("")).toEqual([]);
    expect(parseChapterRange("abc")).toEqual([]);
  });
});
