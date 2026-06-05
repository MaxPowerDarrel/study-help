import XCTest
@testable import StudyHelp

/// Decoding tests against captured `GET /api/daily-reading` payloads. Sample
/// JSON mirrors `internal/server/daily_test.go` so the Swift models stay in
/// lockstep with the Go envelope.
final class DailyReadingTests: XCTestCase {

    func testDecodeTwoPassageDay() throws {
        let json = Data("""
        {"plans":[{"id":"bible-year","name":"Bible in One Year","passages":[
          {"book":"Numbers","chapters":"20-22","testament":"OT"},
          {"book":"Revelation","chapters":"22","testament":"NT"}
        ]}]}
        """.utf8)

        let resp = try JSONDecoder().decode(DailyResponse.self, from: json)

        XCTAssertEqual(resp.plans.count, 1)
        let plan = resp.plans[0]
        XCTAssertEqual(plan.id, "bible-year")
        XCTAssertEqual(plan.name, "Bible in One Year")
        XCTAssertEqual(plan.passages.count, 2)
        XCTAssertEqual(plan.passages[0].book, "Numbers")
        XCTAssertEqual(plan.passages[0].chapters, "20-22")
        XCTAssertEqual(plan.passages[0].testament, "OT")
        XCTAssertNil(plan.message)
    }

    func testDecodeMessageOnlyDay() throws {
        let json = Data("""
        {"plans":[{"id":"hope","name":"Hope (2026)","passages":[],"message":"Catch-up day!"}]}
        """.utf8)

        let resp = try JSONDecoder().decode(DailyResponse.self, from: json)

        XCTAssertTrue(resp.plans[0].passages.isEmpty)
        XCTAssertEqual(resp.plans[0].message, "Catch-up day!")
    }

    func testPassageDisplayUsesEnDashAndSpacing() {
        XCTAssertEqual(
            DailyPassage(book: "Genesis", chapters: "1-3", testament: "OT").display,
            "Genesis 1\u{2013}3"
        )
        XCTAssertEqual(
            DailyPassage(book: "Matthew", chapters: "11,12", testament: "NT").display,
            "Matthew 11, 12"
        )
        XCTAssertEqual(
            DailyPassage(book: "Psalm", chapters: "1", testament: "OT").display,
            "Psalm 1"
        )
    }

    func testCachedDailyRoundTrips() throws {
        let cached = CachedDaily(
            response: DailyResponse(plans: [
                DailyPlanResult(id: "hope", name: "Hope (2026)", passages: [
                    DailyPassage(book: "Psalm", chapters: "1", testament: "OT"),
                ], message: nil),
            ]),
            readingDate: "2026-06-05",
            fetchedAt: Date(timeIntervalSince1970: 1_750_000_000)
        )

        let data = try JSONEncoder().encode(cached)
        let decoded = try JSONDecoder().decode(CachedDaily.self, from: data)

        XCTAssertEqual(decoded.readingDate, "2026-06-05")
        XCTAssertEqual(decoded.response, cached.response)
    }

    func testPlanRawValuesMatchServerParams() {
        XCTAssertEqual(PlanOption.bibleYear.rawValue, "bible-year")
        XCTAssertEqual(PlanOption.hope.rawValue, "hope")
        XCTAssertEqual(TranslationOption.esv.rawValue, "ESV")
        XCTAssertEqual(TranslationOption.niv.rawValue, "NIV")
    }
}
