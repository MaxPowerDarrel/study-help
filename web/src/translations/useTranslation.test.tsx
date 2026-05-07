import { describe, expect, it, beforeEach, afterEach } from "vitest";
import { renderHook, act } from "@testing-library/react";

import { useTranslation } from "./useTranslation";

function makeMemoryStorage(): Storage {
  const map = new Map<string, string>();
  return {
    get length() {
      return map.size;
    },
    clear: () => map.clear(),
    getItem: (k: string) => (map.has(k) ? (map.get(k) as string) : null),
    setItem: (k: string, v: string) => {
      map.set(k, String(v));
    },
    removeItem: (k: string) => {
      map.delete(k);
    },
    key: (i: number) => Array.from(map.keys())[i] ?? null,
  };
}

let originalLocalStorageDescriptor: PropertyDescriptor | undefined;

beforeEach(() => {
  originalLocalStorageDescriptor = Object.getOwnPropertyDescriptor(
    window,
    "localStorage",
  );
  Object.defineProperty(window, "localStorage", {
    configurable: true,
    value: makeMemoryStorage(),
  });
});

afterEach(() => {
  if (originalLocalStorageDescriptor) {
    Object.defineProperty(
      window,
      "localStorage",
      originalLocalStorageDescriptor,
    );
  }
});

describe("useTranslation", () => {
  it("falls back to ESV when nothing is stored", () => {
    const { result } = renderHook(() => useTranslation());
    expect(result.current.translation).toBe("ESV");
  });

  it("hydrates from localStorage", () => {
    window.localStorage.setItem("translation", "NIV");
    const { result } = renderHook(() => useTranslation());
    expect(result.current.translation).toBe("NIV");
  });

  it("ignores an unknown stored value", () => {
    window.localStorage.setItem("translation", "BOGUS");
    const { result } = renderHook(() => useTranslation());
    expect(result.current.translation).toBe("ESV");
  });

  it("setTranslation updates state and writes localStorage", () => {
    const { result } = renderHook(() => useTranslation());
    act(() => {
      result.current.setTranslation("NIV");
    });
    expect(result.current.translation).toBe("NIV");
    expect(window.localStorage.getItem("translation")).toBe("NIV");
  });
});
