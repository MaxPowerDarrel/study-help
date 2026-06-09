import XCTest
import SwiftUI
@testable import StudyHelp

/// Inline-attribute rendering: the words-of-Christ render-time flip and the
/// custom-scheme links that make markers tappable.
final class AttributedRenderingTests: XCTestCase {

    func testWordsOfChristColorFollowsToggle() {
        var woc = InlineRun(text: "I am the true vine")
        woc.isWordsOfChrist = true

        var on = PassageRenderer()
        on.showWordsOfChrist = true
        let colored = on.attributedString(for: [woc])
        XCTAssertEqual(colored.runs.first?.foregroundColor, on.palette.wordsOfChrist)

        // Off = inherit, exactly like the web's .passage.no-woc flip.
        var off = PassageRenderer()
        off.showWordsOfChrist = false
        let plain = off.attributedString(for: [woc])
        XCTAssertNil(plain.runs.first?.foregroundColor)
    }

    func testVerseNumberStyling() {
        var num = InlineRun(text: "23:1\u{00A0}")
        num.isVerseNumber = true
        let renderer = PassageRenderer()
        let rendered = renderer.attributedString(for: [num])
        let run = rendered.runs.first
        XCTAssertEqual(run?.foregroundColor, renderer.palette.verseNumber)
        XCTAssertNotNil(run?.baselineOffset)
    }

    func testCrossRefMarkerBecomesTappableLink() throws {
        var marker = InlineRun(text: "d")
        marker.crossRef = "Psalm 103:11; Psalm 116:5"
        marker.isSuperscript = true

        let rendered = PassageRenderer().attributedString(for: [marker])
        let url = try XCTUnwrap(rendered.runs.first?.link)
        XCTAssertEqual(url.scheme, ReaderLink.crossrefScheme)
        XCTAssertEqual(ReaderLink.crossrefQuery(from: url), "Psalm 103:11; Psalm 116:5")
    }

    func testFootnoteMarkerBecomesTappableLink() throws {
        var marker = InlineRun(text: "1")
        marker.footnoteID = "p0-f1-1"

        let rendered = PassageRenderer().attributedString(for: [marker])
        let url = try XCTUnwrap(rendered.runs.first?.link)
        XCTAssertEqual(url.scheme, ReaderLink.footnoteScheme)
        XCTAssertEqual(ReaderLink.footnoteID(from: url), "p0-f1-1")
    }

    func testRealLinksPassThrough() throws {
        var attribution = InlineRun(text: "ESV")
        attribution.link = URL(string: "http://www.esv.org")

        let rendered = PassageRenderer().attributedString(for: [attribution])
        XCTAssertEqual(rendered.runs.first?.link, URL(string: "http://www.esv.org"))
    }

    func testReaderLinkRoundTripsAwkwardQueries() {
        let q = "Song of Solomon 1:1; Psalm 116:5"
        let url = ReaderLink.crossref(q: q)
        XCTAssertNotNil(url)
        XCTAssertEqual(ReaderLink.crossrefQuery(from: url!), q)
        XCTAssertNil(ReaderLink.crossrefQuery(from: URL(string: "https://example.com")!))
    }
}
