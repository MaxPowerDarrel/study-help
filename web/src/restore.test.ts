import { renderHook, act } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { ToggleStore } from "./platform/ToggleStore";
import {
  useStoredTab,
  useStoredReadRef,
  useStoredDailyDate,
  readStoredTab,
  readStoredReadRef,
  readStoredDailyDate,
} from "./restore";

class MemoryStore implements ToggleStore {
  private data = new Map<string, string>();
  private listeners = new Set<() => void>();
  get(key: string): string | null {
    return this.data.get(key) ?? null;
  }
  set(key: string, value: string): void {
    this.data.set(key, value);
    for (const l of this.listeners) l();
  }
  remove(key: string): void {
    this.data.delete(key);
    for (const l of this.listeners) l();
  }
  subscribe(l: () => void): () => void {
    this.listeners.add(l);
    return () => this.listeners.delete(l);
  }
}

describe("useStoredTab", () => {
  it("defaults to 'read' when storage is empty", () => {
    const store = new MemoryStore();
    const { result } = renderHook(() => useStoredTab(store));
    expect(result.current[0]).toBe("read");
  });

  it("hydrates from storage", () => {
    const store = new MemoryStore();
    store.set("reader.active-tab", "daily");
    const { result } = renderHook(() => useStoredTab(store));
    expect(result.current[0]).toBe("daily");
  });

  it("ignores an unknown stored value", () => {
    const store = new MemoryStore();
    store.set("reader.active-tab", "settings");
    expect(readStoredTab(store)).toBeNull();
  });

  it("setTab writes to storage and updates state", () => {
    const store = new MemoryStore();
    const { result } = renderHook(() => useStoredTab(store));
    act(() => {
      result.current[1]("daily");
    });
    expect(result.current[0]).toBe("daily");
    expect(store.get("reader.active-tab")).toBe("daily");
  });
});

describe("useStoredReadRef", () => {
  it("defaults to John 3 when storage is empty", () => {
    const store = new MemoryStore();
    const { result } = renderHook(() => useStoredReadRef(store));
    expect(result.current[0]).toEqual({ bookIndex: 42, chapter: 3 });
  });

  it("hydrates a valid stored ref", () => {
    const store = new MemoryStore();
    store.set(
      "reader.read-location",
      JSON.stringify({ bookIndex: 44, chapter: 8 }),
    );
    const { result } = renderHook(() => useStoredReadRef(store));
    expect(result.current[0]).toEqual({ bookIndex: 44, chapter: 8 });
  });

  it("falls back on malformed JSON", () => {
    const store = new MemoryStore();
    store.set("reader.read-location", "not json");
    expect(readStoredReadRef(store)).toBeNull();
  });

  it("falls back when bookIndex is out of range", () => {
    const store = new MemoryStore();
    store.set(
      "reader.read-location",
      JSON.stringify({ bookIndex: 999, chapter: 1 }),
    );
    expect(readStoredReadRef(store)).toBeNull();
  });

  it("falls back when chapter exceeds the book's chapter count", () => {
    const store = new MemoryStore();
    // John has 21 chapters
    store.set(
      "reader.read-location",
      JSON.stringify({ bookIndex: 42, chapter: 99 }),
    );
    expect(readStoredReadRef(store)).toBeNull();
  });

  it("falls back when chapter is below 1", () => {
    const store = new MemoryStore();
    store.set(
      "reader.read-location",
      JSON.stringify({ bookIndex: 0, chapter: 0 }),
    );
    expect(readStoredReadRef(store)).toBeNull();
  });

  it("falls back when shape is wrong", () => {
    const store = new MemoryStore();
    store.set("reader.read-location", JSON.stringify({ foo: "bar" }));
    expect(readStoredReadRef(store)).toBeNull();
  });

  it("setRef writes to storage and updates state", () => {
    const store = new MemoryStore();
    const { result } = renderHook(() => useStoredReadRef(store));
    act(() => {
      result.current[1]({ bookIndex: 44, chapter: 8 });
    });
    expect(result.current[0]).toEqual({ bookIndex: 44, chapter: 8 });
    expect(JSON.parse(store.get("reader.read-location") ?? "{}")).toEqual({
      bookIndex: 44,
      chapter: 8,
    });
  });
});

describe("useStoredDailyDate", () => {
  it("defaults to today when storage is empty", () => {
    const store = new MemoryStore();
    const { result } = renderHook(() =>
      useStoredDailyDate("2026-05-10", store),
    );
    expect(result.current[0]).toBe("2026-05-10");
  });

  it("hydrates the stored date when savedOn matches today", () => {
    const store = new MemoryStore();
    store.set(
      "reader.daily-date",
      JSON.stringify({ date: "2026-05-08", savedOn: "2026-05-10" }),
    );
    const { result } = renderHook(() =>
      useStoredDailyDate("2026-05-10", store),
    );
    expect(result.current[0]).toBe("2026-05-08");
  });

  it("snaps to today when savedOn is a previous calendar day", () => {
    const store = new MemoryStore();
    store.set(
      "reader.daily-date",
      JSON.stringify({ date: "2026-05-08", savedOn: "2026-05-09" }),
    );
    const { result } = renderHook(() =>
      useStoredDailyDate("2026-05-10", store),
    );
    expect(result.current[0]).toBe("2026-05-10");
  });

  it("falls back on malformed JSON", () => {
    const store = new MemoryStore();
    store.set("reader.daily-date", "not json");
    expect(readStoredDailyDate("2026-05-10", store)).toBeNull();
  });

  it("falls back when shape is wrong", () => {
    const store = new MemoryStore();
    store.set(
      "reader.daily-date",
      JSON.stringify({ date: "2026-05-10" }), // missing savedOn
    );
    expect(readStoredDailyDate("2026-05-10", store)).toBeNull();
  });

  it("setDate writes date + today as savedOn and updates state", () => {
    const store = new MemoryStore();
    const { result } = renderHook(() =>
      useStoredDailyDate("2026-05-10", store),
    );
    act(() => {
      result.current[1]("2026-05-08");
    });
    expect(result.current[0]).toBe("2026-05-08");
    expect(JSON.parse(store.get("reader.daily-date") ?? "{}")).toEqual({
      date: "2026-05-08",
      savedOn: "2026-05-10",
    });
  });

  it("setDate then re-mount on the same day rehydrates the saved date", () => {
    const store = new MemoryStore();
    const { result } = renderHook(() =>
      useStoredDailyDate("2026-05-10", store),
    );
    act(() => {
      result.current[1]("2026-05-08");
    });
    const { result: result2 } = renderHook(() =>
      useStoredDailyDate("2026-05-10", store),
    );
    expect(result2.current[0]).toBe("2026-05-08");
  });

  it("setDate then re-mount on a later day snaps to the new today", () => {
    const store = new MemoryStore();
    const { result } = renderHook(() =>
      useStoredDailyDate("2026-05-10", store),
    );
    act(() => {
      result.current[1]("2026-05-08");
    });
    const { result: result2 } = renderHook(() =>
      useStoredDailyDate("2026-05-11", store),
    );
    expect(result2.current[0]).toBe("2026-05-11");
  });
});
