import { useEffect, useMemo, useRef, useState } from "react";
import {
  CANON,
  ChapterRef,
  nextChapter,
  prevChapter,
  refToQuery,
} from "./canon";
import { DailyPassage, fetchDailyReading, fetchPassage } from "./api";
import { useToggles } from "./toggles";
import { useTheme } from "./theme";
import { defaultTimezoneProvider } from "./platform/TimezoneProvider";
import { SettingsPane } from "./SettingsPane";
import { AuthChip } from "./auth/AuthChip";
import { AuthPanel } from "./auth/AuthPanel";
import { useUser } from "./auth/useUser";
import { PassageView } from "./highlights/PassageView";
import styles from "./App.module.css";

type Tab = "read" | "daily";
type Testament = "OT" | "NT";

type DailyTabState = {
  q: string;
  passage: DailyPassage;
  html: string;
  loading: boolean;
  error: string | null;
};

type DailyState = {
  ot: DailyTabState | null;
  nt: DailyTabState | null;
  active: Testament;
};

type DailyLoad =
  | { kind: "idle" }
  | { kind: "loading" }
  | { kind: "ready"; state: DailyState }
  | { kind: "empty" }
  | { kind: "error" };

export function App() {
  const [tab, setTab] = useState<Tab>("read");
  const [ref, setRef] = useState<ChapterRef>({ bookIndex: 42, chapter: 3 }); // John 3
  const [range, setRange] = useState<{ start: number; end: number } | null>(
    null,
  );
  const [toggles, setToggles] = useToggles();
  const [theme, setTheme] = useTheme();
  const [html, setHtml] = useState<string>("");
  const [loading, setLoading] = useState(false);
  const [toast, setToast] = useState<string | null>(null);
  const [settingsOpen, setSettingsOpen] = useState(false);
  const [authOpen, setAuthOpen] = useState(false);
  const userState = useUser();
  const [daily, setDaily] = useState<DailyLoad>({ kind: "idle" });
  const [selectedDate, setSelectedDate] = useState<string>(todayString());
  const dailyFetchId = useRef(0);

  const q = useMemo(() => refToQuery(ref, range ?? undefined), [ref, range]);

  // Read-tab passage fetch.
  useEffect(() => {
    if (tab !== "read") return;
    let cancelled = false;
    setLoading(true);
    setToast(null);
    fetchPassage(q, toggles).then((result) => {
      if (cancelled) return;
      setLoading(false);
      switch (result.kind) {
        case "ok":
          setHtml(result.html);
          break;
        case "rate_limited":
          setToast("Service is busy, try again in a moment");
          break;
        case "error":
          setToast("Something went wrong, try again");
          break;
      }
    });
    return () => {
      cancelled = true;
    };
  }, [q, toggles, tab]);

  // Daily-tab lazy load: fires when daily.kind is "idle" (first activation or
  // after a date change). selectedDate and toggles are intentionally read from
  // closure rather than as dependencies — re-runs are triggered by daily.kind
  // transitioning to "idle" (via the selectedDate reset effect below).
  useEffect(() => {
    if (tab !== "daily") return;
    if (daily.kind !== "idle") return;
    const fetchId = ++dailyFetchId.current;
    setDaily({ kind: "loading" });
    const tz = defaultTimezoneProvider.get();
    fetchDailyReading(tz, selectedDate).then((result) => {
      if (dailyFetchId.current !== fetchId) return;
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
      const otState: DailyTabState | null = ot
        ? {
            q: assembleQ(ot),
            passage: ot,
            html: "",
            loading: true,
            error: null,
          }
        : null;
      const ntState: DailyTabState | null = nt
        ? {
            q: assembleQ(nt),
            passage: nt,
            html: "",
            loading: true,
            error: null,
          }
        : null;
      const active: Testament = ot ? "OT" : "NT";
      setDaily({ kind: "ready", state: { ot: otState, nt: ntState, active } });

      const snapshotToggles = toggles;
      if (ot) loadDailyPassage(ot, "ot", snapshotToggles);
      if (nt) loadDailyPassage(nt, "nt", snapshotToggles);
    });

    function loadDailyPassage(
      p: DailyPassage,
      slot: "ot" | "nt",
      t: typeof toggles,
    ) {
      fetchPassage(assembleQ(p), t).then((res) => {
        if (dailyFetchId.current !== fetchId) return;
        setDaily((prev) => {
          if (prev.kind !== "ready") return prev;
          const cur = prev.state[slot];
          if (!cur) return prev;
          if (res.kind === "ok") {
            return {
              ...prev,
              state: {
                ...prev.state,
                [slot]: { ...cur, loading: false, html: res.html },
              },
            };
          }
          const errMsg =
            res.kind === "rate_limited"
              ? "Service is busy, try again in a moment"
              : "Something went wrong, try again";
          return {
            ...prev,
            state: {
              ...prev.state,
              [slot]: { ...cur, loading: false, error: errMsg },
            },
          };
        });
      });
    }
    // selectedDate and toggles are snapshot intentionally; see comment above.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [tab, daily.kind]);

  // When the user picks a different date, reset daily to idle so the load
  // effect above re-triggers for the new date.
  useEffect(() => {
    setDaily({ kind: "idle" });
  }, [selectedDate]);

  const next = nextChapter(ref);
  const prev = prevChapter(ref);
  const book = CANON[ref.bookIndex];

  return (
    <div className={styles.app}>
      <header className={styles.appHeader}>
        <div className={styles.appTitle}>study-help</div>
        <nav className={styles.modeTabs} role="tablist" aria-label="Mode">
          <button
            type="button"
            role="tab"
            aria-selected={tab === "read"}
            className={
              tab === "read"
                ? `${styles.modeTab} ${styles.modeTabActive}`
                : styles.modeTab
            }
            onClick={() => setTab("read")}
          >
            Read
          </button>
          <button
            type="button"
            role="tab"
            aria-selected={tab === "daily"}
            className={
              tab === "daily"
                ? `${styles.modeTab} ${styles.modeTabActive}`
                : styles.modeTab
            }
            onClick={() => setTab("daily")}
          >
            Daily
          </button>
        </nav>
        <div className={styles.headerRight}>
          <AuthChip
            user={userState.user}
            onOpenSignin={() => setAuthOpen(true)}
            onSignout={() => userState.signout()}
          />
          <button
            type="button"
            className={styles.gear}
            aria-label="Settings"
            onClick={() => setSettingsOpen((v) => !v)}
          >
            ⚙
          </button>
        </div>
      </header>
      <SettingsPane
        open={settingsOpen}
        onClose={() => setSettingsOpen(false)}
        theme={theme}
        setTheme={setTheme}
      />
      <AuthPanel
        open={authOpen}
        onClose={() => setAuthOpen(false)}
        signin={userState.signin}
        signup={userState.signup}
      />
      <div
        className={
          tab === "daily"
            ? `${styles.layout} ${styles.layoutDaily}`
            : styles.layout
        }
      >
        {tab === "read" && (
          <aside className={styles.picker}>
            <label>
              Book
              <select
                value={ref.bookIndex}
                onChange={(e) => {
                  setRef({ bookIndex: Number(e.target.value), chapter: 1 });
                  setRange(null);
                }}
              >
                {CANON.map((b, i) => (
                  <option key={b.name} value={i}>
                    {b.name}
                  </option>
                ))}
              </select>
            </label>
            <label>
              Chapter
              <select
                value={ref.chapter}
                onChange={(e) => {
                  setRef({ ...ref, chapter: Number(e.target.value) });
                  setRange(null);
                }}
              >
                {Array.from({ length: book.chapters }, (_, i) => i + 1).map(
                  (c) => (
                    <option key={c} value={c}>
                      {c}
                    </option>
                  ),
                )}
              </select>
            </label>

            <fieldset className={styles.range}>
              <legend>Verse range</legend>
              <input
                type="number"
                min={1}
                placeholder="start"
                value={range?.start ?? ""}
                onChange={(e) => {
                  const start = Number(e.target.value);
                  if (!start) {
                    setRange(null);
                    return;
                  }
                  setRange({
                    start,
                    end: range?.end && range.end >= start ? range.end : start,
                  });
                }}
              />
              <span>–</span>
              <input
                type="number"
                min={1}
                placeholder="end"
                value={range?.end ?? ""}
                disabled={!range}
                onChange={(e) => {
                  if (!range) return;
                  const end = Number(e.target.value);
                  setRange({ start: range.start, end: end || range.start });
                }}
              />
              {range && (
                <button
                  type="button"
                  className={styles.clearBtn}
                  onClick={() => setRange(null)}
                >
                  Clear
                </button>
              )}
            </fieldset>

            <fieldset className={styles.toggles}>
              <legend>Show</legend>
              {(
                [
                  ["include_headings", "Section headings"],
                  ["include_footnotes", "Footnotes"],
                  ["include_verse_numbers", "Verse numbers"],
                  ["include_passage_references", "Passage reference"],
                  ["include_word_of_christ", "Words of Christ"],
                ] as const
              ).map(([key, label]) => (
                <label key={key}>
                  <input
                    type="checkbox"
                    checked={toggles[key]}
                    onChange={(e) =>
                      setToggles({ ...toggles, [key]: e.target.checked })
                    }
                  />
                  {label}
                </label>
              ))}
            </fieldset>

            <nav className={styles.chapterNav}>
              {prev && (
                <button
                  type="button"
                  className={styles.navBtn}
                  onClick={() => {
                    setRef(prev);
                    setRange(null);
                  }}
                >
                  ← Previous
                </button>
              )}
              {next && (
                <button
                  type="button"
                  className={styles.navBtn}
                  onClick={() => {
                    setRef(next);
                    setRange(null);
                  }}
                >
                  Next →
                </button>
              )}
            </nav>
          </aside>
        )}

        <main className={styles.readingSurface}>
          {tab === "read" ? (
            <>
              {loading && (
                <div className={styles.spinner} aria-label="loading" />
              )}
              {!loading && html && (
                <PassageView
                  html={html}
                  book={book.name}
                  chapter={ref.chapter}
                  isSignedIn={userState.user !== null}
                  onGuestSignin={() => setAuthOpen(true)}
                />
              )}
            </>
          ) : (
            <DailyPanel
              daily={daily}
              setDaily={setDaily}
              selectedDate={selectedDate}
              setSelectedDate={setSelectedDate}
            />
          )}
          {toast && (
            <div className={styles.toast} role="alert">
              {toast}
            </div>
          )}
        </main>
      </div>
    </div>
  );
}

function assembleQ(p: DailyPassage): string {
  return `${p.book} ${p.chapters}`;
}

function formatChapters(chapters: string): string {
  return chapters.replace(/-/g, "–");
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

function offsetDate(dateStr: string, days: number): string {
  const d = new Date(`${dateStr}T00:00:00`);
  d.setDate(d.getDate() + days);
  return new Intl.DateTimeFormat("en-CA").format(d);
}

function formatDate(dateStr: string): string {
  const tz = defaultTimezoneProvider.get();
  const d = new Date(`${dateStr}T00:00:00`);
  try {
    return new Intl.DateTimeFormat(undefined, {
      weekday: "long",
      month: "long",
      day: "numeric",
      timeZone: tz,
    }).format(d);
  } catch {
    return new Intl.DateTimeFormat(undefined, {
      weekday: "long",
      month: "long",
      day: "numeric",
    }).format(d);
  }
}

function DailyPanel({
  daily,
  setDaily,
  selectedDate,
  setSelectedDate,
}: {
  daily: DailyLoad;
  setDaily: (updater: (prev: DailyLoad) => DailyLoad) => void;
  selectedDate: string;
  setSelectedDate: (date: string) => void;
}) {
  const dateNav = (
    <nav className={styles.dailyNav} aria-label="Date navigation">
      <button
        type="button"
        className={styles.dailyNavBtn}
        aria-label="Previous day"
        onClick={() => setSelectedDate(offsetDate(selectedDate, -1))}
      >
        ←
      </button>
      <input
        type="date"
        className={styles.dailyDateInput}
        value={selectedDate}
        onChange={(e) => {
          if (e.target.value) setSelectedDate(e.target.value);
        }}
      />
      <button
        type="button"
        className={styles.dailyNavBtn}
        aria-label="Next day"
        onClick={() => setSelectedDate(offsetDate(selectedDate, 1))}
      >
        →
      </button>
    </nav>
  );

  if (daily.kind === "loading" || daily.kind === "idle") {
    return <div className={styles.spinner} aria-label="loading" />;
  }
  if (daily.kind === "empty") {
    return (
      <div className={styles.dailyContainer}>
        {dateNav}
        <div className={styles.dailyMessage}>No reading for this day.</div>
      </div>
    );
  }
  if (daily.kind === "error") {
    return (
      <div className={styles.dailyContainer}>
        {dateNav}
        <div className={styles.dailyMessage}>
          Daily reading unavailable. Try again later.
        </div>
      </div>
    );
  }

  const { state } = daily;
  const activeState = state.active === "OT" ? state.ot : state.nt;
  const pills: Testament[] = [];
  if (state.ot) pills.push("OT");
  if (state.nt) pills.push("NT");

  return (
    <div className={styles.dailyContainer}>
      <header className={styles.dailyHeader}>
        {dateNav}
        <div className={styles.dailyDate}>{formatDate(selectedDate)}</div>
        <div className={styles.dailyPlan}>Bible in One Year</div>
      </header>
      <nav
        className={styles.dailyPills}
        role="tablist"
        aria-label="Today's readings"
      >
        {pills.map((t) => {
          const slot = t === "OT" ? state.ot : state.nt;
          if (!slot) return null;
          const active = state.active === t;
          return (
            <button
              key={t}
              type="button"
              role="tab"
              aria-selected={active}
              className={
                active
                  ? `${styles.dailyPill} ${styles.dailyPillActive}`
                  : styles.dailyPill
              }
              onClick={() =>
                setDaily((prev) =>
                  prev.kind === "ready"
                    ? { ...prev, state: { ...prev.state, active: t } }
                    : prev,
                )
              }
            >
              <span className={styles.dailyPillRef}>
                {slot.passage.book} {formatChapters(slot.passage.chapters)}
              </span>
              <span className={styles.dailyPillTestament}>{t}</span>
            </button>
          );
        })}
      </nav>
      <div className={styles.dailyBody}>
        {activeState?.loading && (
          <div className={styles.spinner} aria-label="loading" />
        )}
        {activeState && !activeState.loading && activeState.html && (
          <article
            className="passage"
            dangerouslySetInnerHTML={{ __html: activeState.html }}
          />
        )}
        {activeState?.error && (
          <div className={styles.toast} role="alert">
            {activeState.error}
          </div>
        )}
      </div>
    </div>
  );
}
