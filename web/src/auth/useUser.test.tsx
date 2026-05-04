import { describe, expect, it, vi, beforeEach, afterEach } from "vitest";
import { renderHook, act, waitFor } from "@testing-library/react";
import { useUser } from "./useUser";

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

describe("useUser", () => {
  it("hydrates as guest when /me returns 401", async () => {
    (globalThis.fetch as ReturnType<typeof vi.fn>).mockResolvedValueOnce(
      new Response("", { status: 401 }),
    );
    const { result } = renderHook(() => useUser());
    expect(result.current.hydrating).toBe(true);
    await waitFor(() => expect(result.current.hydrating).toBe(false));
    expect(result.current.user).toBeNull();
  });

  it("hydrates as signed-in when /me returns the user", async () => {
    const user = {
      id: 7,
      email: "a@b.c",
      created_at: "x",
      updated_at: "x",
    };
    (globalThis.fetch as ReturnType<typeof vi.fn>).mockResolvedValueOnce(
      okJSON(user),
    );
    const { result } = renderHook(() => useUser());
    await waitFor(() => expect(result.current.hydrating).toBe(false));
    expect(result.current.user).toEqual(user);
  });

  it("signin sets the user on success", async () => {
    const fetchMock = globalThis.fetch as ReturnType<typeof vi.fn>;
    fetchMock.mockResolvedValueOnce(new Response("", { status: 401 })); // initial /me
    const { result } = renderHook(() => useUser());
    await waitFor(() => expect(result.current.hydrating).toBe(false));

    const user = { id: 1, email: "a@b.c", created_at: "x", updated_at: "x" };
    fetchMock.mockResolvedValueOnce(okJSON(user));
    await act(async () => {
      const res = await result.current.signin("a@b.c", "twelvecharspw");
      expect(res.kind).toBe("ok");
    });
    expect(result.current.user).toEqual(user);
  });

  it("signout clears the user", async () => {
    const fetchMock = globalThis.fetch as ReturnType<typeof vi.fn>;
    const user = { id: 1, email: "a@b.c", created_at: "x", updated_at: "x" };
    fetchMock.mockResolvedValueOnce(okJSON(user)); // initial /me
    const { result } = renderHook(() => useUser());
    await waitFor(() => expect(result.current.user).toEqual(user));

    fetchMock.mockResolvedValueOnce(new Response(null, { status: 204 }));
    await act(async () => {
      await result.current.signout();
    });
    expect(result.current.user).toBeNull();
  });
});
