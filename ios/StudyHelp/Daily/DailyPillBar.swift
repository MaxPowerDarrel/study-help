import SwiftUI

/// Horizontal pill row for the day's readings — one pill per passage,
/// tagged with its testament when a single plan contributes and with the
/// plan name when several do (mirrors the web's `DailyPanel` pills).
struct DailyPillBar: View {
    let pills: [DailyViewModel.Pill]
    @Binding var activeIndex: Int
    let showPlanTag: Bool

    var body: some View {
        ScrollView(.horizontal, showsIndicators: false) {
            HStack(spacing: 8) {
                ForEach(Array(pills.enumerated()), id: \.element.id) { index, pill in
                    pillButton(pill, index: index)
                }
            }
            .padding(.horizontal)
            .padding(.vertical, 8)
        }
    }

    private func pillButton(_ pill: DailyViewModel.Pill, index: Int) -> some View {
        let active = index == activeIndex
        let tag = showPlanTag ? pill.planName : pill.passage.testament
        return Button {
            activeIndex = index
        } label: {
            VStack(spacing: 2) {
                Text("\(pill.passage.book) \(formatChapters(pill.passage.chapters))")
                    .font(.subheadline.weight(active ? .semibold : .regular))
                if !tag.isEmpty {
                    Text(tag)
                        .font(.caption2)
                        .foregroundStyle(active ? Color.accentColor : .secondary)
                }
            }
            .padding(.horizontal, 14)
            .padding(.vertical, 6)
            .background(
                active ? Color.accentColor.opacity(0.15) : Color(.secondarySystemBackground),
                in: Capsule()
            )
        }
        .buttonStyle(.plain)
        .accessibilityAddTraits(active ? [.isSelected] : [])
    }

    private func formatChapters(_ chapters: String) -> String {
        chapters
            .replacingOccurrences(of: "-", with: "\u{2013}")
            .replacingOccurrences(of: ",", with: ", ")
    }
}
