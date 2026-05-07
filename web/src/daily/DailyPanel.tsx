import type { RefObject } from "react";
import { PassageView } from "../highlights/PassageView";
import { defaultTimezoneProvider } from "../platform/TimezoneProvider";
import { Attribution } from "../translations/Attribution";
import { type UpdateTranslationResult } from "../translations/api";
import { TRANSLATIONS, type TranslationID } from "../translations/catalog";
import styles from "../App.module.css";
import type { PlanID } from "./plans";
import type { DailyChapterState, DailyLoad } from "./useDailyTab";

export type DailyPanelProps = {
  daily: DailyLoad;
  selectedDate: string;
  setSelectedDate: (date: string) => void;
  setActivePill: (idx: number) => void;
  showWordsOfChrist: boolean;
  translation: TranslationID;
  setTranslation: (id: TranslationID) => Promise<UpdateTranslationResult>;
  isSignedIn: boolean;
  onGuestSignin: () => void;
  getArticleRef: (
    planID: PlanID,
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
  const { pills, planMessages, activeIdx, planCount } = state;
  const activePill = pills[activeIdx] ?? null;
  const showPlanTagOnPills = planCount > 1;

  const pillsNav = (label: string) =>
    pills.length === 0 ? null : (
      <nav className={styles.dailyPills} role="tablist" aria-label={label}>
        {pills.map((pill, idx) => {
          const active = idx === activeIdx;
          const tagText = showPlanTagOnPills
            ? pill.planName
            : pill.passage.testament;
          return (
            <button
              key={`${pill.planID}|${pill.passage.book}|${pill.passage.chapters}`}
              type="button"
              role="tab"
              aria-selected={active}
              className={
                active
                  ? `${styles.dailyPill} ${styles.dailyPillActive}`
                  : styles.dailyPill
              }
              onClick={() => setActivePill(idx)}
            >
              <span className={styles.dailyPillRef}>
                {pill.passage.book} {formatChapters(pill.passage.chapters)}
              </span>
              {tagText && (
                <span className={styles.dailyPillTestament}>{tagText}</span>
              )}
            </button>
          );
        })}
      </nav>
    );

  const noContent = pills.length === 0 && planMessages.length === 0;

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
      </header>
      {planMessages.length > 0 && (
        <div className={styles.dailyInfoCards}>
          {planMessages.map((m) => (
            <div
              key={`${m.planID}|${m.text}`}
              className={styles.dailyInfoCard}
              role="note"
            >
              {planCount > 1 && (
                <span className={styles.dailyInfoCardPlan}>{m.planName}</span>
              )}
              {m.text}
            </div>
          ))}
        </div>
      )}
      {noContent && (
        <div className={styles.dailyMessage}>No reading for this day.</div>
      )}
      {pillsNav("Today's readings")}
      <div className={styles.dailyBody}>
        {activePill && (
          <div className={styles.dailyChapters}>
            {activePill.chapters.map((c) => (
              <DailyChapterBlock
                key={`${activePill.planID}|${c.book}:${c.chapter}`}
                chapterState={c}
                translation={translation}
                isSignedIn={isSignedIn}
                showWordsOfChrist={showWordsOfChrist}
                onGuestSignin={onGuestSignin}
                articleRef={getArticleRef(activePill.planID, c.book, c.chapter)}
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
  articleRef,
}: {
  chapterState: DailyChapterState;
  translation: TranslationID;
  isSignedIn: boolean;
  showWordsOfChrist: boolean;
  onGuestSignin: () => void;
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
