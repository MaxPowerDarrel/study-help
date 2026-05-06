import { useCallback, useEffect, useRef, useState } from "react";
import type { DependencyList } from "react";

// Fetcher returns the resolved value, or null when the call did not
// produce a usable result (guest, error, etc.). Receives an AbortSignal
// so the hook can cancel on unmount or dep change.
export type Fetcher<T> = (signal: AbortSignal) => Promise<T | null>;

export type ResourceState<T> = {
  data: T;
  loading: boolean;
  refetch: () => Promise<void>;
};

// useResource owns a per-deps async cache. Pass `null` for fetcher to
// short-circuit (guest / unknown chapter / etc.) — the hook stays at
// `fallback` and never hits the network. Re-fetches on dep change and
// on refetch().
//
// `fallback` should be a stable reference across renders (module-level
// const or memoized) so disabled-state setData calls don't churn.
export function useResource<T>(
  fetcher: Fetcher<T> | null,
  fallback: T,
  deps: DependencyList,
): ResourceState<T> {
  const [data, setData] = useState<T>(fallback);
  const [loading, setLoading] = useState(false);
  // Hold the latest fetcher in a ref so refetch() always sees the
  // current closure without forcing the effect to re-fire on identity
  // change. The `deps` array gates re-fetching.
  const fetcherRef = useRef(fetcher);
  fetcherRef.current = fetcher;

  const run = useCallback(async (signal: AbortSignal) => {
    const f = fetcherRef.current;
    if (!f) {
      setData(fallback);
      return;
    }
    setLoading(true);
    try {
      const result = await f(signal);
      if (signal.aborted) return;
      setData(result ?? fallback);
    } finally {
      if (!signal.aborted) setLoading(false);
    }
    // fallback is captured by closure; callers pass a stable reference.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  useEffect(() => {
    const ctrl = new AbortController();
    void run(ctrl.signal);
    return () => ctrl.abort();
    // run is stable; deps drive re-fetching.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, deps);

  const refetch = useCallback(async () => {
    const ctrl = new AbortController();
    await run(ctrl.signal);
  }, [run]);

  return { data, loading, refetch };
}
