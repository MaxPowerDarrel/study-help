import { useEffect, useMemo, useState } from "react";
import {
  CANON,
  ChapterRef,
  nextChapter,
  prevChapter,
  refToQuery,
} from "./canon";
import { fetchPassage } from "./api";
import { useToggles } from "./toggles";

export function App() {
  const [ref, setRef] = useState<ChapterRef>({ bookIndex: 42, chapter: 3 }); // John 3
  const [range, setRange] = useState<{ start: number; end: number } | null>(null);
  const [toggles, setToggles] = useToggles();
  const [html, setHtml] = useState<string>("");
  const [loading, setLoading] = useState(false);
  const [toast, setToast] = useState<string | null>(null);

  const q = useMemo(() => refToQuery(ref, range ?? undefined), [ref, range]);

  useEffect(() => {
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
  }, [q, toggles]);

  const next = nextChapter(ref);
  const prev = prevChapter(ref);
  const book = CANON[ref.bookIndex];

  return (
    <div className="layout">
      <aside className="picker">
        <h1>study-help</h1>
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
            {Array.from({ length: book.chapters }, (_, i) => i + 1).map((c) => (
              <option key={c} value={c}>
                {c}
              </option>
            ))}
          </select>
        </label>

        <fieldset className="range">
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
              setRange({ start, end: range?.end && range.end >= start ? range.end : start });
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
            <button type="button" onClick={() => setRange(null)}>
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

      <main className="reading-surface">
        {loading && <div className="spinner" aria-label="loading" />}
        {!loading && html && (
          <article
            className="passage"
            // ESV's HTML payload is forwarded verbatim; attribution
            // markup is included by ESV. See passage-reader.md
            // Decisions, 2026-05-02.
            dangerouslySetInnerHTML={{ __html: html }}
          />
        )}
        {toast && (
          <div className="toast" role="alert">
            {toast}
          </div>
        )}
      </main>
    </div>
  );
}
