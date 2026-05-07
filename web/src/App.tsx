import { useEffect, useMemo, useRef, useState } from "react";
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
import { Attribution } from "./translations/Attribution";
import { TRANSLATIONS, type TranslationID } from "./translations/catalog";
import { useTranslation } from "./translations/useTranslation";
import { DailyPanel } from "./daily/DailyPanel";
import { useDailyTab } from "./daily/useDailyTab";
import { usePlanSelection } from "./daily/usePlanSelection";
import styles from "./App.module.css";

type Tab = "read" | "daily";

export function App() {
  const [tab, setTab] = useState<Tab>("read");
  const [ref, setRef] = useState<ChapterRef>({ bookIndex: 42, chapter: 3 }); // John 3
  const [toggles, setToggles] = useToggles();
  const [theme, setTheme] = useTheme();
  const [html, setHtml] = useState<string>("");
  const [loading, setLoading] = useState(false);
  const [toast, setToast] = useState<string | null>(null);
  const [settingsOpen, setSettingsOpen] = useState(false);
  const readingSurfaceRef = useRef<HTMLElement | null>(null);
  const { translation, setTranslation } = useTranslation();
  const [planIDs, setPlanIDs] = usePlanSelection();
  const dailyTab = useDailyTab(tab === "daily", toggles, translation, planIDs);

  const q = useMemo(() => refToQuery(ref), [ref]);

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

  const activePillIdx =
    dailyTab.daily.kind === "ready" ? dailyTab.daily.state.activeIdx : -1;
  useEffect(() => {
    if (tab !== "daily" || activePillIdx < 0) return;
    readingSurfaceRef.current?.scrollTo({ top: 0, behavior: "smooth" });
  }, [activePillIdx, tab]);

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
        planIDs={planIDs}
        setPlanIDs={setPlanIDs}
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
                onChange={(e) => {
                  setTranslation(e.target.value as TranslationID);
                }}
              >
                {TRANSLATIONS.map((t) => (
                  <option key={t.id} value={t.id}>
                    {t.id}
                  </option>
                ))}
              </select>
              <Attribution translation={translation} />
            </label>
            <label>
              Book
              <select
                value={ref.bookIndex}
                onChange={(e) => {
                  setRef({ bookIndex: Number(e.target.value), chapter: 1 });
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

            <nav className={styles.chapterNav}>
              {prev && (
                <button
                  type="button"
                  className={styles.navBtn}
                  onClick={() => {
                    setRef(prev);
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
                <article
                  className={
                    toggles.include_word_of_christ
                      ? "passage"
                      : "passage no-woc"
                  }
                  dangerouslySetInnerHTML={{ __html: html }}
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
              setTranslation={setTranslation}
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
