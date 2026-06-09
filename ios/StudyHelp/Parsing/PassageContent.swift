import Foundation

/// Provider-neutral intermediate model between publisher HTML and rendered
/// `AttributedString`s. The parser (`PassageHTMLParser`) maps both markup
/// dialects — ESV (`verse-num`, `woc`, `a.cf`, …) and YouVersion/NIV
/// (`yv-vlbl`, `wj`, `q1`/`q2`, …) — onto these types; the renderer
/// (`AttributedRendering.swift`) never sees HTML.
struct ParsedPassage: Equatable {
    var blocks: [PassageBlock] = []
    var footnotes: [Footnote] = []
    /// CSS classes the parser did not recognize. Content under them still
    /// renders (never-drop rule) — this exists so tests can pin fixtures to
    /// "fully understood" and DEBUG builds can log upstream markup drift.
    var unknownClasses: Set<String> = []
}

struct PassageBlock: Equatable {
    enum Kind: Equatable {
        /// Passage reference label — ESV `h2.extra_text`, NIV `div.cl`.
        /// Present only when `include_passage_references` is on (ESV).
        case passageTitle
        /// Section heading — ESV `h3`, NIV `div.s1.yv-h`.
        case heading
        /// Sub-title line — ESV `h4.psalm-title`, NIV `div.d`
        /// (e.g. "A Psalm of David.").
        case subtitle
        /// Prose paragraph — ESV `p`, NIV `div.p`. Also the fallback for
        /// any unrecognized block, so unknown markup degrades to plain
        /// readable text instead of disappearing.
        case prose
        /// One line of poetry. ESV: `span.line` (indent 0) /
        /// `span.indent.line` (indent 1) inside `p.block-indent`.
        /// NIV: `div.q1` (indent 0) / `div.q2` (indent 1).
        case poetryLine(indent: Int)
    }

    var kind: Kind
    var runs: [InlineRun]

    /// Concatenated plain text — convenient for tests and accessibility.
    var text: String { runs.map(\.text).joined() }
}

/// A run of text with uniform inline semantics. Flags combine: ESV emits
/// `<b class="verse-num woc">`, which is both a verse number and
/// words-of-Christ.
struct InlineRun: Equatable {
    var text: String
    /// Verse/chapter label — ESV `b.verse-num` / `b.chapter-num`,
    /// NIV `span.yv-vlbl`. Rendered small, muted, superscript.
    var isVerseNumber = false
    /// ESV `span.woc`, NIV `span.wj`. Colored at render time iff the
    /// words-of-Christ toggle is on — the native equivalent of the web's
    /// `.passage.no-woc` CSS flip.
    var isWordsOfChrist = false
    var isItalic = false
    var isBold = false
    /// Divine name set in small caps — ESV `span.divine-name`, NIV `span.nd`.
    var isSmallCaps = false
    /// Generic superscript (ESV wraps `.cf` markers in a bare `<sup>`).
    var isSuperscript = false
    /// Bare reference list from an ESV `a.cf` marker's href, trailing
    /// slashes stripped (e.g. `"Psalm 103:11; Psalm 116:5"`) — the exact
    /// string `/api/crossref` accepts, mirroring `PassageArticle.tsx`.
    var crossRef: String?
    /// Footnote anchor id from an inline `a.fn` marker's href (`"f1-1"`),
    /// matching `Footnote.id`.
    var footnoteID: String?
    /// Real hyperlink (e.g. the ESV copyright link, preface links in notes).
    var link: URL?
}

/// One entry from the ESV footnotes tail (`div.footnotes`). The inline
/// marker carrying the matching `footnoteID` is the tap target; this is the
/// content a tap reveals.
struct Footnote: Equatable, Identifiable {
    /// Anchor id, e.g. `"f1-1"`.
    var id: String
    /// Marker label as shown inline, e.g. `"1"`.
    var label: String
    /// Verse the note annotates, e.g. `"23:2"` (ESV `span.footnote-ref`).
    var reference: String
    /// Note body with inline styling preserved.
    var runs: [InlineRun]

    var text: String { runs.map(\.text).joined() }
}
