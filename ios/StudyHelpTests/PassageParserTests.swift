import XCTest
@testable import StudyHelp

/// Fixture-driven parser tests. The fixtures under `Fixtures/` are real
/// `/api/passage` and `/api/crossref` responses recorded from the live
/// backend (see `ios/README.md` for the one-line re-record step). They pin
/// both providers' markup as *fully understood*: every test asserts
/// `unknownClasses` is empty, so an upstream markup change shows up the
/// moment fixtures are re-recorded.
final class PassageParserTests: XCTestCase {

    private func fixture(_ name: String) throws -> PassageResponse {
        let url = try XCTUnwrap(
            Bundle(for: Self.self).url(forResource: name, withExtension: "json"),
            "missing fixture \(name).json"
        )
        return try JSONDecoder().decode(PassageResponse.self, from: Data(contentsOf: url))
    }

    private func allRuns(_ passage: ParsedPassage) -> [InlineRun] {
        passage.blocks.flatMap(\.runs)
    }

    private func blocks(_ passage: ParsedPassage, _ kind: PassageBlock.Kind) -> [PassageBlock] {
        passage.blocks.filter { $0.kind == kind }
    }

    private func poetryLines(_ passage: ParsedPassage) -> [PassageBlock] {
        passage.blocks.filter {
            if case .poetryLine = $0.kind { return true }
            return false
        }
    }

    // MARK: ESV default toggles (Psalm 23)

    func testEsvPsalm23Structure() throws {
        let resp = try fixture("esv-psalm23-default")
        XCTAssertEqual(resp.canonical, "Psalm 23")
        let p = PassageHTMLParser.parse(html: resp.passages[0])

        XCTAssertEqual(p.unknownClasses, [])
        XCTAssertEqual(p.blocks.first?.kind, .passageTitle)
        XCTAssertEqual(p.blocks.first?.text, "Psalm 23")

        let headings = blocks(p, .heading)
        XCTAssertEqual(headings.count, 1)
        XCTAssertEqual(headings[0].text, "The Lord Is My Shepherd")
        // ESV sets the divine name in small caps via span.divine-name.
        XCTAssertTrue(headings[0].runs.contains { $0.isSmallCaps && $0.text == "Lord" })

        XCTAssertEqual(blocks(p, .subtitle).map(\.text), ["A Psalm of David."])
    }

    func testEsvPoetryLinesCarryStructuralIndent() throws {
        let resp = try fixture("esv-psalm23-default")
        let p = PassageHTMLParser.parse(html: resp.passages[0])

        let lines = poetryLines(p)
        XCTAssertFalse(lines.isEmpty)
        XCTAssertTrue(lines.contains { $0.kind == .poetryLine(indent: 0) })
        XCTAssertTrue(lines.contains { $0.kind == .poetryLine(indent: 1) })

        // First line: chapter-num label, then text with the literal &nbsp;
        // indent prefix stripped (indent is structural now).
        let first = try XCTUnwrap(lines.first)
        XCTAssertEqual(first.kind, .poetryLine(indent: 0))
        XCTAssertEqual(first.runs.first?.isVerseNumber, true)
        XCTAssertEqual(first.runs.first?.text, "23:1\u{00A0}")
        let firstText = try XCTUnwrap(first.runs.dropFirst().first)
        XCTAssertTrue(firstText.text.hasPrefix("The LORD is my shepherd"),
                      "leading nbsp indent should be stripped, got: \(firstText.text)")
    }

    func testEsvFootnotesParsedFromTailSection() throws {
        let resp = try fixture("esv-psalm23-default")
        let p = PassageHTMLParser.parse(html: resp.passages[0])

        XCTAssertEqual(p.footnotes.count, 7)
        let first = p.footnotes[0]
        XCTAssertEqual(first.id, "f1-1")
        XCTAssertEqual(first.label, "1")
        XCTAssertEqual(first.reference, "23:2")
        XCTAssertEqual(first.text, "Hebrew beside waters of rest")
        XCTAssertTrue(first.runs.contains { $0.isItalic })

        // The inline marker pointing at it survives with the matching id.
        XCTAssertTrue(allRuns(p).contains { $0.footnoteID == "f1-1" && $0.text == "1" })
    }

