import XCTest
@testable import StudyHelp

/// Guards the `web/src/canon.ts` port: same books, same chapter counts,
/// same prev/next roll-over behavior.
final class CanonTests: XCTestCase {

    func testCanonShape() {
        XCTAssertEqual(Canon.books.count, 66)
        XCTAssertEqual(Canon.books.first, Canon.Book(name: "Genesis", chapters: 50))
        XCTAssertEqual(Canon.books.last, Canon.Book(name: "Revelation", chapters: 22))
        XCTAssertEqual(Canon.books[18], Canon.Book(name: "Psalms", chapters: 150))
        XCTAssertEqual(Canon.books[42].name, "John")
    }

    func testNextRollsIntoFollowingBook() {
        XCTAssertEqual(
            Canon.next(Canon.ChapterRef(bookIndex: 42, chapter: 21)),
            Canon.ChapterRef(bookIndex: 43, chapter: 1)
        )
        XCTAssertEqual(
            Canon.next(Canon.ChapterRef(bookIndex: 0, chapter: 1)),
            Canon.ChapterRef(bookIndex: 0, chapter: 2)
        )
        XCTAssertNil(Canon.next(Canon.ChapterRef(bookIndex: 65, chapter: 22)))
    }

    func testPreviousRollsIntoPrecedingBookLastChapter() {
        XCTAssertEqual(
            Canon.previous(Canon.ChapterRef(bookIndex: 43, chapter: 1)),
            Canon.ChapterRef(bookIndex: 42, chapter: 21)
        )
        XCTAssertEqual(
            Canon.previous(Canon.ChapterRef(bookIndex: 0, chapter: 2)),
            Canon.ChapterRef(bookIndex: 0, chapter: 1)
        )
        XCTAssertNil(Canon.previous(Canon.ChapterRef(bookIndex: 0, chapter: 1)))
    }

    func testQueryMatchesServerFormat() {
        XCTAssertEqual(Canon.query(for: Canon.ChapterRef(bookIndex: 42, chapter: 3)), "John 3")
        XCTAssertEqual(Canon.query(for: Canon.ChapterRef(bookIndex: 18, chapter: 117)), "Psalms 117")
    }

    func testIsValidBounds() {
        XCTAssertTrue(Canon.isValid(Canon.ChapterRef(bookIndex: 65, chapter: 22)))
        XCTAssertFalse(Canon.isValid(Canon.ChapterRef(bookIndex: 0, chapter: 0)))
        XCTAssertFalse(Canon.isValid(Canon.ChapterRef(bookIndex: 0, chapter: 51)))
        XCTAssertFalse(Canon.isValid(Canon.ChapterRef(bookIndex: 66, chapter: 1)))
        XCTAssertFalse(Canon.isValid(Canon.ChapterRef(bookIndex: -1, chapter: 1)))
    }
}
