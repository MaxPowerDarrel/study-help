import {
  type RefObject,
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react";
import { type DailyPassage, fetchDailyReading } from "../api";
import { fetchPassage } from "../api";
import { defaultTimezoneProvider } from "../platform/TimezoneProvider";
import type { Toggles } from "../toggles";
import type { TranslationID } from "../translations/catalog";
import type { PlanID } from "./plans";

export type Testament = "OT" | "NT" | "";

export type DailyChapterState = {
  book: string;
  chapter: number;
  html: string;
  loading: boolean;
  error: string | null;
};

export type DailyPill = {
  planID: PlanID;
  planName: string;
  passage: DailyPassage;
  chapters: DailyChapterState[];
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
  // The number of plans contributing to this load — used by DailyPanel
  // to decide whether to label pills with their testament (single plan)
  // or with their plan name (multiple plans).
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
  activePill: DailyPill | null;
  activeChapterNumbers: number[];
  getArticleRef: (
    planID: PlanID,
    book: string,
    chapter: number,
  ) => RefObject<HTMLElement | null>;
  getArticleEl: (book: string, chapter: number) => HTMLElement | null;
  // If a tap targets a chapter inside an inactive pill, switch active
  // and return the new pill index. Returns null when no switch was
  // needed.
  switchToOwningPill: (book: string, chapter: number) => number | null;
};

// useDailyTab owns the daily-reading state machine: date selection,
// the per-plan / per-pill loads, the per-chapter article refs, and
// the active-pill switch. The load effect snapshots `toggles` and
// `translation` via refs so identity changes don't refire it; mid-
// flight requests are dropped via fetchId comparison. Re-fetching is
// gated by daily.kind === "idle" (re-armed by the date / translation
// / planIDs reset effect below).
export function useDailyTab(
  enabled: boolean,
  toggles: Toggles,
  translation: TranslationID,
  planIDs: PlanID[],
): UseDailyTab {
  const [daily, setDaily] = useState<DailyLoad>({ kind: "idle" });
  const [selectedDate, setSelectedDate] = useState<string>(todayString);
  const fetchId = useRef(0);

  const togglesRef = useRef(toggles);
  togglesRef.current = toggles;
  const translationRef = useRef(translation);
  translationRef.current = translation;
  const planIDsRef = useRef(planIDs);
  planIDsRef.current = planIDs;

  // Per-chapter article refs keyed by `${planID}|${book}:${chapter}`.
  // The plan-ID prefix avoids collision when two plans schedule the
  // same chapter on the same day. Populated lazily on first request;
  // never pruned (entries are tiny ref objects and old keys are
  // harmless once their DOM unmounts).
  const articleRefs = useRef<Map<string, RefObject<HTMLElement | null>>>(
    new Map(),
  );
  const refKey = (planID: PlanID, book: string, chapter: number) =>
    `${planID}|${book}:${chapter}`;
  const getArticleRef = useCallback(
    (
      planID: PlanID,
      book: string,
      chapter: number,
    ): RefObject<HTMLElement | null> => {
      const key = refKey(planID, book, chapter);
      const existing = articleRefs.current.get(key);
      if (existing) return existing;
      const ref: RefObject<HTMLElement | null> = { current: null };
      articleRefs.current.set(key, ref);
      return ref;
    },
    [],
  );
  // Note-tap callers don't know which plan owns a chapter; look across
  // all currently-mapped refs and return the first DOM node we have.
  const getArticleEl = useCallback(
    (book: string, chapter: number): HTMLElement | null => {
      const suffix = `|${book}:${chapter}`;
      for (const [key, ref] of articleRefs.current) {
        if (key.endsWith(suffix) && ref.current) return ref.current;
      }
      return null;
    },
    [],
  );

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
            chapters: parseChapterRange(passage.chapters).map((c) => ({
              book: passage.book,
              chapter: c,
              html: "",
              loading: true,
              error: null,
            })),
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
      pills.forEach((pill, idx) => loadPillChapters(pill, idx, t, tr, id));
    });

    function loadPillChapters(
      pill: DailyPill,
      idx: number,
      t: Toggles,
      tr: TranslationID,
      id: number,
    ) {
      // Per-chapter concurrent fetches; each chapter renders as it
      // arrives (a slow chapter doesn't block earlier ones).
      pill.chapters.forEach((c) => {
        fetchPassage(`${c.book} ${c.chapter}`, t, tr).then((res) => {
          if (fetchId.current !== id) return;
          setDaily((prev) => updateChapter(prev, idx, c.book, c.chapter, res));
        });
      });
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

  const activePill = useMemo(() => activePillState(daily), [daily]);
  const activeChapterNumbers = useMemo(
    () => activePill?.chapters.map((c) => c.chapter) ?? [],
    [activePill],
  );

  const switchToOwningPill = useCallback(
    (book: string, chapter: number): number | null => {
      if (daily.kind !== "ready") return null;
      const idx = daily.state.pills.findIndex((p) =>
        p.chapters.some((c) => c.book === book && c.chapter === chapter),
      );
      if (idx < 0 || idx === daily.state.activeIdx) return null;
      setActivePill(idx);
      return idx;
    },
    [daily, setActivePill],
  );

  return {
    daily,
    setActivePill,
    selectedDate,
    setSelectedDate,
    activePill,
    activeChapterNumbers,
    getArticleRef,
    getArticleEl,
    switchToOwningPill,
  };
}

