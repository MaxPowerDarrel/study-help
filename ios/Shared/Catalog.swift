import AppIntents

/// Reading plans, as a static catalog matching the server's plan registry
/// (`internal/dailyreader/dailyreader.go`). `rawValue` is the exact `plans=`
/// query value the API expects. Conforms to `AppEnum` so the widget's
/// long-press configuration can offer it as a picker.
enum PlanOption: String, AppEnum, CaseIterable {
    case bibleYear = "bible-year"
    case hope = "hope"

    var planName: String {
        switch self {
        case .bibleYear: return "Bible in One Year"
        case .hope: return "Hope (2026)"
        }
    }

    static var typeDisplayRepresentation: TypeDisplayRepresentation { "Reading Plan" }

    static var caseDisplayRepresentations: [PlanOption: DisplayRepresentation] {
        [
            .bibleYear: "Bible in One Year",
            .hope: "Hope (2026)",
        ]
    }
}

/// Translations, matching the server's translation catalog. The daily-reading
/// endpoint returns translation-independent references, so this only feeds the
/// widget's deep link (so a tap can open the chosen translation in-app once the
/// SPA honors a `translation` query param — see `specs/native-ios.md`).
enum TranslationOption: String, AppEnum, CaseIterable {
    case esv = "ESV"
    case niv = "NIV"

    static var typeDisplayRepresentation: TypeDisplayRepresentation { "Translation" }

    static var caseDisplayRepresentations: [TranslationOption: DisplayRepresentation] {
        [
            .esv: "ESV",
            .niv: "NIV",
        ]
    }
}
