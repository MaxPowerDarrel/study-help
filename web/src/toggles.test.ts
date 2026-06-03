import { describe, expect, it } from "vitest";
import { DEFAULT_TOGGLES, togglesToQuery } from "./toggles";

describe("togglesToQuery", () => {
  it("sends include_cross_references", () => {
    const params = new URLSearchParams(
      togglesToQuery({ ...DEFAULT_TOGGLES, include_cross_references: true }),
    );
    expect(params.get("include_cross_references")).toBe("true");
  });

  it("defaults cross-references off", () => {
    expect(DEFAULT_TOGGLES.include_cross_references).toBe(false);
    const params = new URLSearchParams(togglesToQuery(DEFAULT_TOGGLES));
    expect(params.get("include_cross_references")).toBe("false");
  });

  it("does not send the client-only words-of-Christ toggle", () => {
    const params = new URLSearchParams(togglesToQuery(DEFAULT_TOGGLES));
    expect(params.has("include_word_of_christ")).toBe(false);
  });
});
