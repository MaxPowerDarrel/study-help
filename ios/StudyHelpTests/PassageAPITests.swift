import XCTest
@testable import StudyHelp

/// `PassageAPI` against a stubbed `URLProtocol`: request construction
/// mirrors `web/src/api.ts` (q + translation + five include_* params) and
/// the error mapping surfaces 429 distinctly.
final class PassageAPITests: XCTestCase {

    private func makeAPI() -> PassageAPI { makeStubbedPassageAPI() }

    override func tearDown() {
        StubURLProtocol.handler = nil
        super.tearDown()
    }

    func testPassageRequestCarriesQueryTranslationAndToggles() async throws {
        var captured: URL?
        StubURLProtocol.handler = { request in
            captured = request.url
            return (200, Data(#"{"canonical":"John 3","passages":["<p>x</p>"]}"#.utf8))
        }

        var toggles = PassageToggles()
        toggles.crossReferences = true
        _ = try await makeAPI().passage(q: "John 3", translation: .esv, toggles: toggles)

        let comps = try XCTUnwrap(
            URLComponents(url: try XCTUnwrap(captured), resolvingAgainstBaseURL: false)
        )
        XCTAssertEqual(comps.path, "/api/passage")
        var params: [String: String] = [:]
        for item in comps.queryItems ?? [] { params[item.name] = item.value }
        XCTAssertEqual(params["q"], "John 3")
        XCTAssertEqual(params["translation"], "ESV")
        XCTAssertEqual(params["include_headings"], "true")
        XCTAssertEqual(params["include_footnotes"], "true")
        XCTAssertEqual(params["include_verse_numbers"], "true")
        XCTAssertEqual(params["include_passage_references"], "true")
        XCTAssertEqual(params["include_cross_references"], "true")
    }

    func testDecodesEnvelope() async throws {
        StubURLProtocol.handler = { _ in
            (200, Data(#"{"canonical":"Psalm 23","passages":["<p>a</p>","<p>b</p>"]}"#.utf8))
        }
        let resp = try await makeAPI().passage(q: "Psalm 23", translation: .esv)
        XCTAssertEqual(resp.canonical, "Psalm 23")
        XCTAssertEqual(resp.passages.count, 2)
    }

    func testCrossrefHitsCrossrefEndpoint() async throws {
        var captured: URL?
        StubURLProtocol.handler = { request in
            captured = request.url
            return (200, Data(#"{"canonical":"Job 38:4-7","passages":["<p>x</p>"]}"#.utf8))
        }
        _ = try await makeAPI().crossref(q: "Job 38:4-7; Psalm 33:6")

        let comps = try XCTUnwrap(
            URLComponents(url: try XCTUnwrap(captured), resolvingAgainstBaseURL: false)
        )
        XCTAssertEqual(comps.path, "/api/crossref")
        XCTAssertEqual(
            comps.queryItems?.first(where: { $0.name == "q" })?.value,
            "Job 38:4-7; Psalm 33:6"
        )
    }

    func testRateLimitedSurfacesDistinctly() async {
        StubURLProtocol.handler = { _ in (429, Data()) }
        do {
            _ = try await makeAPI().passage(q: "John 3", translation: .esv)
            XCTFail("expected rateLimited")
        } catch let error as PassageAPI.Failure {
            XCTAssertEqual(error, .rateLimited)
        } catch {
            XCTFail("unexpected error: \(error)")
        }
    }

    func testServerErrorSurfacesStatus() async {
        StubURLProtocol.handler = { _ in (502, Data()) }
        do {
            _ = try await makeAPI().passage(q: "John 3", translation: .niv)
            XCTFail("expected badStatus")
        } catch let error as PassageAPI.Failure {
            XCTAssertEqual(error, .badStatus(502))
        } catch {
            XCTFail("unexpected error: \(error)")
        }
    }
}