    func testEsvAttributionSurvivesParsing() throws {
        let resp = try fixture("esv-psalm23-default")
        let p = PassageHTMLParser.parse(html: resp.passages[0])

        // ESV ships its attribution inside the HTML: (<a class="copyright">ESV</a>).
        // The never-drop rule must carry it through with a live link.
        let last = try XCTUnwrap(p.blocks.last)
        XCTAssertEqual(last.kind, .prose)
        XCTAssertEqual(last.text, "(ESV)")
        XCTAssertTrue(last.runs.contains {
            $0.text == "ESV" && $0.link == URL(string: "http://www.esv.org")
        })
    }

    // MARK: ESV with toggles off (Psalm 117)

    func testEsvAllTogglesOffOmitsChrome() throws {
        let resp = try fixture("esv-psalm117-all-off")
        let p = PassageHTMLParser.parse(html: resp.passages[0])

        XCTAssertEqual(p.unknownClasses, [])
        XCTAssertTrue(blocks(p, .passageTitle).isEmpty)
        XCTAssertTrue(blocks(p, .heading).isEmpty)
        XCTAssertTrue(p.footnotes.isEmpty)
        XCTAssertFalse(poetryLines(p).isEmpty)

        // include_verse_numbers=false still leaves the chapter label —
        // that's upstream ESV behavior, not a parser bug.
        let verseNums = allRuns(p).filter(\.isVerseNumber)
        XCTAssertEqual(verseNums.map(\.text), ["117:1\u{00A0}"])
    }

    // MARK: ESV cross-references (Psalm 117)

    func testEsvCrossRefMarkersLiftedVerbatimWithoutTrailingSlash() throws {
        let resp = try fixture("esv-psalm117-crossrefs")
        let p = PassageHTMLParser.parse(html: resp.passages[0])

        XCTAssertEqual(p.unknownClasses, [])
        let markers = allRuns(p).filter { $0.crossRef != nil }
        // ESV emits the href with a trailing slash ("Romans 15:11/");
        // the parser strips it exactly as PassageArticle.tsx does.
        XCTAssertEqual(markers.map(\.crossRef), [
            "Romans 15:11",
            "Psalm 103:11; Psalm 116:5",
            "Psalm 100:5",
            "Psalm 116:19",
        ])
        XCTAssertEqual(markers.map(\.text), ["c", "d", "e", "b"])
    }

    // MARK: ESV words of Christ (John 15)

    func testEsvWordsOfChristTagged() throws {
        let resp = try fixture("esv-john15-default")
        let p = PassageHTMLParser.parse(html: resp.passages[0])

        XCTAssertEqual(p.unknownClasses, [])
        XCTAssertEqual(blocks(p, .heading).first?.text, "I Am the True Vine")

        let runs = allRuns(p)
        XCTAssertTrue(runs.contains { $0.isWordsOfChrist && !$0.isVerseNumber })
        // ESV emits <b class="verse-num woc"> — flags must combine.
        XCTAssertTrue(runs.contains { $0.isWordsOfChrist && $0.isVerseNumber })
        // The opening chapter label is NOT words-of-Christ.
        let chapterNum = try XCTUnwrap(runs.first { $0.text == "15:1\u{00A0}" })
        XCTAssertTrue(chapterNum.isVerseNumber)
        XCTAssertFalse(chapterNum.isWordsOfChrist)

        XCTAssertEqual(p.footnotes.map(\.id), ["f1-1", "f2-1"])
        XCTAssertEqual(p.footnotes.map(\.reference), ["15:15", "15:22"])
    }

    // MARK: NIV / YouVersion markup (John 15, Psalm 23)

