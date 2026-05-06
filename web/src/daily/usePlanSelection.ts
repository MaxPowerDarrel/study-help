import { useMemo, useSyncExternalStore } from "react";
import { defaultToggleStore, ToggleStore } from "../platform/ToggleStore";
import { DEFAULT_PLAN_IDS, isPlanID, type PlanID } from "./plans";

const STORAGE_KEY = "reader.plans";

// usePlanSelection returns the currently-selected plan IDs for the
// daily tab. Persisted via ToggleStore (localStorage) so guests and
// signed-in users share the same path; cross-device persistence is a
// non-goal (see specs/multi-plan.md). Empty / invalid storage falls
// back to DEFAULT_PLAN_IDS so the daily tab is never stuck on a blank
// state.
export function usePlanSelection(
  store: ToggleStore = defaultToggleStore,
): readonly [PlanID[], (next: PlanID[]) => void] {
  const raw = useSyncExternalStore(
    (cb) => store.subscribe(cb),
    () => store.get(STORAGE_KEY),
    () => null,
  );
  const planIDs = useMemo<PlanID[]>(() => parse(raw), [raw]);
  const setPlanIDs = (next: PlanID[]): void => {
    const cleaned = next.filter(isPlanID);
    const final = cleaned.length === 0 ? DEFAULT_PLAN_IDS : cleaned;
    store.set(STORAGE_KEY, JSON.stringify(final));
  };
  return [planIDs, setPlanIDs] as const;
}

function parse(raw: string | null): PlanID[] {
  if (!raw) return DEFAULT_PLAN_IDS;
  try {
    const parsed = JSON.parse(raw);
    if (!Array.isArray(parsed)) return DEFAULT_PLAN_IDS;
    const ids = parsed.filter(isPlanID);
    return ids.length === 0 ? DEFAULT_PLAN_IDS : ids;
  } catch {
    return DEFAULT_PLAN_IDS;
  }
}
