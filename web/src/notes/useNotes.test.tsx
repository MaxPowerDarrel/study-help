import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { act, renderHook, waitFor } from "@testing-library/react";
import { useNotes } from "./useNotes";
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

const sample: Note = {
  id: 1,
  translation: "ESV",
  book: "John",
  chapter: 3,
  start_verse: 16,
  start_offset: 0,
  end_verse: 16,
  end_offset: 10,
  body: "for God so loved",
  created_at: "2026-05-04T00:00:00Z",
  updated_at: "2026-05-04T00:00:00Z",
};

describe("useNotes", () => {
  it("does not fetch when disabled (guest)", async () => {
    const fetchMock = globalThis.fetch as ReturnType<typeof vi.fn>;
    const { result } = renderHook(() => useNotes("John", 3, "ESV", false));
    await waitFor(() => expect(result.current.loading).toBe(false));
    expect(fetchMock).not.toHaveBeenCalled();
    expect(result.current.notes).toEqual([]);
  });

  it("fetches on mount when enabled", async () => {
    const fetchMock = globalThis.fetch as ReturnType<typeof vi.fn>;
    fetchMock.mockResolvedValueOnce(okJSON([sample]));
    const { result } = renderHook(() => useNotes("John", 3, "ESV", true));
    await waitFor(() => expect(result.current.notes.length).toBe(1));
    expect(fetchMock).toHaveBeenCalledOnce();
    expect(fetchMock.mock.calls[0][0]).toContain("/api/notes?book=John");
  });

  it("re-fetches when (book, chapter) changes", async () => {
    const fetchMock = globalThis.fetch as ReturnType<typeof vi.fn>;
    fetchMock.mockResolvedValue(okJSON([]));
    const { rerender } = renderHook(
      ({ book, chapter }) => useNotes(book, chapter, "ESV", true),
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
    fetchMock.mockResolvedValueOnce(okJSON(sample, 201)); // create
    fetchMock.mockResolvedValueOnce(okJSON([sample])); // re-fetch

    const { result } = renderHook(() => useNotes("John", 3, "ESV", true));
    await waitFor(() => expect(result.current.loading).toBe(false));

    await act(async () => {
      const r = await result.current.create({
        book: "John",
        chapter: 3,
        start_verse: 16,
        start_offset: 0,
        end_verse: 16,
        end_offset: 10,
        body: "x",
      });
      expect(r.kind).toBe("ok");
    });

    await waitFor(() => expect(result.current.notes.length).toBe(1));
    expect(fetchMock).toHaveBeenCalledTimes(3);
  });

  it("re-fetches after a successful update", async () => {
    const fetchMock = globalThis.fetch as ReturnType<typeof vi.fn>;
    fetchMock.mockResolvedValueOnce(okJSON([sample])); // initial list
    fetchMock.mockResolvedValueOnce(
      okJSON({ ...sample, body: "edited", updated_at: "2026-05-05T00:00:00Z" }),
    );
    fetchMock.mockResolvedValueOnce(
      okJSON([
        { ...sample, body: "edited", updated_at: "2026-05-05T00:00:00Z" },
      ]),
    );

    const { result } = renderHook(() => useNotes("John", 3, "ESV", true));
    await waitFor(() => expect(result.current.notes.length).toBe(1));

    await act(async () => {
      const r = await result.current.update(1, "edited");
      expect(r.kind).toBe("ok");
    });

    await waitFor(() => expect(result.current.notes[0]?.body).toBe("edited"));
    expect(fetchMock).toHaveBeenCalledTimes(3);
  });

  it("does not re-fetch after a 400 invalid on create", async () => {
    const fetchMock = globalThis.fetch as ReturnType<typeof vi.fn>;
    fetchMock.mockResolvedValueOnce(okJSON([]));
    fetchMock.mockResolvedValueOnce(new Response("", { status: 400 }));

    const { result } = renderHook(() => useNotes("John", 3, "ESV", true));
    await waitFor(() => expect(result.current.loading).toBe(false));

    await act(async () => {
      const r = await result.current.create({
        book: "John",
        chapter: 3,
        start_verse: 16,
        start_offset: 0,
        end_verse: 16,
        end_offset: 10,
        body: "x",
      });
      expect(r.kind).toBe("invalid");
    });

    expect(fetchMock).toHaveBeenCalledTimes(2);
  });
});
