import XCTest
@testable import StudyHelp

/// Session restoration — mirrors the web's `restore.test.ts`: defaults
/// when absent, round-trips, corrupt/out-of-range fallbacks, and the
/// daily date's snap-to-today across calendar days.
final class SessionRestoreTests: XCTestCase {
    private let suiteName = "SessionRestoreTests"
    private var store: UserDefaults!
    private var restore: SessionRestore!

    override func setUp() {
        super.setUp()
        store = UserDefaults(suiteName: suiteName)
        store.removePersistentDomain(forName: suiteName)
        restore = SessionRestore(store: store)
    }

    override func tearDown() {
        store.removePersistentDomain(forName: suiteName)
        super.tearDown()
    }

    // MARK: Active tab

    func testTabDefaultsToRead() {
        XCTAssertEqual(restore.storedTab(), .read)
    }

    func testTabRoundTrips() {
        restore.saveTab(.daily)
        XCTAssertEqual(restore.storedTab(), .daily)
        restore.saveTab(.read)
        XCTAssertEqual(restore.storedTab(), .read)
    }

    func testGarbageTabFallsBackToRead() {
        store.set("settings", forKey: "reader.active-tab")
        XCTAssertEqual(restore.storedTab(), .read)
    }

    // MARK: Read-tab passage

    func testReadRefDefaultsToJohn3() {
        XCTAssertEqual(restore.storedReadRef(), Canon.ChapterRef(bookIndex: 42, chapter: 3))
    }

    func testReadRefRoundTrips() {
        let ref = Canon.ChapterRef(bookIndex: 18, chapter: 117) // Psalm 117
        restore.saveReadRef(ref)
        XCTAssertEqual(restore.storedReadRef(), ref)
    }

    func testOutOfRangeReadRefFallsBackToDefault() {
        restore.saveReadRef(Canon.ChapterRef(bookIndex: 99, chapter: 1))
        XCTAssertEqual(restore.storedReadRef(), SessionRestore.defaultReadRef)

        restore.saveReadRef(Canon.ChapterRef(bookIndex: 0, chapter: 51)) // Genesis has 50
        XCTAssertEqual(restore.storedReadRef(), SessionRestore.defaultReadRef)
    }

    func testCorruptReadRefFallsBackToDefault() {
        store.set(Data("not json".utf8), forKey: "reader.read-location")
        XCTAssertEqual(restore.storedReadRef(), SessionRestore.defaultReadRef)
    }

    // MARK: Daily date

    func testDailyDateDefaultsToToday() {
        XCTAssertEqual(restore.storedDailyDate(today: "2026-06-09"), "2026-06-09")
    }

    func testDailyDateSurvivesWithinTheSameDay() {
        restore.saveDailyDate("2026-06-01", today: "2026-06-09")
        XCTAssertEqual(restore.storedDailyDate(today: "2026-06-09"), "2026-06-01")
    }

    func testDailyDateSnapsToTodayAcrossDays() {
        restore.saveDailyDate("2026-06-01", today: "2026-06-09")
        // Reopened the next day: the pinned date is expired.
        XCTAssertEqual(restore.storedDailyDate(today: "2026-06-10"), "2026-06-10")
    }

    func testCorruptDailyDateFallsBackToToday() {
        store.set(Data("{\"date\": 42}".utf8), forKey: "reader.daily-date")
        XCTAssertEqual(restore.storedDailyDate(today: "2026-06-09"), "2026-06-09")
    }
}
