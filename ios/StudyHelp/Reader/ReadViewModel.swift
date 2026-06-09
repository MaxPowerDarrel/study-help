import Foundation
import Observation

/// State machine for the Read tab: a chapter location (plus optional range
/// end), fetch → parse → publish, with the same race guard and error split
/// the web's reader has (`web/src/App.tsx` / `api.ts`).
///
/// Parsed passages are cached in memory for the session, keyed by
/// query + translation + the five server-sent toggles — never persisted
/// (ToU; `specs/native-reader.md`). The words-of-Christ toggle is absent
/// from the key on purpose: it's render-time only.
@Observable
@MainActor
final class ReadViewModel {
    enum Phase: Equatable {
        case idle
        case loading
        case loaded
        case rateLimited
        case failed
    }

    private(set) var phase: Phase = .idle
    private(set) var parsed: ParsedPassage?
    /// Server-normalized reference for the loaded passage (e.g. "John 3").
    private(set) var canonical = ""

    /// Defaults to John 3, the web reader's starting passage.
    var location = Canon.ChapterRef(bookIndex: 42, chapter: 3)
    /// End chapter for a range within the same book (e.g. Genesis 1–3).
    var rangeEnd: Int?

    @ObservationIgnored private let api: PassageAPI
    @ObservationIgnored private let cache = NSCache<NSString, CacheEntry>()
    @ObservationIgnored private var fetchID = 0

    init(api: PassageAPI = PassageAPI()) {
        self.api = api
    }

    /// `GET /api/passage` query for the current location, e.g. `"John 3"`
    /// or `"Genesis 1-3"`.
    var query: String {
        if let end = rangeEnd, end > location.chapter {
            return "\(Canon.query(for: location))-\(end)"
        }
        return Canon.query(for: location)
    }

    var canGoPrevious: Bool { Canon.previous(location) != nil }
    var canGoNext: Bool { Canon.next(location) != nil }

    func goPrevious() {
        guard let prev = Canon.previous(location) else { return }
        rangeEnd = nil
        location = prev
    }

    func goNext() {
        // Leaving a range resumes single-chapter reading after its end.
        if let end = rangeEnd {
            rangeEnd = nil
            let last = Canon.ChapterRef(bookIndex: location.bookIndex, chapter: end)
            if let next = Canon.next(last) {
                location = next
            }
            return
        }
        guard let next = Canon.next(location) else { return }
        location = next
    }

    func select(_ ref: Canon.ChapterRef, rangeEnd end: Int?) {
        location = ref
        rangeEnd = (end ?? 0) > ref.chapter ? end : nil
    }

    func load(translation: TranslationOption, toggles: PassageToggles) async {
        fetchID += 1
        let id = fetchID
        let q = query
        let key = cacheKey(q: q, translation: translation, toggles: toggles)

        if let hit = cache.object(forKey: key) {
            parsed = hit.parsed
            canonical = hit.canonical
            phase = .loaded
            return
        }

        phase = .loading
        do {
            let resp = try await api.passage(q: q, translation: translation, toggles: toggles)
            guard id == fetchID else { return } // a newer load superseded us
            let result = PassageHTMLParser.parse(passages: resp.passages)
            cache.setObject(CacheEntry(parsed: result, canonical: resp.canonical), forKey: key)
            parsed = result
            canonical = resp.canonical
            phase = .loaded
        } catch is CancellationError {
            // .task(id:) cancelled us because the inputs changed; the
            // replacement load owns the state now.
        } catch PassageAPI.Failure.rateLimited {
            guard id == fetchID else { return }
            phase = .rateLimited
        } catch {
            guard id == fetchID else { return }
            if (error as? URLError)?.code == .cancelled { return }
            phase = .failed
        }
    }

    private func cacheKey(
        q: String, translation: TranslationOption, toggles: PassageToggles
    ) -> NSString {
        "\(q)|\(translation.rawValue)|\(toggles.headings)|\(toggles.footnotes)|" +
        "\(toggles.verseNumbers)|\(toggles.passageReferences)|\(toggles.crossReferences)" as NSString
    }

    private final class CacheEntry {
        let parsed: ParsedPassage
        let canonical: String
        init(parsed: ParsedPassage, canonical: String) {
            self.parsed = parsed
            self.canonical = canonical
        }
    }
}
