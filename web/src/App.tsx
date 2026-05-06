import {
  type RefObject,
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react";
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
import type { ToolbarTuple } from "./highlights/HighlightToolbar";
import { tupleToRange } from "./highlights/parseSelection";
import { PassageView } from "./highlights/PassageView";
import { NotesDrawer } from "./notes/NotesDrawer";
import { type Note } from "./notes/api";
import { useNotes } from "./notes/useNotes";
import { useDailyNotes } from "./notes/useDailyNotes";
import { Attribution } from "./translations/Attribution";
import { type UpdateTranslationResult } from "./translations/api";
import { TRANSLATIONS, type TranslationID } from "./translations/catalog";
import { useTranslation } from "./translations/useTranslation";
import styles from "./App.module.css";

type Tab = "read" | "daily";
type Testament = "OT" | "NT";

type DailyChapterState = {
  book: string;
  chapter: number;
  html: string;
  loading: boolean;
  error: string | null;
};

type DailyTabState = {
  passage: DailyPassage;
  chapters: DailyChapterState[];
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

type PendingNote = { tuple: ToolbarTuple; book: string; chapter: number };

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
  const [notesOpen, setNotesOpen] = useState(false);
  const [pendingNote, setPendingNote] = useState<PendingNote | null>(null);
  const articleRef = useRef<HTMLElement | null>(null);
  // Per-chapter article refs for the Daily tab. Keyed by `${book}:${chapter}`;
  // populated lazily on first request, never pruned (entries are tiny ref
  // objects and old keys are harmless once their DOM unmounts).
  const dailyArticleRefs = useRef<Map<string, RefObject<HTMLElement | null>>>(
    new Map(),
  );
  const getDailyArticleRef = useCallback(
    (book: string, chapter: number): RefObject<HTMLElement | null> => {
      const key = `${book}:${chapter}`;
      const existing = dailyArticleRefs.current.get(key);
      if (existing) return existing;
      const ref: RefObject<HTMLElement | null> = { current: null };
      dailyArticleRefs.current.set(key, ref);
      return ref;
    },
    [],
  );
  const userState = useUser();
  const translationState = useTranslation(userState.user, userState.applyUser);
  const translation = translationState.translation;
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
    fetchPassage(q, toggles, translation).then((result) => {
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
  }, [q, toggles, tab, translation]);

  // Daily-tab lazy load: fires when daily.kind is "idle" (first activation,
  // or after a date / translation change). selectedDate and toggles are read
  // from closure rather than as dependencies — re-runs are triggered by
  // daily.kind transitioning to "idle" (via the reset effects below).
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
      const makeTabState = (p: DailyPassage | null): DailyTabState | null => {
        if (!p) return null;
        return {
          passage: p,
          chapters: parseChapterRange(p.chapters).map((c) => ({
            book: p.book,
            chapter: c,
            html: "",
            loading: true,
            error: null,
          })),
        };
      };
      const otState = makeTabState(ot);
      const ntState = makeTabState(nt);
      const active: Testament = ot ? "OT" : "NT";
      setDaily({ kind: "ready", state: { ot: otState, nt: ntState, active } });

      const snapshotToggles = toggles;
      if (otState) loadChapters(otState, "ot", snapshotToggles);
      if (ntState) loadChapters(ntState, "nt", snapshotToggles);
    });

    function loadChapters(
      s: DailyTabState,
      slot: "ot" | "nt",
      t: typeof toggles,
    ) {
      // Per-chapter concurrent fetches; each chapter renders as it arrives
      // (a slow chapter doesn't block earlier ones).
      s.chapters.forEach((c) => {
        fetchPassage(`${c.book} ${c.chapter}`, t, translation).then((res) => {
          if (dailyFetchId.current !== fetchId) return;
          setDaily((prev) => updateChapter(prev, slot, c.book, c.chapter, res));
        });
      });
    }
    // selectedDate, toggles, and translation are snapshot intentionally; see
    // comment above. Re-runs are gated by daily.kind === "idle".
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [tab, daily.kind]);

  // When the user picks a different date or switches translation, reset daily
  // to idle so the load effect above re-triggers with the new context.
  useEffect(() => {
    setDaily({ kind: "idle" });
  }, [selectedDate, translation]);

  const next = nextChapter(ref);
  const prev = prevChapter(ref);
  const book = CANON[ref.bookIndex];

  const isSignedIn = userState.user !== null;

  // Read tab uses per-chapter useNotes; Daily aggregates per-pill chapters
  // via useDailyNotes. Both are called unconditionally; the active surface
  // routes drawer mutations to the right hook.
  const readNotes = useNotes(book.name, ref.chapter, translation, isSignedIn);
  const dailyActiveTab = useMemo(() => activePillState(daily), [daily]);
  const dailyChapterNumbers = useMemo(
    () => dailyActiveTab?.chapters.map((c) => c.chapter) ?? [],
    [dailyActiveTab],
  );
  const dailyNotes = useDailyNotes(
    dailyActiveTab?.passage.book ?? null,
    dailyChapterNumbers,
    translation,
    isSignedIn,
  );
  const notesApi = tab === "daily" ? dailyNotes : readNotes;
  const drawerTitle =
    tab === "daily"
      ? dailyActiveTab
        ? `${dailyActiveTab.passage.book} ${dailyActiveTab.passage.chapters}`
        : "Daily"
      : `${book.name} ${ref.chapter}`;

  const handleAddNote = (
    tuple: ToolbarTuple,
    addBook: string,
    addChapter: number,
  ) => {
    setPendingNote({ tuple, book: addBook, chapter: addChapter });
    setNotesOpen(true);
  };

  const closeNotesDrawer = () => {
    setNotesOpen(false);
    setPendingNote(null);
  };

  const createPendingNote = async (body: string) => {
    if (!pendingNote) {
      return { kind: "error" as const };
    }
    const res = await notesApi.create({
      book: pendingNote.book,
      chapter: pendingNote.chapter,
      start_verse: pendingNote.tuple.start_verse,
      start_offset: pendingNote.tuple.start_offset,
      end_verse: pendingNote.tuple.end_verse,
      end_offset: pendingNote.tuple.end_offset,
      body,
    });
    if (res.kind === "ok") setPendingNote(null);
    return res;
  };

  // Resolve the article element for a note. Read tab returns the single
  // articleRef; Daily looks up the per-chapter ref. Returns null when the
  // chapter isn't currently rendered (e.g., note belongs to the other pill —
  // caller should auto-switch first).
  const findArticle = useCallback(
    (noteBook: string, noteChapter: number): HTMLElement | null => {
      if (tab === "read") return articleRef.current;
      const ref = dailyArticleRefs.current.get(`${noteBook}:${noteChapter}`);
      return ref?.current ?? null;
    },
    [tab],
  );

  const scrollToInArticle = useCallback(
    (article: HTMLElement, n: Note) => {
      const range = tupleToRange(
        {
          start_verse: n.start_verse,
          start_offset: n.start_offset,
          end_verse: n.end_verse,
          end_offset: n.end_offset,
        },
        article,
        translation,
      );
      range?.startContainer.parentElement?.scrollIntoView({
        behavior: "smooth",
        block: "center",
      });
    },
    [translation],
  );

  const scrollToNote = (n: Note) => {
    // Daily-tab tap may target a chapter that lives in the inactive pill
    // (e.g., note on Romans 1 while OT pill is active). Auto-switch first,
    // then scroll once the new pill's article is in the DOM.
    if (tab === "daily" && daily.kind === "ready") {
      const owning = findOwningTestament(daily.state, n.book, n.chapter);
      if (owning && daily.state.active !== owning) {
        setDaily((prev) =>
          prev.kind === "ready"
            ? { ...prev, state: { ...prev.state, active: owning } }
            : prev,
        );
        // Wait for the inactive pill's chapter blocks to mount, then scroll.
        // Two animation frames is enough for React's commit + layout.
        requestAnimationFrame(() => {
          requestAnimationFrame(() => {
            const el = findArticle(n.book, n.chapter);
            if (el) scrollToInArticle(el, n);
          });
        });
        return;
      }
    }
    const el = findArticle(n.book, n.chapter);
    if (el) scrollToInArticle(el, n);
  };

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
          {isSignedIn && (
            <button
              type="button"
              className={styles.notesToggle}
              onClick={() => setNotesOpen((v) => !v)}
            >
              Notes
            </button>
          )}
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
        toggles={toggles}
        setToggles={setToggles}
      />
      <AuthPanel
        open={authOpen}
        onClose={() => setAuthOpen(false)}
        signin={userState.signin}
        signup={userState.signup}
      />
      <NotesDrawer
        open={notesOpen}
        onClose={closeNotesDrawer}
        title={drawerTitle}
        notes={notesApi.notes}
        loading={notesApi.loading}
        pendingTuple={pendingNote?.tuple ?? null}
        onCancelPending={() => setPendingNote(null)}
        onCreate={createPendingNote}
        onUpdate={notesApi.update}
        onRemove={notesApi.remove}
        onScrollToAnchor={scrollToNote}
        onError={(msg) => setToast(msg)}
        showChapter={tab === "daily"}
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
              Translation
              <select
                value={translation}
                disabled={!isSignedIn}
                onChange={(e) => {
                  void translationState.setTranslation(
                    e.target.value as TranslationID,
                  );
                }}
              >
                {TRANSLATIONS.map((t) => (
                  <option key={t.id} value={t.id}>
                    {t.id}
                  </option>
                ))}
              </select>
              {!isSignedIn && (
                <span className={styles.translationHint}>
                  Sign in to choose
                </span>
              )}
              <Attribution translation={translation} />
            </label>
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
                  translation={translation}
                  isSignedIn={isSignedIn}
                  showWordsOfChrist={toggles.include_word_of_christ}
                  onGuestSignin={() => setAuthOpen(true)}
                  onAddNote={handleAddNote}
                  articleRef={articleRef}
                />
              )}
            </>
          ) : (
            <DailyPanel
              daily={daily}
              setDaily={setDaily}
              selectedDate={selectedDate}
              setSelectedDate={setSelectedDate}
              showWordsOfChrist={toggles.include_word_of_christ}
              translation={translation}
              setTranslation={translationState.setTranslation}
              isSignedIn={isSignedIn}
              onGuestSignin={() => setAuthOpen(true)}
              onAddNote={handleAddNote}
              getArticleRef={getDailyArticleRef}
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

function formatChapters(chapters: string): string {
  return chapters.replace(/-/g, "–");
}

// Expand a daily-reader chapters string ("1-3", "1", "1,3") into individual
// chapter numbers. Falls back to an empty array on unrecognized formats.
function parseChapterRange(chapters: string): number[] {
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
// Tolerates concurrent updates: if `prev` is no longer "ready" or the slot /
// chapter don't match, returns prev unchanged.
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

// Find which pill (OT or NT) contains the given chapter, so the drawer can
// auto-switch when a tap targets a chapter in the inactive pill.
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
  showWordsOfChrist,
  translation,
  setTranslation,
  isSignedIn,
  onGuestSignin,
  onAddNote,
  getArticleRef,
}: {
  daily: DailyLoad;
  setDaily: (updater: (prev: DailyLoad) => DailyLoad) => void;
  selectedDate: string;
  setSelectedDate: (date: string) => void;
  showWordsOfChrist: boolean;
  translation: TranslationID;
  setTranslation: (id: TranslationID) => Promise<UpdateTranslationResult>;
  isSignedIn: boolean;
  onGuestSignin: () => void;
  onAddNote: (tuple: ToolbarTuple, book: string, chapter: number) => void;
  getArticleRef: (
    book: string,
    chapter: number,
  ) => RefObject<HTMLElement | null>;
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
        <div className={styles.dailyHeaderTop}>
          {dateNav}
          <label className={styles.dailyTranslation}>
            <span className={styles.dailyTranslationLabel}>Translation</span>
            <select
              value={translation}
              disabled={!isSignedIn}
              onChange={(e) => {
                void setTranslation(e.target.value as TranslationID);
              }}
            >
              {TRANSLATIONS.map((t) => (
                <option key={t.id} value={t.id}>
                  {t.id}
                </option>
              ))}
            </select>
            {!isSignedIn && (
              <span className={styles.translationHint}>Sign in to choose</span>
            )}
          </label>
        </div>
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
        {activeState && (
          <div className={styles.dailyChapters}>
            {activeState.chapters.map((c) => (
              <DailyChapterBlock
                key={`${c.book}:${c.chapter}`}
                chapterState={c}
                translation={translation}
                isSignedIn={isSignedIn}
                showWordsOfChrist={showWordsOfChrist}
                onGuestSignin={onGuestSignin}
                onAddNote={onAddNote}
                articleRef={getArticleRef(c.book, c.chapter)}
              />
            ))}
            <Attribution translation={translation} />
          </div>
        )}
      </div>
    </div>
  );
}

