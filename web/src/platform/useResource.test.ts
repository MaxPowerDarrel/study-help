import { describe, expect, it, vi } from "vitest";
import { act, renderHook, waitFor } from "@testing-library/react";
import { useResource } from "./useResource";

const EMPTY: number[] = [];

describe("useResource", () => {
  it("returns fallback when fetcher is null", async () => {
    const { result } = renderHook(() => useResource<number[]>(null, EMPTY, []));
    await waitFor(() => expect(result.current.loading).toBe(false));
    expect(result.current.data).toBe(EMPTY);
  });

  it("runs the fetcher on mount and stores the result", async () => {
    const fetcher = vi.fn(async () => [1, 2, 3]);
    const { result } = renderHook(() => useResource(fetcher, EMPTY, []));
    await waitFor(() => expect(result.current.data).toEqual([1, 2, 3]));
    expect(fetcher).toHaveBeenCalledOnce();
    expect(result.current.loading).toBe(false);
  });

  it("re-fetches when deps change", async () => {
    const fetcher = vi.fn(async (_signal: AbortSignal) => [1]);
    const { rerender } = renderHook(
      ({ id }: { id: number }) =>
        useResource((signal) => fetcher(signal), EMPTY, [id]),
      { initialProps: { id: 1 } },
    );
    await waitFor(() => expect(fetcher).toHaveBeenCalledOnce());
    rerender({ id: 2 });
    await waitFor(() => expect(fetcher).toHaveBeenCalledTimes(2));
  });

  it("aborts the previous run when deps change", async () => {
    const seen: AbortSignal[] = [];
    const fetcher = (signal: AbortSignal): Promise<number[]> => {
      seen.push(signal);
      return new Promise((resolve) => {
        setTimeout(() => resolve([1]), 100);
      });
    };
    const { rerender } = renderHook(
      ({ id }: { id: number }) => useResource(fetcher, EMPTY, [id]),
      { initialProps: { id: 1 } },
    );
    await waitFor(() => expect(seen.length).toBe(1));
    rerender({ id: 2 });
    await waitFor(() => expect(seen.length).toBe(2));
    expect(seen[0]?.aborted).toBe(true);
    expect(seen[1]?.aborted).toBe(false);
  });

  it("treats fetcher returning null as fallback", async () => {
    const fetcher = vi.fn(async () => null);
    const { result } = renderHook(() => useResource(fetcher, EMPTY, []));
    await waitFor(() => expect(result.current.loading).toBe(false));
    expect(result.current.data).toBe(EMPTY);
  });

  it("refetch() re-runs the latest fetcher", async () => {
    let calls = 0;
    const fetcher = async () => [++calls];
    const { result } = renderHook(() => useResource(fetcher, EMPTY, []));
    await waitFor(() => expect(result.current.data).toEqual([1]));
    await act(async () => {
      await result.current.refetch();
    });
    expect(result.current.data).toEqual([2]);
  });
});
