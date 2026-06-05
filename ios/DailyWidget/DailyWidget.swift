import SwiftUI
import WidgetKit

struct DailyWidget: Widget {
    let kind = "DailyWidget"

    var body: some WidgetConfiguration {
        AppIntentConfiguration(
            kind: kind,
            intent: ConfigurationAppIntent.self,
            provider: Provider()
        ) { entry in
            DailyWidgetView(entry: entry)
                .containerBackground(.fill.tertiary, for: .widget)
        }
        .configurationDisplayName("Daily Reading")
        .description("Today's reading assignment.")
        .supportedFamilies([.systemSmall, .systemMedium])
    }
}
