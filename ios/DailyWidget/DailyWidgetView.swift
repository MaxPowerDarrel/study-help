import SwiftUI
import WidgetKit

/// Renders one `DailyEntry`. Small = compact reference list; medium adds
/// testament chips and more breathing room. Tapping deep-links into the app.
struct DailyWidgetView: View {
    var entry: DailyEntry
    @Environment(\.widgetFamily) private var family

    private var deepLink: URL {
        var comps = URLComponents()
        comps.scheme = "studyhelp"
        comps.host = "daily"
        comps.queryItems = [URLQueryItem(name: "translation", value: entry.translation.rawValue)]
        return comps.url ?? URL(string: "studyhelp://daily")!
    }

    var body: some View {
        VStack(alignment: .leading, spacing: family == .systemSmall ? 5 : 7) {
            header
            content
            Spacer(minLength: 0)
        }
        .frame(maxWidth: .infinity, maxHeight: .infinity, alignment: .topLeading)
        .widgetURL(deepLink)
    }

    private var header: some View {
        HStack(spacing: 4) {
            Text(entry.planName)
                .font(.caption)
                .fontWeight(.semibold)
                .foregroundStyle(.secondary)
                .lineLimit(1)
            Spacer(minLength: 0)
            if entry.isStale {
                Image(systemName: "arrow.clockwise.circle")
                    .font(.caption2)
                    .foregroundStyle(.tertiary)
            }
        }
    }

    @ViewBuilder
    private var content: some View {
        if let message = entry.message, entry.passages.isEmpty {
            Text(message)
                .font(.subheadline)
                .foregroundStyle(.primary)
        } else if entry.passages.isEmpty {
            Text("No reading today")
                .font(.subheadline)
                .foregroundStyle(.secondary)
        } else {
            ForEach(entry.passages, id: \.display) { passage in
                HStack(spacing: 6) {
                    if family != .systemSmall, !passage.testament.isEmpty {
                        Text(passage.testament)
                            .font(.caption2)
                            .fontWeight(.bold)
                            .foregroundStyle(.secondary)
                            .padding(.horizontal, 5)
                            .padding(.vertical, 1)
                            .background(.quaternary, in: Capsule())
                    }
                    Text(passage.display)
                        .font(family == .systemSmall ? .footnote : .body)
                        .fontWeight(.medium)
                        .lineLimit(2)
                        .minimumScaleFactor(0.8)
                }
            }
        }
    }
}
