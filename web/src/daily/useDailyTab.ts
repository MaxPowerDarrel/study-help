import { useCallback, useEffect, useRef, useState } from "react";
import { type DailyPassage, fetchDailyReading, fetchPassage } from "../api";
import { defaultTimezoneProvider } from "../platform/TimezoneProvider";
import type { Toggles } from "../toggles";
import type { TranslationID } from "../translations/catalog";
import type { PlanID } from "./plans";

// todayString returns today's local calendar date in YYYY-MM-DD form,
// using the user's resolved timezone. Exported so callers (e.g.
// useStoredDailyDate in restore.ts) can pass a single shared "today"
// value in to keep the snap-to-today comparison consistent with what the
// daily tab itself uses.
export function todayString(): string {
  const tz = defaultTimezoneProvider.get();
  try {
    return new Intl.DateTimeFormat("en-CA", { timeZone: tz }).format(
      new Date(),
    );
  } catch {
    return new Intl.DateTimeFormat("en-CA").format(new Date());
  }
}

export type Testament = "OT" | "NT" | "Psalm" | "";

export type DailyPill = {
  planID: PlanID;
  planName: string;
  passage: DailyPassage;
  html: string;
  loading: boolean;
  error: string | null;
};

export type PlanMessage = {
  planID: PlanID;
  planName: string;
  text: string;
};

export type DailyState = {
  pills: DailyPill[];
  // Index into pills. -1 when there are no pills (info-card-only days).
  activeIdx: number;
  planMessages: PlanMessage[];
  // Number of plans contributing to this load. DailyPanel uses this to
  // decide whether to label pills with their testament (single plan) or
  // their plan name (multiple plans).
  planCount: number;
};

export type DailyLoad =
  | { kind: "idle" }
  | { kind: "loading" }
  | { kind: "ready"; state: DailyState }
  | { kind: "error" };

export type UseDailyTab = {
  daily: DailyLoad;
  setActivePill: (idx: number) => void;
  selectedDate: string;
  setSelectedDate: (date: string) => void;
};

// useDailyTab owns the daily-reading state machine: the per-pill load
// and the active-pill switch. `selectedDate` is owned by the caller
// (App.tsx) so it can be hydrated from the restore-last-location store
// alongside the active tab and read-tab passage. The load effect
// snapshots `toggles` and `translation` via refs so identity changes
// don't refire it; mid-flight requests are dropped via fetchId
// comparison. Re-fetching is gated by daily.kind === "idle" (re-armed
// by the date / translation / planIDs reset effect below).
export function useDailyTab(
  enabled: boolean,
  toggles: Toggles,
  translation: TranslationID,
  planIDs: PlanID[],
  selectedDate: string,
  setSelectedDate: (date: string) => void,
): UseDailyTab {
  const [daily, setDaily] = useState<DailyLoad>({ kind: "idle" });
  const fetchId = useRef(0);

  const togglesRef = useRef(toggles);
  togglesRef.current = toggles;
  const translationRef = useRef(translation);
  translationRef.current = translation;
  const planIDsRef = useRef(planIDs);
  planIDsRef.current = planIDs;

  useEffect(() => {
    if (!enabled) return;
    if (daily.kind !== "idle") return;
    const id = ++fetchId.current;
    setDaily({ kind: "loading" });
    const tz = defaultTimezoneProvider.get();
    const planIDsSnapshot = planIDsRef.current;
    fetchDailyReading(tz, planIDsSnapshot, selectedDate).then((result) => {
      if (fetchId.current !== id) return;
      if (result.kind === "error") {
        setDaily({ kind: "error" });
        return;
      }

      const pills: DailyPill[] = [];
      const planMessages: PlanMessage[] = [];
      for (const plan of result.plans) {
        if (plan.message) {
          planMessages.push({
            planID: plan.id,
            planName: plan.name,
            text: plan.message,
          });
        }
        for (const passage of plan.passages) {
          pills.push({
            planID: plan.id,
            planName: plan.name,
            passage,
            html: "",
            loading: true,
            error: null,
          });
        }
      }
      const activeIdx = pills.length > 0 ? 0 : -1;
      setDaily({
        kind: "ready",
        state: {
          pills,
          activeIdx,
          planMessages,
          planCount: result.plans.length,
        },
      });

      const t = togglesRef.current;
      const tr = translationRef.current;
      pills.forEach((pill, idx) => loadPill(pill, idx, t, tr, id));
    });

    function loadPill(
      pill: DailyPill,
      idx: number,
      t: Toggles,
      tr: TranslationID,
      id: number,
    ) {
      const queries = passageQueries(pill.passage);
      Promise.all(queries.map((q) => fetchPassage(q, t, tr))).then(
        (results) => {
          if (fetchId.current !== id) return;
          const failed = results.find((r) => r.kind !== "ok");
          if (failed) {
            const msg =
              failed.kind === "rate_limited"
                ? "Service is busy, try again in a moment"
                : "Something went wrong, try again";
            setDaily((prev) =>
              updatePill(prev, idx, { error: msg, loading: false }),
            );
            return;
          }
          const html = results
            .map((r) => (r.kind === "ok" ? r.html : ""))
            .join("");
          setDaily((prev) => updatePill(prev, idx, { html, loading: false }));
        },
      );
    }
    // selectedDate / toggles / translation / planIDs are snapshotted via
    // closure and refs intentionally; re-runs are gated by
    // daily.kind === "idle" (re-armed by the reset effect below).
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [enabled, daily.kind]);

  // Date / translation / planIDs change → reset to idle so the load
  // effect re-triggers with the new context.
  const planIDsKey = planIDs.join(",");
  useEffect(() => {
    setDaily({ kind: "idle" });
  }, [selectedDate, translation, planIDsKey]);

  const setActivePill = useCallback((idx: number) => {
    setDaily((prev) =>
      prev.kind === "ready"
        ? { ...prev, state: { ...prev.state, activeIdx: idx } }
        : prev,
    );
  }, []);

  return {
    daily,
    setActivePill,
    selectedDate,
    setSelectedDate,
  };
}

// passageQueries turns one DailyPassage into one or more `?q=` strings.
// Comma-separated chapters (e.g. "Matthew 11,12") fan out into
// multiple queries because the canon validator on the server only
// accepts single chapters or hyphen ranges per call.
export function passageQueries(p: DailyPassage): string[] {
  const pieces = p.chapters
    .split(",")
    .map((s) => s.trim())
    .filter(Boolean);
  if (pieces.length === 0) return [];
  return pieces.map((piece) => `${p.book} ${piece}`);
}

function updatePill(
  prev: DailyLoad,
  idx: number,
  patch: Partial<Pick<DailyPill, "html" | "loading" | "error">>,
): DailyLoad {
  if (prev.kind !== "ready") return prev;
  const pills = prev.state.pills.map((p, i) =>
    i === idx ? { ...p, ...patch } : p,
  );
  return { ...prev, state: { ...prev.state, pills } };
}
