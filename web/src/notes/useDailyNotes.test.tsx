import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { act, renderHook, waitFor } from "@testing-library/react";
import { useDailyNotes } from "./useDailyNotes";
import type { Note } from "./api";

beforeEach(() => {
  vi.stubGlobal("fetch", vi.fn());
});

afterEach(() => {
  vi.unstubAllGlobals();
});

const okJSON = (body: unknown, status = 200) =>
  new Response(JSON.stringify(body), {
    status,
    headers: { "Content-Type": "application/json" },
  });

const note = (over: Partial<Note>): Note => ({
  id: 1,
  translation: "ESV",
  book: "Genesis",
  chapter: 1,
  start_verse: 1,
  start_offset: 0,
  end_verse: 1,
  end_offset: 5,
  body: "x",
  created_at: "2026-05-05T00:00:00Z",
  updated_at: "2026-05-05T00:00:00Z",
  ...over,
});

describe("useDailyNotes", () => {
  it("does not fetch when disabled", async () => {
    const fetchMock = globalThis.fetch as ReturnType<typeof vi.fn>;
    const { result } = renderHook(() =>
      useDailyNotes("Genesis", [1, 2], "ESV", false),
    );
    await waitFor(() => expect(result.current.loading).toBe(false));
    expect(fetchMock).not.toHaveBeenCalled();
    expect(result.current.notes).toEqual([]);
  });

  it("fans out one listNotes call per chapter", async () => {
    const fetchMock = globalThis.fetch as ReturnType<typeof vi.fn>;
    fetchMock
      .mockResolvedValueOnce(okJSON([note({ id: 10, chapter: 1 })]))
      .mockResolvedValueOnce(okJSON([note({ id: 20, chapter: 2 })]))
      .mockResolvedValueOnce(okJSON([note({ id: 30, chapter: 3 })]));
    const { result } = renderHook(() =>
      useDailyNotes("Genesis", [1, 2, 3], "ESV", true),
    );
    await waitFor(() => expect(result.current.notes.length).toBe(3));
    expect(fetchMock).toHaveBeenCalledTimes(3);
  });

  it("aggregates and sorts by (chapter, created_at)", async () => {
    const fetchMock = globalThis.fetch as ReturnType<typeof vi.fn>;
    fetchMock
      .mockResolvedValueOnce(
        okJSON([
          note({
            id: 11,
            chapter: 1,
            created_at: "2026-05-05T00:00:02Z",
          }),
          note({
            id: 12,
            chapter: 1,
            created_at: "2026-05-05T00:00:01Z",
          }),
        ]),
      )
      .mockResolvedValueOnce(okJSON([note({ id: 21, chapter: 2 })]));
    const { result } = renderHook(() =>
      useDailyNotes("Genesis", [1, 2], "ESV", true),
    );
    await waitFor(() => expect(result.current.notes.length).toBe(3));
    expect(result.current.notes.map((n) => n.id)).toEqual([12, 11, 21]);
  });

  it("re-fetches all chapters after a successful create", async () => {
    const fetchMock = globalThis.fetch as ReturnType<typeof vi.fn>;
    fetchMock
      .mockResolvedValueOnce(okJSON([])) // ch 1 initial
      .mockResolvedValueOnce(okJSON([])) // ch 2 initial
      .mockResolvedValueOnce(okJSON(note({ id: 100, chapter: 1 }), 201)) // create
      .mockResolvedValueOnce(okJSON([note({ id: 100, chapter: 1 })])) // ch 1 refresh
      .mockResolvedValueOnce(okJSON([])); // ch 2 refresh

    const { result } = renderHook(() =>
      useDailyNotes("Genesis", [1, 2], "ESV", true),
    );
    await waitFor(() => expect(result.current.loading).toBe(false));

    await act(async () => {
      const r = await result.current.create({
        book: "Genesis",
        chapter: 1,
        start_verse: 1,
        start_offset: 0,
        end_verse: 1,
        end_offset: 5,
        body: "x",
      });
      expect(r.kind).toBe("ok");
    });

    await waitFor(() => expect(result.current.notes.length).toBe(1));
    expect(fetchMock).toHaveBeenCalledTimes(5);
  });

  it("clears notes when book becomes null", async () => {
    const fetchMock = globalThis.fetch as ReturnType<typeof vi.fn>;
    fetchMock.mockResolvedValueOnce(okJSON([note({ id: 1 })]));
    const { result, rerender } = renderHook(
      ({ book }: { book: string | null }) =>
        useDailyNotes(book, [1], "ESV", true),
      { initialProps: { book: "Genesis" as string | null } },
    );
    await waitFor(() => expect(result.current.notes.length).toBe(1));
    rerender({ book: null });
    await waitFor(() => expect(result.current.notes).toEqual([]));
  });
});
