import {
  type RefObject,
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react";
import { type DailyPassage, fetchDailyReading, fetchPassage } from "../api";
import { defaultTimezoneProvider } from "../platform/TimezoneProvider";
import type { Toggles } from "../toggles";
import type { TranslationID } from "../translations/catalog";

export type Testament = "OT" | "NT";

export type DailyChapterState = {
  book: string;
  chapter: number;
  html: string;
  loading: boolean;
  error: string | null;
};

export type DailyTabState = {
  passage: DailyPassage;
  chapters: DailyChapterState[];
};

export type DailyState = {
  ot: DailyTabState | null;
  nt: DailyTabState | null;
  active: Testament;
};

export type DailyLoad =
  | { kind: "idle" }
  | { kind: "loading" }
  | { kind: "ready"; state: DailyState }
  | { kind: "empty" }
  | { kind: "error" };

export type UseDailyTab = {
  daily: DailyLoad;
  setActivePill: (t: Testament) => void;
  selectedDate: string;
  setSelectedDate: (date: string) => void;
  activeTab: DailyTabState | null;
  activeChapterNumbers: number[];
  getArticleRef: (
    book: string,
    chapter: number,
  ) => RefObject<HTMLElement | null>;
  getArticleEl: (book: string, chapter: number) => HTMLElement | null;
  // If a tap targets a chapter in the inactive pill, switch active and
  // return the new testament. Returns null when no switch was needed.
  switchToOwningPill: (book: string, chapter: number) => Testament | null;
};

// useDailyTab owns the daily-reading state machine: date selection, the
// per-pill OT/NT load, the per-chapter article refs, and the active-pill
// switch. The load effect snapshots `toggles` and `translation` via refs
// so identity changes don't refire it; mid-flight requests are dropped
// via fetchId comparison. Re-fetching is gated by daily.kind === "idle"
// (re-armed by the date / translation reset effect below).
export function useDailyTab(
  enabled: boolean,
  toggles: Toggles,
  translation: TranslationID,
): UseDailyTab {
  const [daily, setDaily] = useState<DailyLoad>({ kind: "idle" });
  const [selectedDate, setSelectedDate] = useState<string>(todayString);
  const fetchId = useRef(0);

  const togglesRef = useRef(toggles);
  togglesRef.current = toggles;
  const translationRef = useRef(translation);
  translationRef.current = translation;

  // Per-chapter article refs, keyed by `${book}:${chapter}`. Populated
  // lazily on first request; never pruned (entries are tiny ref objects
  // and old keys are harmless once their DOM unmounts).
  const articleRefs = useRef<Map<string, RefObject<HTMLElement | null>>>(
    new Map(),
  );
  const getArticleRef = useCallback(
    (book: string, chapter: number): RefObject<HTMLElement | null> => {
      const key = `${book}:${chapter}`;
      const existing = articleRefs.current.get(key);
      if (existing) return existing;
      const ref: RefObject<HTMLElement | null> = { current: null };
      articleRefs.current.set(key, ref);
      return ref;
    },
    [],
  );
  const getArticleEl = useCallback(
    (book: string, chapter: number): HTMLElement | null =>
      articleRefs.current.get(`${book}:${chapter}`)?.current ?? null,
    [],
  );

  useEffect(() => {
    if (!enabled) return;
    if (daily.kind !== "idle") return;
    const id = ++fetchId.current;
    setDaily({ kind: "loading" });
    const tz = defaultTimezoneProvider.get();
    fetchDailyReading(tz, selectedDate).then((result) => {
      if (fetchId.current !== id) return;
      if (result.kind === "error") {
        setDaily({ kind: "error" });
        return;
      }
      if (result.kind === "empty") {
        setDaily({ kind: "empty" });
        return;
      }
      const ot = result.passages.find((p) => p.testament === "OT") ?? null;
      const nt = result.passages.find((p) => p.testament === "NT") ?? null;
      const make = (p: DailyPassage | null): DailyTabState | null =>
        p
          ? {
              passage: p,
              chapters: parseChapterRange(p.chapters).map((c) => ({
                book: p.book,
                chapter: c,
                html: "",
                loading: true,
                error: null,
              })),
            }
          : null;
      const otState = make(ot);
      const ntState = make(nt);
      const active: Testament = ot ? "OT" : "NT";
      setDaily({ kind: "ready", state: { ot: otState, nt: ntState, active } });

      const t = togglesRef.current;
      const tr = translationRef.current;
      if (otState) loadChapters(otState, "ot", t, tr, id);
      if (ntState) loadChapters(ntState, "nt", t, tr, id);
    });

    function loadChapters(
      s: DailyTabState,
      slot: "ot" | "nt",
      t: Toggles,
      tr: TranslationID,
      id: number,
    ) {
      // Per-chapter concurrent fetches; each chapter renders as it
      // arrives (a slow chapter doesn't block earlier ones).
      s.chapters.forEach((c) => {
        fetchPassage(`${c.book} ${c.chapter}`, t, tr).then((res) => {
          if (fetchId.current !== id) return;
          setDaily((prev) => updateChapter(prev, slot, c.book, c.chapter, res));
        });
      });
    }
    // selectedDate / toggles / translation are snapshotted via closure
    // and refs intentionally; re-runs are gated by daily.kind === "idle"
    // (re-armed by the reset effect below).
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [enabled, daily.kind]);

  // Date or translation change → reset to idle so the load effect
  // re-triggers with the new context.
  useEffect(() => {
    setDaily({ kind: "idle" });
  }, [selectedDate, translation]);

  const setActivePill = useCallback((t: Testament) => {
    setDaily((prev) =>
      prev.kind === "ready"
        ? { ...prev, state: { ...prev.state, active: t } }
        : prev,
    );
  }, []);

  const activeTab = useMemo(() => activePillState(daily), [daily]);
  const activeChapterNumbers = useMemo(
    () => activeTab?.chapters.map((c) => c.chapter) ?? [],
    [activeTab],
  );

  const switchToOwningPill = useCallback(
    (book: string, chapter: number): Testament | null => {
      if (daily.kind !== "ready") return null;
      const owning = findOwningTestament(daily.state, book, chapter);
      if (!owning || daily.state.active === owning) return null;
      setActivePill(owning);
      return owning;
    },
    [daily, setActivePill],
  );

  return {
    daily,
    setActivePill,
    selectedDate,
    setSelectedDate,
    activeTab,
    activeChapterNumbers,
    getArticleRef,
    getArticleEl,
    switchToOwningPill,
  };
}

// Expand a daily-reader chapters string ("1-3", "1", "1,3") into
// individual chapter numbers. Falls back to an empty array on
// unrecognized formats.
export function parseChapterRange(chapters: string): number[] {
  const out: number[] = [];
  for (const part of chapters.split(",")) {
    const trimmed = part.trim();
    if (trimmed === "") continue;
    const dash = trimmed.indexOf("-");
    if (dash < 0) {
      const n = Number(trimmed);
      if (Number.isFinite(n) && n > 0) out.push(n);
      continue;
    }
    const start = Number(trimmed.slice(0, dash).trim());
    const end = Number(trimmed.slice(dash + 1).trim());
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
// slot/chapter don't match, returns prev unchanged.
function updateChapter(
  prev: DailyLoad,
  slot: "ot" | "nt",
  book: string,
  chapter: number,
  res:
    | { kind: "ok"; html: string }
    | { kind: "rate_limited" }
    | { kind: "error" },
): DailyLoad {
  if (prev.kind !== "ready") return prev;
  const cur = prev.state[slot];
  if (!cur) return prev;
  const chapters = cur.chapters.map((c) => {
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
  return {
    ...prev,
    state: { ...prev.state, [slot]: { ...cur, chapters } },
  };
}

function activePillState(daily: DailyLoad): DailyTabState | null {
  if (daily.kind !== "ready") return null;
  return daily.state.active === "OT" ? daily.state.ot : daily.state.nt;
}

function findOwningTestament(
  state: DailyState,
  book: string,
  chapter: number,
): Testament | null {
  for (const t of ["OT", "NT"] as Testament[]) {
    const slot = t === "OT" ? state.ot : state.nt;
    if (!slot) continue;
    if (slot.chapters.some((c) => c.book === book && c.chapter === chapter)) {
      return t;
    }
  }
  return null;
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
