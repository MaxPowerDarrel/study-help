import { defaultTimezoneProvider } from "../platform/TimezoneProvider";
import { Attribution } from "../translations/Attribution";
import { TRANSLATIONS, type TranslationID } from "../translations/catalog";
import styles from "../App.module.css";
import type { DailyLoad } from "./useDailyTab";

export type DailyPanelProps = {
  daily: DailyLoad;
  selectedDate: string;
  setSelectedDate: (date: string) => void;
  setActivePill: (idx: number) => void;
  showWordsOfChrist: boolean;
  translation: TranslationID;
  setTranslation: (id: TranslationID) => void;
};

export function DailyPanel({
  daily,
  selectedDate,
  setSelectedDate,
  setActivePill,
  showWordsOfChrist,
  translation,
  setTranslation,
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
          <>
            {activePill.loading && (
              <div className={styles.spinner} aria-label="loading" />
            )}
            {!activePill.loading && activePill.html && (
              <article
                className={showWordsOfChrist ? "passage" : "passage no-woc"}
                dangerouslySetInnerHTML={{ __html: activePill.html }}
              />
            )}
            {activePill.error && (
              <div className={styles.dailyMessage} role="alert">
                {activePill.error}
              </div>
            )}
            <Attribution translation={translation} />
            {pills.length > 1 && pillsNav("Today's readings (bottom)")}
          </>
        )}
      </div>
    </div>
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
