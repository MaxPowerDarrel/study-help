import XCTest
@testable import StudyHelp

/// Cross-reference sheet model: the `/api/crossref` fetch behind a tapped
/// marker, with the web popover's loading / busy / error split.
@MainActor
final class CrossrefLookupTests: XCTestCase {

    override func tearDown() {
        StubURLProtocol.handler = nil
        super.tearDown()
    }

    private func makeLookup() -> CrossrefLookup {
        CrossrefLookup(api: makeStubbedPassageAPI())
    }

    func testLoadParsesPerPassage() async throws {
        var captured: URL?
        StubURLProtocol.handler = { request in
            captured = request.url
            let body = #"""
            {"canonical":"Job 38:4–7; Psalm 33:6","passages":[
              "<h2 class=\"extra_text\">Job 38:4–7</h2><p>Where were you</p>",
              "<h2 class=\"extra_text\">Psalm 33:6</h2><p>By the word of the LORD</p>"
            ]}
            """#
            return (200, Data(body.utf8))
        }

        let lookup = makeLookup()
        await lookup.load(q: "Job 38:4-7; Psalm 33:6")

        XCTAssertEqual(lookup.phase, .ready)
        XCTAssertEqual(lookup.canonical, "Job 38:4\u{2013}7; Psalm 33:6")
        let parsed = try XCTUnwrap(lookup.parsed)
        XCTAssertEqual(
            parsed.blocks.filter { $0.kind == .passageTitle }.map(\.text),
            ["Job 38:4\u{2013}7", "Psalm 33:6"]
        )

        // The marker's reference string goes out verbatim as `q`.
        let comps = URLComponents(url: try XCTUnwrap(captured), resolvingAgainstBaseURL: false)
        XCTAssertEqual(comps?.path, "/api/crossref")
        XCTAssertEqual(
            comps?.queryItems?.first(where: { $0.name == "q" })?.value,
            "Job 38:4-7; Psalm 33:6"
        )
    }

    func testRateLimitedSurfacesDistinctly() async {
        StubURLProtocol.handler = { _ in (429, Data()) }
        let lookup = makeLookup()
        await lookup.load(q: "Psalm 100:5")
        XCTAssertEqual(lookup.phase, .rateLimited)
    }

    func testUpstreamErrorFails() async {
        StubURLProtocol.handler = { _ in (502, Data()) }
        let lookup = makeLookup()
        await lookup.load(q: "Psalm 100:5")
        XCTAssertEqual(lookup.phase, .failed)
        XCTAssertNil(lookup.parsed)
    }
}
