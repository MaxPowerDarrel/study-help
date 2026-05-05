import { describe, expect, it, vi, beforeEach, afterEach } from "vitest";
import { renderHook, act } from "@testing-library/react";

import type { User } from "../auth/api";
import { useTranslation } from "./useTranslation";

beforeEach(() => {
  vi.stubGlobal("fetch", vi.fn());
});

afterEach(() => {
  vi.unstubAllGlobals();
});

const sampleUser = (translation = "ESV"): User => ({
  id: 1,
  email: "alice@example.com",
  translation,
  created_at: "2026-05-04T00:00:00Z",
  updated_at: "2026-05-04T00:00:00Z",
});

describe("useTranslation", () => {
  it("falls back to ESV when there is no user", () => {
    const { result } = renderHook(() => useTranslation(null));
    expect(result.current.translation).toBe("ESV");
  });

  it("hydrates from the user record", () => {
    const { result } = renderHook(() => useTranslation(sampleUser("ESV")));
    expect(result.current.translation).toBe("ESV");
  });

  it("re-syncs when the user prop changes", () => {
    const { result, rerender } = renderHook(
      ({ user }: { user: User | null }) => useTranslation(user),
      { initialProps: { user: null as User | null } },
    );
    expect(result.current.translation).toBe("ESV");
    rerender({ user: sampleUser("ESV") });
    expect(result.current.translation).toBe("ESV");
  });

  it("setTranslation calls PATCH and updates state on 200", async () => {
    const fetchMock = globalThis.fetch as ReturnType<typeof vi.fn>;
    const updated = sampleUser("ESV");
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
      res = await result.current.setTranslation("ESV");
    });
    expect(res?.kind).toBe("ok");
    expect(onUserUpdated).toHaveBeenCalledWith(updated);
    expect(fetchMock).toHaveBeenCalledWith(
      "/api/auth/me",
      expect.objectContaining({ method: "PATCH" }),
    );
  });

  it("setTranslation does not update state on 400", async () => {
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
      res = await result.current.setTranslation("ESV");
    });
    expect(res?.kind).toBe("invalid");
    expect(onUserUpdated).not.toHaveBeenCalled();
  });

  it("setTranslation surfaces guest result on 401", async () => {
    const fetchMock = globalThis.fetch as ReturnType<typeof vi.fn>;
    fetchMock.mockResolvedValueOnce(new Response("", { status: 401 }));
    const { result } = renderHook(() => useTranslation(null));
    let res: { kind: string } | undefined;
    await act(async () => {
      res = await result.current.setTranslation("ESV");
    });
    expect(res?.kind).toBe("guest");
  });
});
