import XCTest
@testable import StudyHelp

/// `ReaderSettings` defaults must match the web client's
/// (`web/src/toggles.ts`: cross-references off, everything else on, ESV),
/// and every preference must survive a "relaunch" (a fresh instance over
/// the same store).
final class ReaderSettingsTests: XCTestCase {
    private let suiteName = "ReaderSettingsTests"
    private var store: UserDefaults!

    override func setUp() {
        super.setUp()
        store = UserDefaults(suiteName: suiteName)
        store.removePersistentDomain(forName: suiteName)
    }

    override func tearDown() {
        store.removePersistentDomain(forName: suiteName)
        super.tearDown()
    }

    func testDefaultsMatchWebClient() {
        let settings = ReaderSettings(store: store)
        XCTAssertFalse(settings.useNativeReader, "beta switch must default off")
        XCTAssertEqual(settings.translation, .esv)
        XCTAssertTrue(settings.headings)
        XCTAssertTrue(settings.footnotes)
        XCTAssertTrue(settings.verseNumbers)
        XCTAssertTrue(settings.passageReferences)
        XCTAssertFalse(settings.crossReferences, "cross-references default off")
        XCTAssertTrue(settings.showWordsOfChrist)
    }

    func testPreferencesSurviveRelaunch() {
        let first = ReaderSettings(store: store)
        first.useNativeReader = true
        first.translation = .niv
        first.footnotes = false
        first.crossReferences = true
        first.showWordsOfChrist = false

        let second = ReaderSettings(store: store)
        XCTAssertTrue(second.useNativeReader)
        XCTAssertEqual(second.translation, .niv)
        XCTAssertFalse(second.footnotes)
        XCTAssertTrue(second.crossReferences)
        XCTAssertFalse(second.showWordsOfChrist)
        // Untouched values keep their defaults.
        XCTAssertTrue(second.headings)
        XCTAssertTrue(second.verseNumbers)
    }

    func testPassageTogglesExcludeWordsOfChrist() {
        let settings = ReaderSettings(store: store)
        settings.showWordsOfChrist = false
        // The five server-sent params are unaffected by the render-time
        // woc flag — flipping it must not change the fetch key.
        XCTAssertEqual(settings.passageToggles, PassageToggles())
    }

    func testCorruptTranslationFallsBackToESV() {
        store.set("KJV", forKey: "reader.translation")
        XCTAssertEqual(ReaderSettings(store: store).translation, .esv)
    }

    // MARK: Theme (mirrors web/src/theme.ts)

    func testThemeDefaultsToSystemAndPersists() {
        let settings = ReaderSettings(store: store)
        XCTAssertEqual(settings.theme, .system)

        settings.theme = .dark
        XCTAssertEqual(ReaderSettings(store: store).theme, .dark)
    }

    func testCorruptThemeFallsBackToSystem() {
        store.set("sepia", forKey: "reader.theme")
        XCTAssertEqual(ReaderSettings(store: store).theme, .system)
    }

    // MARK: Plan selection (mirrors web/src/daily/usePlanSelection.ts)

    func testPlansDefaultToBibleYear() {
        XCTAssertEqual(ReaderSettings(store: store).selectedPlans, [.bibleYear])
    }

    func testPlansPersistAcrossRelaunch() {
        let first = ReaderSettings(store: store)
        first.selectedPlans = [.hope]
        XCTAssertEqual(ReaderSettings(store: store).selectedPlans, [.hope])

        first.selectedPlans = [.bibleYear, .hope]
        XCTAssertEqual(ReaderSettings(store: store).selectedPlans, [.bibleYear, .hope])
    }

    func testClearingLastPlanFallsBackToDefault() {
        let settings = ReaderSettings(store: store)
        settings.selectedPlans = [.hope]
        settings.selectedPlans = []
        XCTAssertEqual(settings.selectedPlans, [.bibleYear], "never-empty fallback")
        // And the fallback is what persisted.
        XCTAssertEqual(ReaderSettings(store: store).selectedPlans, [.bibleYear])
    }

    func testUnknownStoredPlansFallBackToDefault() {
        store.set(["not-a-plan"], forKey: "reader.plans")
        XCTAssertEqual(ReaderSettings(store: store).selectedPlans, [.bibleYear])
    }
}
