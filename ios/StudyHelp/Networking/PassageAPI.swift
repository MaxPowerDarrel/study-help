import Foundation

/// Mirrors the `{canonical, passages}` envelope shared by `GET /api/passage`
/// and `GET /api/crossref` (`internal/server/passage.go`, `crossref.go`).
/// `passages` holds provider-rendered HTML blobs — scripture text. It is held
/// in memory only and never persisted (ESV/YouVersion ToU; see
/// `specs/native-reader.md`).
struct PassageResponse: Codable, Equatable {
    let canonical: String
    let passages: [String]
}

/// The five formatting options the server forwards upstream as query params.
/// Words-of-Christ is deliberately absent: it has no upstream toggle and is
/// applied at render time (see `web/src/toggles.ts` and
/// `AttributedRendering.swift`). Defaults match the web client's.
struct PassageToggles: Equatable, Hashable {
    var headings = true
    var footnotes = true
    var verseNumbers = true
    var passageReferences = true
    /// ESV-only; NIV ignores it. Off by default to match the web UI.
    var crossReferences = false

    var queryItems: [URLQueryItem] {
        [
            URLQueryItem(name: "include_headings", value: String(headings)),
            URLQueryItem(name: "include_footnotes", value: String(footnotes)),
            URLQueryItem(name: "include_verse_numbers", value: String(verseNumbers)),
            URLQueryItem(name: "include_passage_references", value: String(passageReferences)),
            URLQueryItem(name: "include_cross_references", value: String(crossReferences)),
        ]
    }
}

/// Client for the scripture endpoints. Lives in the app target — not
/// `Shared/` — so the widget extension structurally cannot fetch scripture
/// text (`specs/native-reader.md`, Decisions 2026-06-09).
struct PassageAPI {
    var baseURL: URL = Config.backendBaseURL
    var session: URLSession = .shared

    /// Failure cases mirror the web client's `FetchResult` mapping
    /// (`web/src/api.ts`): 429 is surfaced distinctly so the UI can say
    /// "busy, try again" instead of a generic error.
    enum Failure: Error, Equatable {
        case invalidURL
        case rateLimited
        case badStatus(Int)
    }

    /// Fetch a chapter or chapter range, e.g. `"John 3"` or `"Genesis 1-3"`.
    func passage(
        q: String,
        translation: TranslationOption,
        toggles: PassageToggles = PassageToggles()
    ) async throws -> PassageResponse {
        var items = toggles.queryItems
        items.append(URLQueryItem(name: "q", value: q))
        items.append(URLQueryItem(name: "translation", value: translation.rawValue))
        return try await get(path: "api/passage", queryItems: items)
    }

    /// Look up the verse text behind an ESV cross-reference marker. `q` is
    /// the bare reference list lifted from the marker's href, e.g.
    /// `"Job 38:4-7; Psalm 33:6"`. ESV-only by construction — NIV passages
    /// carry no markers.
    func crossref(q: String) async throws -> PassageResponse {
        try await get(path: "api/crossref", queryItems: [URLQueryItem(name: "q", value: q)])
    }

    private func get(path: String, queryItems: [URLQueryItem]) async throws -> PassageResponse {
        var comps = URLComponents(
            url: baseURL.appendingPathComponent(path),
            resolvingAgainstBaseURL: false
        )
        comps?.queryItems = queryItems
        guard let url = comps?.url else { throw Failure.invalidURL }

        let (data, response) = try await session.data(from: url)
        let status = (response as? HTTPURLResponse)?.statusCode ?? -1
        if status == 429 { throw Failure.rateLimited }
        guard (200..<300).contains(status) else { throw Failure.badStatus(status) }

        return try JSONDecoder().decode(PassageResponse.self, from: data)
    }
}
