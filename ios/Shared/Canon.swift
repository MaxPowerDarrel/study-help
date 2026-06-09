import Foundation

/// 66-book ESV canon with chapter counts, hard-coded so the picker has zero
/// round-trips on load. Direct port of `web/src/canon.ts` — keep the two in
/// lockstep (see passage-reader.md Decisions, 2026-05-02 — picker source).
enum Canon {
    struct Book: Equatable {
        let name: String
        let chapters: Int
    }

    /// A chapter position expressed against `books` — same shape the web
    /// client persists in `localStorage` (`web/src/restore.ts`).
    struct ChapterRef: Equatable, Codable {
        var bookIndex: Int
        var chapter: Int
    }

    static let books: [Book] = [
        Book(name: "Genesis", chapters: 50),
        Book(name: "Exodus", chapters: 40),
        Book(name: "Leviticus", chapters: 27),
        Book(name: "Numbers", chapters: 36),
        Book(name: "Deuteronomy", chapters: 34),
        Book(name: "Joshua", chapters: 24),
        Book(name: "Judges", chapters: 21),
        Book(name: "Ruth", chapters: 4),
        Book(name: "1 Samuel", chapters: 31),
        Book(name: "2 Samuel", chapters: 24),
        Book(name: "1 Kings", chapters: 22),
        Book(name: "2 Kings", chapters: 25),
        Book(name: "1 Chronicles", chapters: 29),
        Book(name: "2 Chronicles", chapters: 36),
        Book(name: "Ezra", chapters: 10),
        Book(name: "Nehemiah", chapters: 13),
        Book(name: "Esther", chapters: 10),
        Book(name: "Job", chapters: 42),
        Book(name: "Psalms", chapters: 150),
        Book(name: "Proverbs", chapters: 31),
        Book(name: "Ecclesiastes", chapters: 12),
        Book(name: "Song of Solomon", chapters: 8),
        Book(name: "Isaiah", chapters: 66),
        Book(name: "Jeremiah", chapters: 52),
        Book(name: "Lamentations", chapters: 5),
        Book(name: "Ezekiel", chapters: 48),
        Book(name: "Daniel", chapters: 12),
        Book(name: "Hosea", chapters: 14),
        Book(name: "Joel", chapters: 3),
        Book(name: "Amos", chapters: 9),
        Book(name: "Obadiah", chapters: 1),
        Book(name: "Jonah", chapters: 4),
        Book(name: "Micah", chapters: 7),
        Book(name: "Nahum", chapters: 3),
        Book(name: "Habakkuk", chapters: 3),
        Book(name: "Zephaniah", chapters: 3),
        Book(name: "Haggai", chapters: 2),
        Book(name: "Zechariah", chapters: 14),
        Book(name: "Malachi", chapters: 4),
        Book(name: "Matthew", chapters: 28),
        Book(name: "Mark", chapters: 16),
        Book(name: "Luke", chapters: 24),
        Book(name: "John", chapters: 21),
        Book(name: "Acts", chapters: 28),
        Book(name: "Romans", chapters: 16),
        Book(name: "1 Corinthians", chapters: 16),
        Book(name: "2 Corinthians", chapters: 13),
        Book(name: "Galatians", chapters: 6),
        Book(name: "Ephesians", chapters: 6),
        Book(name: "Philippians", chapters: 4),
        Book(name: "Colossians", chapters: 4),
        Book(name: "1 Thessalonians", chapters: 5),
        Book(name: "2 Thessalonians", chapters: 3),
        Book(name: "1 Timothy", chapters: 6),
        Book(name: "2 Timothy", chapters: 4),
        Book(name: "Titus", chapters: 3),
        Book(name: "Philemon", chapters: 1),
        Book(name: "Hebrews", chapters: 13),
        Book(name: "James", chapters: 5),
        Book(name: "1 Peter", chapters: 5),
        Book(name: "2 Peter", chapters: 3),
        Book(name: "1 John", chapters: 5),
        Book(name: "2 John", chapters: 1),
        Book(name: "3 John", chapters: 1),
        Book(name: "Jude", chapters: 1),
        Book(name: "Revelation", chapters: 22),
    ]

    static func next(_ ref: ChapterRef) -> ChapterRef? {
        let book = books[ref.bookIndex]
        if ref.chapter < book.chapters {
            return ChapterRef(bookIndex: ref.bookIndex, chapter: ref.chapter + 1)
        }
        if ref.bookIndex < books.count - 1 {
            return ChapterRef(bookIndex: ref.bookIndex + 1, chapter: 1)
        }
        return nil
    }

    static func previous(_ ref: ChapterRef) -> ChapterRef? {
        if ref.chapter > 1 {
            return ChapterRef(bookIndex: ref.bookIndex, chapter: ref.chapter - 1)
        }
        if ref.bookIndex > 0 {
            let prevBook = books[ref.bookIndex - 1]
            return ChapterRef(bookIndex: ref.bookIndex - 1, chapter: prevBook.chapters)
        }
        return nil
    }

    /// `GET /api/passage` query string for a chapter, e.g. `"John 3"`.
    static func query(for ref: ChapterRef) -> String {
        "\(books[ref.bookIndex].name) \(ref.chapter)"
    }

    /// Whether a persisted ref still points inside the canon (guards
    /// against corrupt or out-of-range restore data).
    static func isValid(_ ref: ChapterRef) -> Bool {
        guard books.indices.contains(ref.bookIndex) else { return false }
        return (1...books[ref.bookIndex].chapters).contains(ref.chapter)
    }
}
