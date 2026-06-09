import SwiftUI

/// The native reading surface: parsed blocks rendered as a scrollable stack
/// of `Text`s. Inline semantics (verse numbers, words of Christ, marker
/// links) come from `PassageRenderer`; this view owns block-level
/// presentation — fonts per block kind, poetry indentation, spacing —
/// approximating the web's `passage.css`.
struct PassageView: View {
    let parsed: ParsedPassage
    let showWordsOfChrist: Bool
    /// Anchor identity for scroll-to-top when a new passage loads.
    let contentID: String

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
        ScrollViewReader { proxy in
            ScrollView {
                VStack(alignment: .leading, spacing: 0) {
                    Color.clear.frame(height: 0).id("top")
                    ForEach(Array(parsed.blocks.enumerated()), id: \.offset) { _, block in
                        blockView(block)
                    }
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
