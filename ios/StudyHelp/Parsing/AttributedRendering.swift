import SwiftUI

/// Custom URL schemes that make cross-reference and footnote markers
/// tappable for free: the renderer attaches them as `.link` attributes and
/// the reading view intercepts them via the `OpenURLAction` environment
/// (phase ⑤ of `specs/native-reader.md`). They never leave the process.
enum ReaderLink {
    static let crossrefScheme = "studyhelp-crossref"
    static let footnoteScheme = "studyhelp-footnote"

    /// e.g. `studyhelp-crossref://lookup?q=Psalm%20103:11;%20Psalm%20116:5`
    static func crossref(q: String) -> URL? {
        var comps = URLComponents()
        comps.scheme = crossrefScheme
        comps.host = "lookup"
        comps.queryItems = [URLQueryItem(name: "q", value: q)]
        return comps.url
    }

    /// e.g. `studyhelp-footnote://note?id=p0-f1-1`
    static func footnote(id: String) -> URL? {
        var comps = URLComponents()
        comps.scheme = footnoteScheme
        comps.host = "note"
        comps.queryItems = [URLQueryItem(name: "id", value: id)]
        return comps.url
    }

    static func crossrefQuery(from url: URL) -> String? {
        guard url.scheme == crossrefScheme,
              let comps = URLComponents(url: url, resolvingAgainstBaseURL: false)
        else { return nil }
        return comps.queryItems?.first(where: { $0.name == "q" })?.value
    }

    static func footnoteID(from url: URL) -> String? {
        guard url.scheme == footnoteScheme,
              let comps = URLComponents(url: url, resolvingAgainstBaseURL: false)
        else { return nil }
        return comps.queryItems?.first(where: { $0.name == "id" })?.value
    }
}

/// Colors for the inline semantics. Defaults approximate the web's design
/// tokens (`web/src/styles/tokens.css`); phase ④ swaps in Asset Catalog
/// colors with proper light/dark variants.
struct PassageRenderPalette {
    var verseNumber: Color = .secondary
    var wordsOfChrist: Color = Color(red: 0.72, green: 0.18, blue: 0.18)
    /// Cross-reference and footnote markers — interactive, so accent-colored
    /// like the web's `.cf` styling.
    var marker: Color = .accentColor
}

/// Renders `InlineRun`s into `AttributedString`. Block-level presentation
/// (fonts for headings, poetry indentation, spacing) is the view's job —
/// each block becomes its own `Text` in a stack; this type owns only the
/// inline semantics.
struct PassageRenderer {
    var palette = PassageRenderPalette()
    /// The words-of-Christ toggle, applied at render time — the native
    /// equivalent of the web's `.passage.no-woc` CSS flip. Re-rendering
    /// needs no re-fetch and no re-parse.
    var showWordsOfChrist = true
    var baseFont: Font = .body

    func attributedString(for block: PassageBlock) -> AttributedString {
        attributedString(for: block.runs)
    }

    func attributedString(for runs: [InlineRun]) -> AttributedString {
        var out = AttributedString()
        for run in runs {
            out += attributedString(for: run)
        }
        return out
    }

    private func attributedString(for run: InlineRun) -> AttributedString {
        var s = AttributedString(run.text)

        let isMarker = run.crossRef != nil || run.footnoteID != nil
        var font = baseFont
        if run.isVerseNumber || run.isSuperscript || isMarker {
            // Verse labels and markers render small and raised, matching
            // the web's superscript treatment.
            font = .footnote
            s.baselineOffset = 3
        }
        if run.isBold && !run.isVerseNumber { font = font.bold() }
        if run.isItalic { font = font.italic() }
        if run.isSmallCaps { font = font.smallCaps() }
        s.font = font

        if run.isVerseNumber {
            s.foregroundColor = palette.verseNumber
        }
        if run.isWordsOfChrist && showWordsOfChrist {
            s.foregroundColor = palette.wordsOfChrist
        }

        if let q = run.crossRef, let url = ReaderLink.crossref(q: q) {
            s.link = url
            s.foregroundColor = palette.marker
        } else if let id = run.footnoteID, let url = ReaderLink.footnote(id: id) {
            s.link = url
            s.foregroundColor = palette.marker
        } else if let url = run.link {
            s.link = url
        }

        return s
    }
}
