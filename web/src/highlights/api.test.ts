import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import {
  createHighlight,
  deleteHighlight,
  listHighlights,
  type Highlight,
} from "./api";

beforeEach(() => {
  vi.stubGlobal("fetch", vi.fn());
});

afterEach(() => {
  vi.unstubAllGlobals();
});

const okJSON = (body: unknown) =>
  new Response(JSON.stringify(body), {
    status: 200,
    headers: { "Content-Type": "application/json" },
  });
// 204/304 must have a null body per the Fetch spec.
const status = (code: number) =>
  new Response(code === 204 || code === 304 ? null : "", { status: code });

const sample: Highlight = {
  id: 1,
  translation: "ESV",
  book: "John",
  chapter: 3,
  start_verse: 16,
  start_offset: 0,
  end_verse: 16,
  end_offset: 10,
  created_at: "2026-05-04T00:00:00Z",
};

describe("listHighlights", () => {
  it("returns highlights on 200", async () => {
    (globalThis.fetch as ReturnType<typeof vi.fn>).mockResolvedValueOnce(
      okJSON([sample]),
    );
    const r = await listHighlights("John", 3, "ESV");
    expect(r).toEqual({ kind: "ok", highlights: [sample] });
  });
  it("maps 401 to guest", async () => {
    (globalThis.fetch as ReturnType<typeof vi.fn>).mockResolvedValueOnce(
      status(401),
    );
    const r = await listHighlights("John", 3, "ESV");
    expect(r.kind).toBe("guest");
  });
  it("maps network failure to error", async () => {
    (globalThis.fetch as ReturnType<typeof vi.fn>).mockRejectedValueOnce(
      new Error("nope"),
    );
    const r = await listHighlights("John", 3, "ESV");
    expect(r.kind).toBe("error");
  });
});

describe("createHighlight", () => {
  const input = {
    book: "John",
    chapter: 3,
    start_verse: 16,
    start_char_offset: 0,
    end_verse: 16,
    end_char_offset: 10,
  };
  it("returns highlight on 201", async () => {
    (globalThis.fetch as ReturnType<typeof vi.fn>).mockResolvedValueOnce(
      new Response(JSON.stringify(sample), {
        status: 201,
        headers: { "Content-Type": "application/json" },
      }),
    );
    const r = await createHighlight(input, "ESV");
    expect(r).toEqual({ kind: "ok", highlight: sample });
  });
  it("maps 409 to overlap", async () => {
    (globalThis.fetch as ReturnType<typeof vi.fn>).mockResolvedValueOnce(
      status(409),
    );
    const r = await createHighlight(input, "ESV");
    expect(r.kind).toBe("overlap");
  });
  it("maps 400 to invalid", async () => {
    (globalThis.fetch as ReturnType<typeof vi.fn>).mockResolvedValueOnce(
      status(400),
    );
    const r = await createHighlight(input, "ESV");
    expect(r.kind).toBe("invalid");
  });
  it("maps 401 to guest", async () => {
    (globalThis.fetch as ReturnType<typeof vi.fn>).mockResolvedValueOnce(
      status(401),
    );
    const r = await createHighlight(input, "ESV");
    expect(r.kind).toBe("guest");
  });
});

describe("deleteHighlight", () => {
  it("returns ok on 204", async () => {
    (globalThis.fetch as ReturnType<typeof vi.fn>).mockResolvedValueOnce(
      status(204),
    );
    const r = await deleteHighlight(1);
    expect(r.kind).toBe("ok");
  });
  it("maps 404 to not_found", async () => {
    (globalThis.fetch as ReturnType<typeof vi.fn>).mockResolvedValueOnce(
      status(404),
    );
    const r = await deleteHighlight(1);
    expect(r.kind).toBe("not_found");
  });
  it("maps 401 to guest", async () => {
    (globalThis.fetch as ReturnType<typeof vi.fn>).mockResolvedValueOnce(
      status(401),
    );
    const r = await deleteHighlight(1);
    expect(r.kind).toBe("guest");
  });
});
