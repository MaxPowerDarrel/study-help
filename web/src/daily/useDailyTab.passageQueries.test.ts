import { describe, expect, it } from "vitest";
import { passageQueries } from "./useDailyTab";

describe("passageQueries", () => {
  it("returns one query for a hyphen range", () => {
    expect(
      passageQueries({ book: "Genesis", chapters: "1-3", testament: "OT" }),
    ).toEqual(["Genesis 1-3"]);
  });

  it("returns one query for a single chapter", () => {
    expect(
      passageQueries({ book: "Obadiah", chapters: "1", testament: "OT" }),
    ).toEqual(["Obadiah 1"]);
  });

  it("fans out comma-separated chapters into multiple queries", () => {
    // Hope plan emits cells like "Matthew 11,12"; canon validation on
    // the server only accepts single chapters or hyphen ranges per
    // call, so the SPA splits on the comma.
    expect(
      passageQueries({ book: "Matthew", chapters: "11,12", testament: "NT" }),
    ).toEqual(["Matthew 11", "Matthew 12"]);
  });

  it("preserves verse-range refs (the provider handles them)", () => {
    expect(
      passageQueries({
        book: "Psalm",
        chapters: "119:33-40",
        testament: "OT",
      }),
    ).toEqual(["Psalm 119:33-40"]);
  });

  it("returns no queries for an empty chapters string", () => {
    expect(passageQueries({ book: "", chapters: "", testament: "" })).toEqual(
      [],
    );
  });
});
