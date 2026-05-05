import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { act, renderHook, waitFor } from "@testing-library/react";
import { useHighlights } from "./useHighlights";
import type { Highlight } from "./api";

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

const sample: Highlight = {
  id: 1,
  book: "John",
  chapter: 3,
  start_verse: 16,
  start_offset: 0,
  end_verse: 16,
  end_offset: 10,
  created_at: "2026-05-04T00:00:00Z",
};

describe("useHighlights", () => {
  it("does not fetch when disabled (guest)", async () => {
    const fetchMock = globalThis.fetch as ReturnType<typeof vi.fn>;
    const { result } = renderHook(() => useHighlights("John", 3, false));
    await waitFor(() => expect(result.current.loading).toBe(false));
    expect(fetchMock).not.toHaveBeenCalled();
    expect(result.current.highlights).toEqual([]);
  });

  it("fetches on mount when enabled", async () => {
    const fetchMock = globalThis.fetch as ReturnType<typeof vi.fn>;
    fetchMock.mockResolvedValueOnce(okJSON([sample]));
    const { result } = renderHook(() => useHighlights("John", 3, true));
    await waitFor(() => expect(result.current.highlights.length).toBe(1));
    expect(fetchMock).toHaveBeenCalledOnce();
    expect(fetchMock.mock.calls[0][0]).toContain("/api/highlights?book=John");
  });

  it("re-fetches when (book, chapter) changes", async () => {
    const fetchMock = globalThis.fetch as ReturnType<typeof vi.fn>;
    fetchMock.mockResolvedValue(okJSON([]));
    const { rerender } = renderHook(
      ({ book, chapter }) => useHighlights(book, chapter, true),
      { initialProps: { book: "John", chapter: 3 } },
    );
    await waitFor(() => expect(fetchMock).toHaveBeenCalledOnce());
    rerender({ book: "John", chapter: 4 });
    await waitFor(() => expect(fetchMock).toHaveBeenCalledTimes(2));
    expect(fetchMock.mock.calls[1][0]).toContain("chapter=4");
  });

  it("re-fetches after a successful create", async () => {
    const fetchMock = globalThis.fetch as ReturnType<typeof vi.fn>;
    fetchMock.mockResolvedValueOnce(okJSON([])); // initial list
    fetchMock.mockResolvedValueOnce(
      new Response(JSON.stringify(sample), {
        status: 201,
        headers: { "Content-Type": "application/json" },
      }),
    ); // create
    fetchMock.mockResolvedValueOnce(okJSON([sample])); // re-fetch

    const { result } = renderHook(() => useHighlights("John", 3, true));
    await waitFor(() => expect(result.current.loading).toBe(false));

    await act(async () => {
      const r = await result.current.create({
        book: "John",
        chapter: 3,
        start_verse: 16,
        start_char_offset: 0,
        end_verse: 16,
        end_char_offset: 10,
      });
      expect(r.kind).toBe("ok");
    });

    await waitFor(() => expect(result.current.highlights.length).toBe(1));
    expect(fetchMock).toHaveBeenCalledTimes(3);
  });

  it("does not re-fetch after a 409 overlap on create", async () => {
    const fetchMock = globalThis.fetch as ReturnType<typeof vi.fn>;
    fetchMock.mockResolvedValueOnce(okJSON([])); // initial list
    fetchMock.mockResolvedValueOnce(new Response("", { status: 409 })); // create overlap

    const { result } = renderHook(() => useHighlights("John", 3, true));
    await waitFor(() => expect(result.current.loading).toBe(false));

    await act(async () => {
      const r = await result.current.create({
        book: "John",
        chapter: 3,
        start_verse: 16,
        start_char_offset: 0,
        end_verse: 16,
        end_char_offset: 10,
      });
      expect(r.kind).toBe("overlap");
    });

    expect(fetchMock).toHaveBeenCalledTimes(2); // initial list + create only
  });
});
