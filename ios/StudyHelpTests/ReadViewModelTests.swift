import XCTest
@testable import StudyHelp

/// Read-tab state machine: fetch → parse → publish, error split (429 vs
/// the rest), session cache, and chapter/range navigation over `Canon`.
@MainActor
final class ReadViewModelTests: XCTestCase {

    private let envelope =
        #"{"canonical":"John 3","passages":["<p><b class=\"chapter-num\">3:1&nbsp;</b>Now there was a man of the Pharisees</p>"]}"#

    override func tearDown() {
        StubURLProtocol.handler = nil
        super.tearDown()
    }

    private func makeModel() -> ReadViewModel {
        ReadViewModel(api: makeStubbedPassageAPI())
    }

    func testLoadParsesAndPublishes() async {
        StubURLProtocol.handler = { [envelope] _ in (200, Data(envelope.utf8)) }
        let model = makeModel()

        await model.load(translation: .esv, toggles: PassageToggles())

        XCTAssertEqual(model.phase, .loaded)
        XCTAssertEqual(model.canonical, "John 3")
        let text = model.parsed?.blocks.first?.text ?? ""
        XCTAssertTrue(text.contains("Now there was a man"))
        XCTAssertTrue(model.parsed?.blocks.first?.runs.first?.isVerseNumber ?? false)
    }

    func testRateLimitedSurfacesDistinctly() async {
        StubURLProtocol.handler = { _ in (429, Data()) }
        let model = makeModel()
        await model.load(translation: .esv, toggles: PassageToggles())
        XCTAssertEqual(model.phase, .rateLimited)
    }

    func testUpstreamErrorFails() async {
        StubURLProtocol.handler = { _ in (502, Data()) }
        let model = makeModel()
        await model.load(translation: .esv, toggles: PassageToggles())
        XCTAssertEqual(model.phase, .failed)
    }

    func testSessionCacheAvoidsRefetch() async {
        var requests = 0
        StubURLProtocol.handler = { [envelope] _ in
            requests += 1
            return (200, Data(envelope.utf8))
        }
        let model = makeModel()

        await model.load(translation: .esv, toggles: PassageToggles())
        await model.load(translation: .esv, toggles: PassageToggles())
        XCTAssertEqual(requests, 1, "same query+translation+toggles must hit the session cache")

        // A different toggle set is a different cache key.
        var crossRefs = PassageToggles()
        crossRefs.crossReferences = true
        await model.load(translation: .esv, toggles: crossRefs)
        XCTAssertEqual(requests, 2)

        // And so is a different translation.
        await model.load(translation: .niv, toggles: PassageToggles())
        XCTAssertEqual(requests, 3)
    }

    func testQueryFormatsSingleChapterAndRange() {
        let model = makeModel()
        XCTAssertEqual(model.query, "John 3", "defaults to the web reader's start passage")

        model.select(Canon.ChapterRef(bookIndex: 0, chapter: 1), rangeEnd: 3)
        XCTAssertEqual(model.query, "Genesis 1-3")

        // A range end at or below the start chapter means single chapter.
        model.select(Canon.ChapterRef(bookIndex: 0, chapter: 5), rangeEnd: 5)
        XCTAssertEqual(model.query, "Genesis 5")
    }

    func testNextFromRangeResumesAfterRangeEnd() {
        let model = makeModel()
        model.select(Canon.ChapterRef(bookIndex: 0, chapter: 1), rangeEnd: 3)

        model.goNext()
        XCTAssertEqual(model.location, Canon.ChapterRef(bookIndex: 0, chapter: 4))
        XCTAssertNil(model.rangeEnd)
        XCTAssertEqual(model.query, "Genesis 4")
    }

    func testNavigationRollsAcrossBooks() {
        let model = makeModel()
        model.select(Canon.ChapterRef(bookIndex: 42, chapter: 21), rangeEnd: nil) // John 21

        model.goNext()
        XCTAssertEqual(model.location, Canon.ChapterRef(bookIndex: 43, chapter: 1)) // Acts 1

        model.goPrevious()
        XCTAssertEqual(model.location, Canon.ChapterRef(bookIndex: 42, chapter: 21))
    }
}
