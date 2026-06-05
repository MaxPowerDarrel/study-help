import WidgetKit

/// One rendered snapshot of the daily reading for the widget timeline.
struct DailyEntry: TimelineEntry {
    /// Timeline entry time (WidgetKit ordering key).
    let date: Date
    /// The `YYYY-MM-DD` the references are for.
    let readingDate: String
    let planName: String
    let passages: [DailyPassage]
    /// Set on special / catch-up / no-reading days; passages is empty then.
    let message: String?
    /// True when the network fetch failed and we fell back to the cache.
    let isStale: Bool
    /// Translation chosen in the widget config; feeds the deep link only.
    let translation: TranslationOption
}
