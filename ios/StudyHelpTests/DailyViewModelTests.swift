import XCTest
@testable import StudyHelp

/// Daily-tab state machine: pill/message construction from
/// `/api/daily-reading`, the comma-chapter fan-out (port of the web's
/// `passageQueries`), per-pill error handling, and date navigation.
@MainActor
final class DailyViewModelTests: XCTestCase {

    override func tearDown() {
        StubURLProtocol.handler = nil
        super.tearDown()
    }

    private func makeModel() -> DailyViewModel {
        let config = URLSessionConfiguration.ephemeral
        config.protocolClasses = [StubURLProtocol.self]
        let session = URLSession(configuration: config)
        let base = URL(string: "https://example.test")!
        return DailyViewModel(
            dailyAPI: DailyReadingAPI(baseURL: base, session: session),
            passageAPI: PassageAPI(baseURL: base, session: session)
        )
    }

    /// Routes the stub: daily endpoint gets `dailyJSON`; passage requests
    /// echo their `q` so tests can assert content and ordering.
    private func route(daily dailyJSON: String, capture: ((URL) -> Void)? = nil) {
        StubURLProtocol.handler = { request in
            guard let url = request.url else { return (500, Data()) }
            capture?(url)
            if url.path == "/api/daily-reading" {
                return (200, Data(dailyJSON.utf8))
            }
            let comps = URLComponents(url: url, resolvingAgainstBaseURL: false)
            let q = comps?.queryItems?.first(where: { $0.name == "q" })?.value ?? "?"
            let body = #"{"canonical":"\#(q)","passages":["<p>text of \#(q)</p>"]}"#
            return (200, Data(body.utf8))
        }
    }

    // MARK: passageQueries (port of useDailyTab.passageQueries)

    func testPassageQueriesFanOutCommaChapters() {
        func queries(_ book: String, _ chapters: String) -> [String] {
            DailyViewModel.passageQueries(
                for: DailyPassage(book: book, chapters: chapters, testament: "NT")
            )
        }
        XCTAssertEqual(queries("Matthew", "11,12"), ["Matthew 11", "Matthew 12"])
        XCTAssertEqual(queries("Genesis", "1-3"), ["Genesis 1-3"], "hyphen ranges stay one query")
        XCTAssertEqual(queries("Luke", " 1 , 2 "), ["Luke 1", "Luke 2"], "pieces are trimmed")
        XCTAssertEqual(queries("Jude", "1"), ["Jude 1"])
        XCTAssertEqual(queries("Jude", ""), [])
    }

    // MARK: Loading

    func testLoadBuildsPillsMessagesAndLoadsPassages() async throws {
        var dailyURL: URL?
        route(daily: """
        {"plans":[
          {"id":"bible-year","name":"Bible in One Year","passages":[
            {"book":"Numbers","chapters":"20-22","testament":"OT"},
            {"book":"Luke","chapters":"1","testament":"NT"}]},
          {"id":"hope","name":"Hope (2026)","passages":[],"message":"Catch-up day!"}
        ]}
        """) { url in
            if url.path == "/api/daily-reading" { dailyURL = url }
        }

        let model = makeModel()
        model.selectedDate = "2026-06-09"
        await model.load(translation: .esv, toggles: PassageToggles(), plans: [.bibleYear, .hope])

        XCTAssertEqual(model.phase, .ready)
        XCTAssertEqual(model.planCount, 2)
        XCTAssertEqual(model.activeIndex, 0)
        XCTAssertEqual(model.pills.map(\.passage.book), ["Numbers", "Luke"])
        XCTAssertEqual(model.planMessages.map(\.text), ["Catch-up day!"])

        // load() returns only once every pill has settled.
        for pill in model.pills {
            XCTAssertNil(pill.error)
            XCTAssertFalse(pill.isLoading)
        }
        let first = try XCTUnwrap(model.pills.first?.parsed)
        XCTAssertTrue(first.blocks.first?.text.contains("Numbers 20-22") ?? false)

        // The request carried the joined plans param and the date.
        let comps = URLComponents(url: try XCTUnwrap(dailyURL), resolvingAgainstBaseURL: false)
        let params = Dictionary(uniqueKeysWithValues: (comps?.queryItems ?? []).map { ($0.name, $0.value ?? "") })
        XCTAssertEqual(params["plans"], "bible-year,hope")
        XCTAssertEqual(params["date"], "2026-06-09")
        XCTAssertFalse((params["tz"] ?? "").isEmpty)
    }

    func testCommaChaptersConcatenateInQueryOrder() async throws {
        route(daily: """
        {"plans":[{"id":"bible-year","name":"Bible in One Year","passages":[
          {"book":"Matthew","chapters":"11,12","testament":"NT"}]}]}
        """)

        let model = makeModel()
        await model.load(translation: .esv, toggles: PassageToggles(), plans: [.bibleYear])

        XCTAssertEqual(model.pills.count, 1)
        let parsed = try XCTUnwrap(model.pills[0].parsed)
        XCTAssertEqual(parsed.blocks.map(\.text), ["text of Matthew 11", "text of Matthew 12"],
                       "fan-out results must concatenate in chapter order")
    }

    func testDailyEndpointFailureFailsTheDay() async {
        StubURLProtocol.handler = { _ in (500, Data()) }
        let model = makeModel()
        await model.load(translation: .esv, toggles: PassageToggles(), plans: [.bibleYear])
        XCTAssertEqual(model.phase, .failed)
        XCTAssertTrue(model.pills.isEmpty)
    }

    func testPillRateLimitShowsBusyMessage() async {
        StubURLProtocol.handler = { request in
            guard let url = request.url else { return (500, Data()) }
            if url.path == "/api/daily-reading" {
                return (200, Data("""
                {"plans":[{"id":"hope","name":"Hope (2026)","passages":[
                  {"book":"Psalm","chapters":"1","testament":"OT"}]}]}
                """.utf8))
            }
            return (429, Data())
        }

        let model = makeModel()
        await model.load(translation: .esv, toggles: PassageToggles(), plans: [.hope])

        XCTAssertEqual(model.phase, .ready, "one failed pill must not fail the day")
        XCTAssertEqual(model.pills.first?.error, "Service is busy, try again in a moment")
        XCTAssertNil(model.pills.first?.parsed)
    }

    // MARK: Date navigation

    func testDateOffsetsCrossMonthBoundaries() {
        let model = makeModel()
        model.selectedDate = "2026-06-01"
        model.offsetDate(by: -1)
        XCTAssertEqual(model.selectedDate, "2026-05-31")
        model.offsetDate(by: 1)
        XCTAssertEqual(model.selectedDate, "2026-06-01")
    }

    func testTodaySnapsBack() {
        let model = makeModel()
        model.selectedDate = "2020-01-01"
        XCTAssertFalse(model.isToday)
        model.goToday()
        XCTAssertTrue(model.isToday)
        XCTAssertEqual(model.selectedDate, DateTZ.todayString())
    }

    func testCorruptDateFallsBackToToday() {
        let model = makeModel()
        model.selectedDate = "not-a-date"
        model.offsetDate(by: 1)
        XCTAssertEqual(model.selectedDate, DateTZ.todayString())
    }
}
