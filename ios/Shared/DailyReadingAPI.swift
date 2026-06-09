import Foundation

/// Thin client for `GET /api/daily-reading`. The endpoint returns plan
/// references only (book + chapters), so no API key or auth is involved and
/// nothing here touches scripture text.
struct DailyReadingAPI {
    var baseURL: URL = Config.backendBaseURL
    var session: URLSession = .shared

    enum APIError: Error, Equatable {
        case invalidURL
        case badStatus(Int)
    }

    /// Fetch the day's reading for a single plan. `date` is `YYYY-MM-DD`;
    /// `timeZone` is sent as the required `tz` param so the server resolves the
    /// same calendar day the device is on.
    func fetch(
        plan: PlanOption,
        date: String,
        timeZone: TimeZone = .current
    ) async throws -> DailyResponse {
        try await fetch(plans: [plan], date: date, timeZone: timeZone)
    }

    /// Multi-plan variant for the app's Daily tab: one request with a
    /// joined `plans=` param, matching `web/src/api.ts`'s
    /// `fetchDailyReading`. The widget keeps the single-plan call above
    /// (one plan per widget).
    func fetch(
        plans: [PlanOption],
        date: String,
        timeZone: TimeZone = .current
    ) async throws -> DailyResponse {
        var comps = URLComponents(
            url: baseURL.appendingPathComponent("api/daily-reading"),
            resolvingAgainstBaseURL: false
        )
        comps?.queryItems = [
            URLQueryItem(name: "tz", value: timeZone.identifier),
            URLQueryItem(name: "plans", value: plans.map(\.rawValue).joined(separator: ",")),
            URLQueryItem(name: "date", value: date),
        ]
        guard let url = comps?.url else { throw APIError.invalidURL }

        let (data, response) = try await session.data(from: url)
        let status = (response as? HTTPURLResponse)?.statusCode ?? -1
        guard (200..<300).contains(status) else { throw APIError.badStatus(status) }

        return try JSONDecoder().decode(DailyResponse.self, from: data)
    }
}