function DailyChapterBlock({
  chapterState,
  translation,
  isSignedIn,
  showWordsOfChrist,
  onGuestSignin,
  onAddNote,
  articleRef,
}: {
  chapterState: DailyChapterState;
  translation: TranslationID;
  isSignedIn: boolean;
  showWordsOfChrist: boolean;
  onGuestSignin: () => void;
  onAddNote: (tuple: ToolbarTuple, book: string, chapter: number) => void;
  articleRef: RefObject<HTMLElement | null>;
}) {
  return (
    <section
      className={styles.dailyChapter}
      aria-label={`${chapterState.book} ${chapterState.chapter}`}
    >
      <h3 className={styles.dailyChapterHeading}>
        {chapterState.book} {chapterState.chapter}
      </h3>
      {chapterState.loading && (
        <div className={styles.dailyChapterSpinner} aria-label="loading" />
      )}
      {!chapterState.loading && chapterState.html && (
        <PassageView
          html={chapterState.html}
          book={chapterState.book}
          chapter={chapterState.chapter}
          translation={translation}
          isSignedIn={isSignedIn}
          showWordsOfChrist={showWordsOfChrist}
          onGuestSignin={onGuestSignin}
          onAddNote={onAddNote}
          articleRef={articleRef}
        />
      )}
      {chapterState.error && (
        <div className={styles.dailyChapterError} role="alert">
          {chapterState.error}
        </div>
      )}
    </section>
  );
}
