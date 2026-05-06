import type { RefObject } from "react";
import type { ToolbarTuple } from "../highlights/HighlightToolbar";
import { PassageView } from "../highlights/PassageView";
import { defaultTimezoneProvider } from "../platform/TimezoneProvider";
import { Attribution } from "../translations/Attribution";
import { type UpdateTranslationResult } from "../translations/api";
import { TRANSLATIONS, type TranslationID } from "../translations/catalog";
import styles from "../App.module.css";
import type { DailyChapterState, DailyLoad, Testament } from "./useDailyTab";

export type DailyPanelProps = {
  daily: DailyLoad;
  selectedDate: string;
  setSelectedDate: (date: string) => void;
  setActivePill: (t: Testament) => void;
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
};

export function DailyPanel({
  daily,
  selectedDate,
  setSelectedDate,
  setActivePill,
  showWordsOfChrist,
  translation,
  setTranslation,
  isSignedIn,
  onGuestSignin,
  onAddNote,
  getArticleRef,
}: DailyPanelProps) {
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

  const pillsNav = (label: string) => (
    <nav className={styles.dailyPills} role="tablist" aria-label={label}>
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
            onClick={() => setActivePill(t)}
          >
            <span className={styles.dailyPillRef}>
              {slot.passage.book} {formatChapters(slot.passage.chapters)}
            </span>
            <span className={styles.dailyPillTestament}>{t}</span>
          </button>
        );
      })}
    </nav>
  );

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
      {pillsNav("Today's readings")}
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
            {pills.length > 1 && pillsNav("Today's readings (bottom)")}
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

function formatChapters(chapters: string): string {
  return chapters.replace(/-/g, "–");
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
