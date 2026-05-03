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
import { useAutoLoad } from "./autoload";
import { defaultTimezoneProvider } from "./platform/TimezoneProvider";
import { SettingsPane } from "./SettingsPane";

type Tab = "OT" | "NT";

type DailyTabState = {
  q: string;
  html: string;
  loading: boolean;
  error: string | null;
};

type DailyState = {
  ot: DailyTabState | null;
  nt: DailyTabState | null;
  activeTab: Tab;
};

export function App() {
  const [ref, setRef] = useState<ChapterRef>({ bookIndex: 42, chapter: 3 }); // John 3
  const [range, setRange] = useState<{ start: number; end: number } | null>(
    null,
  );
  const [toggles, setToggles] = useToggles();
  const [autoLoad, setAutoLoad] = useAutoLoad();
  const [html, setHtml] = useState<string>("");
  const [loading, setLoading] = useState(false);
  const [toast, setToast] = useState<string | null>(null);
  const [settingsOpen, setSettingsOpen] = useState(false);
  const [mode, setMode] = useState<"manual" | "daily">("manual");
  const [daily, setDaily] = useState<DailyState | null>(null);
  const startupRan = useRef(false);

  const q = useMemo(() => refToQuery(ref, range ?? undefined), [ref, range]);

  // One-shot startup: if auto-load is on, fetch today's reading and switch
  // to daily mode. Manual picker interactions later flip mode back.
  useEffect(() => {
    if (startupRan.current) return;
    startupRan.current = true;
    if (!autoLoad) return;

    const tz = defaultTimezoneProvider.get();
    fetchDailyReading(tz).then((result) => {
      if (result.kind === "error") {
        setToast("Daily reading unavailable; pick a passage manually.");
        return;
      }
      if (result.kind === "empty") {
        setToast("No reading for today");
        return;
      }
      setMode("daily");
      const ot = result.passages.find((p) => p.testament === "OT") ?? null;
      const nt = result.passages.find((p) => p.testament === "NT") ?? null;
      const initialTab: Tab = ot ? "OT" : "NT";
      const otState: DailyTabState | null = ot
        ? { q: assembleQ(ot), html: "", loading: true, error: null }
        : null;
      const ntState: DailyTabState | null = nt
        ? { q: assembleQ(nt), html: "", loading: true, error: null }
        : null;
      setDaily({ ot: otState, nt: ntState, activeTab: initialTab });

      if (ot) loadDailyTab(ot, "ot");
      if (nt) loadDailyTab(nt, "nt");
    });

    function loadDailyTab(p: DailyPassage, slot: "ot" | "nt") {
      const qStr = assembleQ(p);
      fetchPassage(qStr, toggles).then((res) => {
        setDaily((prev) => {
          if (!prev) return prev;
          const cur = prev[slot];
          if (!cur) return prev;
          if (res.kind === "ok") {
            return {
              ...prev,
              [slot]: { ...cur, loading: false, html: res.html },
            };
          }
          const errMsg =
            res.kind === "rate_limited"
              ? "Service is busy, try again in a moment"
              : "Something went wrong, try again";
          return {
            ...prev,
            [slot]: { ...cur, loading: false, error: errMsg },
          };
        });
      });
    }
    // Fire-and-forget; toggles snapshot is intentional.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // Manual-mode passage fetch: only runs when the user is in manual mode.
  useEffect(() => {
    if (mode !== "manual") return;
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
  }, [q, toggles, mode]);

  const next = nextChapter(ref);
  const prev = prevChapter(ref);
  const book = CANON[ref.bookIndex];

  function flipToManual(update: () => void) {
    setMode("manual");
    update();
  }

  return (
    <div className="app">
      <header className="app-header">
        <div className="app-title">study-help</div>
        <button
          type="button"
          className="gear"
          aria-label="Settings"
          onClick={() => setSettingsOpen((v) => !v)}
        >
          ⚙
        </button>
      </header>
      <SettingsPane
        open={settingsOpen}
        onClose={() => setSettingsOpen(false)}
        autoLoad={autoLoad}
        setAutoLoad={setAutoLoad}
      />
      <div className="layout">
        <aside className="picker">
          <label>
            Book
            <select
              value={ref.bookIndex}
              onChange={(e) =>
                flipToManual(() => {
                  setRef({ bookIndex: Number(e.target.value), chapter: 1 });
                  setRange(null);
                })
              }
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
              onChange={(e) =>
                flipToManual(() => {
                  setRef({ ...ref, chapter: Number(e.target.value) });
                  setRange(null);
                })
              }
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

          <fieldset className="range">
            <legend>Verse range</legend>
            <input
              type="number"
              min={1}
              placeholder="start"
              value={range?.start ?? ""}
              onChange={(e) =>
                flipToManual(() => {
                  const start = Number(e.target.value);
                  if (!start) {
                    setRange(null);
                    return;
                  }
                  setRange({
                    start,
                    end: range?.end && range.end >= start ? range.end : start,
                  });
                })
              }
            />
            <span>–</span>
            <input
              type="number"
              min={1}
              placeholder="end"
              value={range?.end ?? ""}
              disabled={!range}
              onChange={(e) =>
                flipToManual(() => {
                  if (!range) return;
                  const end = Number(e.target.value);
                  setRange({ start: range.start, end: end || range.start });
                })
              }
            />
            {range && (
              <button
                type="button"
                onClick={() => flipToManual(() => setRange(null))}
              >
                Clear
              </button>
            )}
          </fieldset>

          <fieldset className="toggles">
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

          <nav className="chapter-nav">
            {prev && (
              <button
                type="button"
                onClick={() =>
                  flipToManual(() => {
                    setRef(prev);
                    setRange(null);
                  })
                }
              >
                ← Previous
              </button>
            )}
            {next && (
              <button
                type="button"
                onClick={() =>
                  flipToManual(() => {
                    setRef(next);
                    setRange(null);
                  })
                }
              >
                Next →
              </button>
            )}
          </nav>
        </aside>

        <main className="reading-surface">
          {mode === "manual" ? (
            <>
              {loading && <div className="spinner" aria-label="loading" />}
              {!loading && html && (
                <article
                  className="passage"
                  dangerouslySetInnerHTML={{ __html: html }}
                />
              )}
            </>
          ) : (
            daily && <DailyView state={daily} setState={setDaily} />
          )}
          {toast && (
            <div className="toast" role="alert">
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

function DailyView({
  state,
  setState,
}: {
  state: DailyState;
  setState: (updater: (prev: DailyState | null) => DailyState | null) => void;
}) {
  const tabs: Tab[] = [];
  if (state.ot) tabs.push("OT");
  if (state.nt) tabs.push("NT");
  const active = state.activeTab;
  const activeState = active === "OT" ? state.ot : state.nt;

  return (
    <>
      <nav className="ot-nt-tabs" role="tablist">
        {tabs.map((t) => (
          <button
            key={t}
            type="button"
            role="tab"
            aria-selected={active === t}
            className={active === t ? "tab active" : "tab"}
            onClick={() =>
              setState((prev) => (prev ? { ...prev, activeTab: t } : prev))
            }
          >
            {t}
          </button>
        ))}
      </nav>
      {activeState?.loading && <div className="spinner" aria-label="loading" />}
      {activeState && !activeState.loading && activeState.html && (
        <article
          className="passage"
          dangerouslySetInnerHTML={{ __html: activeState.html }}
        />
      )}
      {activeState?.error && (
        <div className="toast" role="alert">
          {activeState.error}
        </div>
      )}
    </>
  );
}