// Expand a daily-reader chapters string ("1-3", "1", "1,3", "119:33-40")
// into individual chapter numbers. Verse-range refs collapse to the
// chapter number (the daily panel renders chapter-block content; per-
// verse fetching is a v1 limitation — see specs/multi-plan.md).
export function parseChapterRange(chapters: string): number[] {
  const out: number[] = [];
  for (const part of chapters.split(",")) {
    const trimmed = part.trim();
    if (trimmed === "") continue;
    // Drop trailing ":start-end" verse range.
    const noVerse = trimmed.split(":")[0];
    const dash = noVerse.indexOf("-");
    if (dash < 0) {
      const n = Number(noVerse);
      if (Number.isFinite(n) && n > 0) out.push(n);
      continue;
    }
    const start = Number(noVerse.slice(0, dash).trim());
    const end = Number(noVerse.slice(dash + 1).trim());
    if (
      !Number.isFinite(start) ||
      !Number.isFinite(end) ||
      start <= 0 ||
      end < start
    ) {
      continue;
    }
    for (let c = start; c <= end; c++) out.push(c);
  }
  return out;
}

// Apply a passage-fetch result to the matching chapter inside `prev`.
// Tolerates concurrent updates: if `prev` is no longer "ready" or the
// pill / chapter don't match, returns prev unchanged.
function updateChapter(
  prev: DailyLoad,
  pillIdx: number,
  book: string,
  chapter: number,
  res:
    | { kind: "ok"; html: string }
    | { kind: "rate_limited" }
    | { kind: "error" },
): DailyLoad {
  if (prev.kind !== "ready") return prev;
  const pill = prev.state.pills[pillIdx];
  if (!pill) return prev;
  const chapters = pill.chapters.map((c) => {
    if (c.book !== book || c.chapter !== chapter) return c;
    if (res.kind === "ok") {
      return { ...c, loading: false, html: res.html };
    }
    const errMsg =
      res.kind === "rate_limited"
        ? "Service is busy, try again in a moment"
        : "Something went wrong, try again";
    return { ...c, loading: false, error: errMsg };
  });
  const pills = prev.state.pills.map((p, i) =>
    i === pillIdx ? { ...p, chapters } : p,
  );
  return { ...prev, state: { ...prev.state, pills } };
}

function activePillState(daily: DailyLoad): DailyPill | null {
  if (daily.kind !== "ready") return null;
  const { pills, activeIdx } = daily.state;
  if (activeIdx < 0 || activeIdx >= pills.length) return null;
  return pills[activeIdx];
}

function todayString(): string {
  const tz = defaultTimezoneProvider.get();
  try {
    return new Intl.DateTimeFormat("en-CA", { timeZone: tz }).format(
      new Date(),
    );
  } catch {
    return new Intl.DateTimeFormat("en-CA").format(new Date());
  }
}
