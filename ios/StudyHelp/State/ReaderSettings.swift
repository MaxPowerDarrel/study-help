import Foundation
import Observation

/// Reading-surface appearance, mirroring the web's three-way theme
/// (`web/src/theme.ts`: light / dark / follow-the-system).
enum ThemeChoice: String, CaseIterable {
    case system
    case light
    case dark

    var label: String {
        switch self {
        case .system: return "System"
        case .light: return "Light"
        case .dark: return "Dark"
        }
    }
}

/// Reader preferences, persisted to the App Group `UserDefaults` — the
/// native equivalent of the web's localStorage-backed stores
/// (`web/src/toggles.ts`, `web/src/translations/store.ts`). Defaults match
/// the web client exactly: cross-references off, everything else on, ESV.
///
/// `useNativeReader` is the phase-③ beta switch: off by default, so the
/// app keeps opening to the embedded SPA until the native surface reaches
/// parity (`specs/native-reader.md`, Decisions 2026-06-09).
@Observable
final class ReaderSettings {
    private enum Keys {
        static let native = "reader.native"
        static let translation = "reader.translation"
        static let headings = "reader.toggle.headings"
        static let footnotes = "reader.toggle.footnotes"
        static let verseNumbers = "reader.toggle.verse-numbers"
        static let passageReferences = "reader.toggle.passage-references"
        static let crossReferences = "reader.toggle.cross-references"
        static let wordsOfChrist = "reader.toggle.words-of-christ"
        static let plans = "reader.plans"
        static let theme = "reader.theme"
    }

    @ObservationIgnored private let store: UserDefaults

    var useNativeReader: Bool {
        didSet { store.set(useNativeReader, forKey: Keys.native) }
    }
    var translation: TranslationOption {
        didSet { store.set(translation.rawValue, forKey: Keys.translation) }
    }
    var headings: Bool {
        didSet { store.set(headings, forKey: Keys.headings) }
    }
    var footnotes: Bool {
        didSet { store.set(footnotes, forKey: Keys.footnotes) }
    }
    var verseNumbers: Bool {
        didSet { store.set(verseNumbers, forKey: Keys.verseNumbers) }
    }
    var passageReferences: Bool {
        didSet { store.set(passageReferences, forKey: Keys.passageReferences) }
    }
    var crossReferences: Bool {
        didSet { store.set(crossReferences, forKey: Keys.crossReferences) }
    }
    /// Render-time only — flipping it re-renders without a re-fetch
    /// (`AttributedRendering.swift`), so it is not part of `passageToggles`.
    var showWordsOfChrist: Bool {
        didSet { store.set(showWordsOfChrist, forKey: Keys.wordsOfChrist) }
    }
    /// Light/dark/system override for the native surface (the web view
    /// keeps the web app's own theme).
    var theme: ThemeChoice {
        didSet { store.set(theme.rawValue, forKey: Keys.theme) }
    }
    /// Daily-tab plan selection, mirroring the web's `usePlanSelection`:
    /// persisted, and never empty — clearing the last plan snaps back to
    /// the default so the Daily tab can't be stuck blank.
    var selectedPlans: [PlanOption] {
        didSet {
            if selectedPlans.isEmpty {
                selectedPlans = [.bibleYear] // re-enters didSet; persists below
                return
            }
            store.set(selectedPlans.map(\.rawValue), forKey: Keys.plans)
        }
    }

    /// The five server-sent options, in the shape `PassageAPI` sends.
    var passageToggles: PassageToggles {
        PassageToggles(
            headings: headings,
            footnotes: footnotes,
            verseNumbers: verseNumbers,
            passageReferences: passageReferences,
            crossReferences: crossReferences
        )
    }

    init(store: UserDefaults = AppGroup.defaults) {
        self.store = store
        useNativeReader = store.bool(forKey: Keys.native, default: false)
        translation = store.string(forKey: Keys.translation)
            .flatMap(TranslationOption.init(rawValue:)) ?? .esv
        headings = store.bool(forKey: Keys.headings, default: true)
        footnotes = store.bool(forKey: Keys.footnotes, default: true)
        verseNumbers = store.bool(forKey: Keys.verseNumbers, default: true)
        passageReferences = store.bool(forKey: Keys.passageReferences, default: true)
        // Off by default: matches the ESV API default and the web UI.
        crossReferences = store.bool(forKey: Keys.crossReferences, default: false)
        showWordsOfChrist = store.bool(forKey: Keys.wordsOfChrist, default: true)
        theme = store.string(forKey: Keys.theme)
            .flatMap(ThemeChoice.init(rawValue:)) ?? .system
        let storedPlans = (store.stringArray(forKey: Keys.plans) ?? [])
            .compactMap(PlanOption.init(rawValue:))
        selectedPlans = storedPlans.isEmpty ? [.bibleYear] : storedPlans
    }
}

private extension UserDefaults {
    /// `bool(forKey:)` returns `false` for missing keys; this keeps the
    /// "absent means default" semantics the web's toggle store has.
    func bool(forKey key: String, default defaultValue: Bool) -> Bool {
        object(forKey: key) == nil ? defaultValue : bool(forKey: key)
    }
}