    func testNivJohn15Structure() throws {
        let resp = try fixture("niv-john15")
        let p = PassageHTMLParser.parse(html: resp.passages[0])

        XCTAssertEqual(p.unknownClasses, [])
        XCTAssertEqual(blocks(p, .heading).map(\.text), [
            "The Vine and the Branches",
            "The World Hates the Disciples",
            "The Work of the Holy Spirit",
        ])
        let runs = allRuns(p)
        XCTAssertTrue(runs.contains { $0.isVerseNumber && $0.text == "1" })
        XCTAssertTrue(runs.contains { $0.isWordsOfChrist })
        XCTAssertFalse(runs.contains { $0.crossRef != nil }, "NIV emits no cross-references")
        XCTAssertTrue(p.footnotes.isEmpty)
        // The empty yv-v boundary spans must not leave stray runs.
        XCTAssertFalse(runs.contains { $0.text.isEmpty })
    }

    func testNivPsalm23PoetryAndDivineName() throws {
        let resp = try fixture("niv-psalm23")
        let p = PassageHTMLParser.parse(html: resp.passages[0])

        XCTAssertEqual(p.unknownClasses, [])
        XCTAssertEqual(blocks(p, .passageTitle).map(\.text), ["Psalm 23"])
        XCTAssertEqual(blocks(p, .subtitle).map(\.text), ["A psalm of David."])

        let lines = poetryLines(p)
        XCTAssertTrue(lines.contains { $0.kind == .poetryLine(indent: 0) })
        XCTAssertTrue(lines.contains { $0.kind == .poetryLine(indent: 1) })
        XCTAssertTrue(allRuns(p).contains { $0.isSmallCaps && $0.text == "Lord" })
    }

    // MARK: /api/crossref responses

    func testCrossrefResponseParsesPerPassage() throws {
        let resp = try fixture("crossref-job38")
        XCTAssertEqual(resp.canonical, "Job 38:4\u{2013}7; Psalm 33:6")
        let p = PassageHTMLParser.parse(passages: resp.passages)

        XCTAssertEqual(p.unknownClasses, [])
        XCTAssertEqual(blocks(p, .passageTitle).map(\.text),
                       ["Job 38:4\u{2013}7", "Psalm 33:6"])
        // Attribution after each passage survives.
        XCTAssertEqual(
            allRuns(p).filter { $0.link == URL(string: "http://www.esv.org") }.count, 2
        )
    }

    // MARK: Multi-passage footnote namespacing

    func testMultiPassageFootnoteIDsAreNamespaced() throws {
        let resp = try fixture("esv-psalm23-default")
        let html = resp.passages[0]
        let p = PassageHTMLParser.parse(passages: [html, html])

        XCTAssertEqual(p.footnotes.count, 14)
        XCTAssertEqual(p.footnotes.first?.id, "p0-f1-1")
        XCTAssertEqual(p.footnotes[7].id, "p1-f1-1")
        let markerIDs = Set(allRuns(p).compactMap(\.footnoteID))
        XCTAssertTrue(markerIDs.contains("p0-f1-1"))
        XCTAssertTrue(markerIDs.contains("p1-f1-1"))
    }

    // MARK: Never-drop guarantees

    func testUnknownClassesPassThroughAndAreReported() {
        let p = PassageHTMLParser.parse(
            html: #"<p><span class="brand-new-thing">still readable</span></p>"#
        )
        XCTAssertEqual(p.blocks.map(\.text), ["still readable"])
        XCTAssertTrue(p.unknownClasses.contains("brand-new-thing"))
    }

    func testMalformedHTMLDegradesInsteadOfThrowing() {
        let p = PassageHTMLParser.parse(html: "<p>unclosed <b>bold")
        XCTAssertEqual(p.blocks.count, 1)
        XCTAssertEqual(p.blocks[0].text, "unclosed bold")
        XCTAssertTrue(p.blocks[0].runs.contains { $0.isBold && $0.text == "bold" })
    }

    func testUnknownBlockTagFallsBackToProse() {
        let p = PassageHTMLParser.parse(
            html: "<aside>side note <a href=\"https://example.com\">link</a></aside>"
        )
        XCTAssertEqual(p.blocks.count, 1)
        XCTAssertEqual(p.blocks[0].kind, .prose)
        XCTAssertEqual(p.blocks[0].text, "side note link")
        XCTAssertTrue(p.blocks[0].runs.contains {
            $0.link == URL(string: "https://example.com")
        })
    }
}
