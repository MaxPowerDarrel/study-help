import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import {
  CANON,
  ChapterRef,
  nextChapter,
  prevChapter,
  refToQuery,
} from "./canon";
import { fetchPassage } from "./api";
import { useToggles } from "./toggles";
import { useTheme } from "./theme";
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
import { TRANSLATIONS, type TranslationID } from "./translations/catalog";
import { useTranslation } from "./translations/useTranslation";
import { DailyPanel } from "./daily/DailyPanel";
import { useDailyTab } from "./daily/useDailyTab";
import styles from "./App.module.css";

type Tab = "read" | "daily";

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
  const readingSurfaceRef = useRef<HTMLElement | null>(null);
  const userState = useUser();
  const translationState = useTranslation(userState.user, userState.applyUser);
  const translation = translationState.translation;
  const dailyTab = useDailyTab(tab === "daily", toggles, translation);

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

  const next = nextChapter(ref);
  const prev = prevChapter(ref);
  const book = CANON[ref.bookIndex];

  const isSignedIn = userState.user !== null;

  // Read tab uses per-chapter useNotes; Daily aggregates per-pill chapters
  // via useDailyNotes. Both are called unconditionally; the active surface
  // routes drawer mutations to the right hook.
  const readNotes = useNotes(book.name, ref.chapter, translation, isSignedIn);
  const dailyNotes = useDailyNotes(
    dailyTab.activeTab?.passage.book ?? null,
    dailyTab.activeChapterNumbers,
    translation,
    isSignedIn,
  );
  const notesApi = tab === "daily" ? dailyNotes : readNotes;
  const drawerTitle =
    tab === "daily"
      ? dailyTab.activeTab
        ? `${dailyTab.activeTab.passage.book} ${dailyTab.activeTab.passage.chapters}`
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

  const activePill =
    dailyTab.daily.kind === "ready" ? dailyTab.daily.state.active : null;
  useEffect(() => {
    if (tab !== "daily" || !activePill) return;
    readingSurfaceRef.current?.scrollTo({ top: 0, behavior: "smooth" });
  }, [activePill, tab]);

  const scrollToNote = (n: Note) => {
    if (tab === "daily") {
      const switched = dailyTab.switchToOwningPill(n.book, n.chapter);
      if (switched) {
        // Wait for the inactive pill's chapter blocks to mount, then
        // scroll. Two animation frames is enough for React's commit +
        // layout.
        requestAnimationFrame(() => {
          requestAnimationFrame(() => {
            const el = dailyTab.getArticleEl(n.book, n.chapter);
            if (el) scrollToInArticle(el, n);
          });
        });
        return;
      }
      const el = dailyTab.getArticleEl(n.book, n.chapter);
      if (el) scrollToInArticle(el, n);
      return;
    }
    if (articleRef.current) scrollToInArticle(articleRef.current, n);
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

        <main ref={readingSurfaceRef} className={styles.readingSurface}>
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
              daily={dailyTab.daily}
              selectedDate={dailyTab.selectedDate}
              setSelectedDate={dailyTab.setSelectedDate}
              setActivePill={dailyTab.setActivePill}
              showWordsOfChrist={toggles.include_word_of_christ}
              translation={translation}
              setTranslation={translationState.setTranslation}
              isSignedIn={isSignedIn}
              onGuestSignin={() => setAuthOpen(true)}
              onAddNote={handleAddNote}
              getArticleRef={dailyTab.getArticleRef}
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
