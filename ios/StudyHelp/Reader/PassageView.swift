import SwiftUI

/// The native reading surface: parsed blocks in a scroll view, plus the
/// marker-tap behavior — cross-reference markers open `CrossRefSheet`
/// (the native counterpart of the web's `PassageArticle` popover) and
/// footnote markers open `FootnoteSheet`. The markers arrive as
/// `studyhelp-crossref://` / `studyhelp-footnote://` links from the
/// renderer and are intercepted here via `OpenURLAction`; real links
/// (e.g. ESV's attribution) fall through to the system.
struct PassageView: View {
    let parsed: ParsedPassage
    let showWordsOfChrist: Bool
    /// Anchor identity for scroll-to-top when a new passage loads.
    let contentID: String
    var api = PassageAPI()

    @State private var crossref: CrossrefLookup?
    @State private var activeFootnote: Footnote?

    var body: some View {
        ScrollViewReader { proxy in
            ScrollView {
                VStack(alignment: .leading, spacing: 0) {
                    Color.clear.frame(height: 0).id("top")
                    PassageBlocksView(blocks: parsed.blocks, showWordsOfChrist: showWordsOfChrist)
                }
                .frame(maxWidth: 600, alignment: .leading)
                .frame(maxWidth: .infinity)
                .padding(.horizontal)
                .padding(.bottom, 24)
            }
            .onChange(of: contentID) {
                proxy.scrollTo("top", anchor: .top)
            }
        }
        .environment(\.openURL, OpenURLAction { url in
            if let q = ReaderLink.crossrefQuery(from: url) {
                let lookup = CrossrefLookup(api: api)
                crossref = lookup
                Task { await lookup.load(q: q) }
                return .handled
            }
            if let id = ReaderLink.footnoteID(from: url) {
                if let note = parsed.footnotes.first(where: { $0.id == id }) {
                    activeFootnote = note
                }
                return .handled
            }
            return .systemAction
        })
        .sheet(item: $crossref) { lookup in
            CrossRefSheet(lookup: lookup, showWordsOfChrist: showWordsOfChrist)
        }
        .sheet(item: $activeFootnote) { note in
            FootnoteSheet(note: note)
        }
    }
}

/// Block-level presentation shared by the main reading surface and the
/// cross-reference sheet: fonts per block kind, poetry indentation, and
/// spacing approximating the web's `passage.css`. Inline semantics come
/// from `PassageRenderer`.
struct PassageBlocksView: View {
    let blocks: [PassageBlock]
    let showWordsOfChrist: Bool

    private var bodyRenderer: PassageRenderer {
        PassageRenderer(
            showWordsOfChrist: showWordsOfChrist,
            baseFont: .system(.body, design: .serif)
        )
    }

    private var chromeRenderer: PassageRenderer {
        PassageRenderer(
            showWordsOfChrist: showWordsOfChrist,
            baseFont: .footnote
        )
    }

    var body: some View {
        ForEach(Array(blocks.enumerated()), id: \.offset) { _, block in
            blockView(block)
        }
    }

    @ViewBuilder
    private func blockView(_ block: PassageBlock) -> some View {
        switch block.kind {
        case .passageTitle:
            Text(chromeRenderer.attributedString(for: block))
                .font(.subheadline.weight(.semibold))
                .textCase(.uppercase)
                .kerning(0.8)
                .foregroundStyle(.secondary)
                .padding(.top, 24)
        case .heading:
            Text(chromeRenderer.attributedString(for: block))
                .font(.subheadline.weight(.semibold))
                .textCase(.uppercase)
                .kerning(0.8)
                .foregroundStyle(.secondary)
                .padding(.top, 20)
        case .subtitle:
            Text(bodyRenderer.attributedString(for: block))
                .italic()
                .foregroundStyle(.secondary)
                .padding(.top, 12)
        case .prose:
            Text(bodyRenderer.attributedString(for: block))
                .lineSpacing(6)
                .padding(.top, 12)
        case .poetryLine(let indent):
            Text(bodyRenderer.attributedString(for: block))
                .lineSpacing(4)
                .padding(.leading, CGFloat(indent) * 24)
                .padding(.top, 2)
        }
    }
}
