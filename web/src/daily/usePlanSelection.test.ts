import { renderHook, act } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { ToggleStore } from "../platform/ToggleStore";
import { DEFAULT_PLAN_IDS } from "./plans";
import { usePlanSelection } from "./usePlanSelection";

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

describe("usePlanSelection", () => {
  it("defaults to bible-year when storage is empty", () => {
    const store = new MemoryStore();
    const { result } = renderHook(() => usePlanSelection(store));
    expect(result.current[0]).toEqual(DEFAULT_PLAN_IDS);
  });

  it("persists a multi-plan selection round-trip", () => {
    const store = new MemoryStore();
    const { result } = renderHook(() => usePlanSelection(store));
    act(() => {
      result.current[1](["bible-year", "hope"]);
    });
    expect(result.current[0]).toEqual(["bible-year", "hope"]);

    // New hook instance picks up the persisted value.
    const { result: result2 } = renderHook(() => usePlanSelection(store));
    expect(result2.current[0]).toEqual(["bible-year", "hope"]);
  });

  it("coerces empty selection to default", () => {
    const store = new MemoryStore();
    const { result } = renderHook(() => usePlanSelection(store));
    act(() => {
      result.current[1]([]);
    });
    expect(result.current[0]).toEqual(DEFAULT_PLAN_IDS);
  });

  it("ignores unknown ids in stored value", () => {
    const store = new MemoryStore();
    store.set("reader.plans", JSON.stringify(["bogus", "hope"]));
    const { result } = renderHook(() => usePlanSelection(store));
    expect(result.current[0]).toEqual(["hope"]);
  });

  it("falls back to default on malformed JSON", () => {
    const store = new MemoryStore();
    store.set("reader.plans", "not json");
    const { result } = renderHook(() => usePlanSelection(store));
    expect(result.current[0]).toEqual(DEFAULT_PLAN_IDS);
  });
});
