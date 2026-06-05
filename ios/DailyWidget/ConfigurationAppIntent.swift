import AppIntents
import WidgetKit

/// Long-press → Edit Widget configuration. Lets the user pick which plan the
/// widget shows and which translation a tap should open. This is the widget's
/// own on-device config — it does not read the web app's `localStorage` (the
/// extension can't), and it introduces no server state.
struct ConfigurationAppIntent: WidgetConfigurationIntent {
    static var title: LocalizedStringResource { "Daily Reading" }
    static var description: IntentDescription {
        IntentDescription("Choose which reading plan to show on the widget.")
    }

    @Parameter(title: "Reading Plan", default: .bibleYear)
    var plan: PlanOption

    @Parameter(title: "Open in", default: .esv)
    var translation: TranslationOption
}
