import { describe, expect, it, vi, beforeEach, afterEach } from "vitest";
import { renderHook, act } from "@testing-library/react";

import type { User } from "../auth/api";
import { useTranslation } from "./useTranslation";

// In-memory localStorage shim. The jsdom localStorage in this vitest
// setup doesn't fully implement the Storage interface (no clear()), so
// we replace it for the duration of the suite.
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
  vi.stubGlobal("fetch", vi.fn());
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
  vi.unstubAllGlobals();
  if (originalLocalStorageDescriptor) {
    Object.defineProperty(
      window,
      "localStorage",
      originalLocalStorageDescriptor,
    );
  }
});

const sampleUser = (translation = "ESV"): User => ({
  id: 1,
  email: "alice@example.com",
  translation,
  created_at: "2026-05-04T00:00:00Z",
  updated_at: "2026-05-04T00:00:00Z",
});

describe("useTranslation", () => {
  it("falls back to ESV when there is no user and no guest pick", () => {
    const { result } = renderHook(() => useTranslation(null));
    expect(result.current.translation).toBe("ESV");
  });

  it("hydrates from the user record when signed in", () => {
    const { result } = renderHook(() => useTranslation(sampleUser("NIV")));
    expect(result.current.translation).toBe("NIV");
  });

  it("hydrates from the guest store when there is no user", () => {
    window.localStorage.setItem("translation", "NIV");
    const { result } = renderHook(() => useTranslation(null));
    expect(result.current.translation).toBe("NIV");
  });

  it("ignores an unknown guest value", () => {
    window.localStorage.setItem("translation", "BOGUS");
    const { result } = renderHook(() => useTranslation(null));
    expect(result.current.translation).toBe("ESV");
  });

  it("guest setTranslation updates state and writes localStorage without fetching", async () => {
    const fetchMock = globalThis.fetch as ReturnType<typeof vi.fn>;
    const { result } = renderHook(() => useTranslation(null));
    let res: { kind: string } | undefined;
    await act(async () => {
      res = await result.current.setTranslation("NIV");
    });
    expect(res?.kind).toBe("ok-local");
    expect(result.current.translation).toBe("NIV");
    expect(window.localStorage.getItem("translation")).toBe("NIV");
    expect(fetchMock).not.toHaveBeenCalled();
  });

  it("signed-in setTranslation calls PATCH and updates state on 200", async () => {
    const fetchMock = globalThis.fetch as ReturnType<typeof vi.fn>;
    const updated = sampleUser("NIV");
    fetchMock.mockResolvedValueOnce(
      new Response(JSON.stringify(updated), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      }),
    );
    const onUserUpdated = vi.fn();
    const { result } = renderHook(() =>
      useTranslation(sampleUser("ESV"), onUserUpdated),
    );
    let res: { kind: string } | undefined;
    await act(async () => {
      res = await result.current.setTranslation("NIV");
    });
    expect(res?.kind).toBe("ok");
    expect(result.current.translation).toBe("NIV");
    expect(onUserUpdated).toHaveBeenCalledWith(updated);
    expect(fetchMock).toHaveBeenCalledWith(
      "/api/auth/me",
      expect.objectContaining({ method: "PATCH" }),
    );
    // Successful PATCH clears the guest store.
    expect(window.localStorage.getItem("translation")).toBeNull();
  });

  it("setTranslation reverts on 400 invalid", async () => {
    const fetchMock = globalThis.fetch as ReturnType<typeof vi.fn>;
    fetchMock.mockResolvedValueOnce(
      new Response("unknown translation", { status: 400 }),
    );
    const onUserUpdated = vi.fn();
    const { result } = renderHook(() =>
      useTranslation(sampleUser("ESV"), onUserUpdated),
    );
    let res: { kind: string } | undefined;
    await act(async () => {
      res = await result.current.setTranslation("NIV");
    });
    expect(res?.kind).toBe("invalid");
    expect(result.current.translation).toBe("ESV");
    expect(onUserUpdated).not.toHaveBeenCalled();
    expect(window.localStorage.getItem("translation")).toBeNull();
  });

  it("on sign-in transition, pushes guest pick to the account when it differs", async () => {
    window.localStorage.setItem("translation", "NIV");
    const fetchMock = globalThis.fetch as ReturnType<typeof vi.fn>;
    const updated = sampleUser("NIV");
    fetchMock.mockResolvedValueOnce(
      new Response(JSON.stringify(updated), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      }),
    );
    const onUserUpdated = vi.fn();
    const { result, rerender } = renderHook(
      ({ user }: { user: User | null }) => useTranslation(user, onUserUpdated),
      { initialProps: { user: null as User | null } },
    );
    expect(result.current.translation).toBe("NIV");

    await act(async () => {
      rerender({ user: sampleUser("ESV") });
    });

    expect(fetchMock).toHaveBeenCalledWith(
      "/api/auth/me",
      expect.objectContaining({ method: "PATCH" }),
    );
    const body = JSON.parse(
      (fetchMock.mock.calls[0][1] as RequestInit).body as string,
    );
    expect(body).toEqual({ translation: "NIV" });
    expect(result.current.translation).toBe("NIV");
    expect(onUserUpdated).toHaveBeenCalledWith(updated);
    expect(window.localStorage.getItem("translation")).toBeNull();
  });

  it("on sign-in transition, no PATCH when guest pick matches the account", async () => {
    window.localStorage.setItem("translation", "ESV");
    const fetchMock = globalThis.fetch as ReturnType<typeof vi.fn>;
    const { result, rerender } = renderHook(
      ({ user }: { user: User | null }) => useTranslation(user),
      { initialProps: { user: null as User | null } },
    );
    expect(result.current.translation).toBe("ESV");

    await act(async () => {
      rerender({ user: sampleUser("ESV") });
    });

    expect(fetchMock).not.toHaveBeenCalled();
    expect(result.current.translation).toBe("ESV");
    // The matching guest entry is cleared on sign-in.
    expect(window.localStorage.getItem("translation")).toBeNull();
  });

  it("on sign-out, persists the last-active translation into the guest store", () => {
    const { result, rerender } = renderHook(
      ({ user }: { user: User | null }) => useTranslation(user),
      { initialProps: { user: sampleUser("NIV") as User | null } },
    );
    expect(result.current.translation).toBe("NIV");

    rerender({ user: null });
    expect(result.current.translation).toBe("NIV");
    expect(window.localStorage.getItem("translation")).toBe("NIV");
  });
});
