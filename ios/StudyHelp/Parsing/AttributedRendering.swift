import SwiftUI
import UIKit

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

/// Colors for the inline semantics, mirroring the web's design tokens
/// (`web/src/styles/tokens.css`) in both appearances:
/// `--color-woc` #c0392b / #ff6b5b, `--color-accent` #4a3a2a / #d4c8a8.
struct PassageRenderPalette {
    var verseNumber: Color = .secondary
    var wordsOfChrist: Color = Color(uiColor: UIColor { trait in
        trait.userInterfaceStyle == .dark
            ? UIColor(red: 255 / 255, green: 107 / 255, blue: 91 / 255, alpha: 1)
            : UIColor(red: 192 / 255, green: 57 / 255, blue: 43 / 255, alpha: 1)
    })
    /// Cross-reference and footnote markers — interactive, so accent-colored
    /// like the web's `.cf` styling.
    var marker: Color = Color(uiColor: UIColor { trait in
        trait.userInterfaceStyle == .dark
            ? UIColor(red: 212 / 255, green: 200 / 255, blue: 168 / 255, alpha: 1)
            : UIColor(red: 74 / 255, green: 58 / 255, blue: 42 / 255, alpha: 1)
    })
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
        for (i, run) in runs.enumerated() {
            out += attributedString(for: run)
            if needsSeparator(after: run, before: i + 1 < runs.count ? runs[i + 1] : nil) {
                out += AttributedString("\u{2009}") // thin space
            }
        }
        return out
    }

    /// YouVersion emits verse labels with no following space (`<span
    /// class="yv-vlbl">1</span>Now…`); the web compensates with a CSS
    /// `margin-right` on the label. ESV labels carry their own trailing
    /// `&nbsp;`. Insert a thin space only where the markup didn't.
    private func needsSeparator(after run: InlineRun, before next: InlineRun?) -> Bool {
        guard run.isVerseNumber, let next, !next.isVerseNumber else { return false }
        guard let last = run.text.last, let first = next.text.first else { return false }
        return !last.isWhitespace && last != "\u{00A0}"
            && !first.isWhitespace && first != "\u{00A0}"
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
