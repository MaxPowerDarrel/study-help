import Foundation

/// Mirrors the `GET /api/daily-reading` JSON envelope produced by
/// `internal/server/daily.go` (`dailyResponse` / `dailyPlanResult` /
/// `dailyPassage`). References only — never scripture text.
struct DailyResponse: Codable, Equatable {
    let plans: [DailyPlanResult]
}

struct DailyPlanResult: Codable, Equatable, Identifiable {
    let id: String
    let name: String
    let passages: [DailyPassage]
    let message: String?
}

struct DailyPassage: Codable, Equatable {
    let book: String
    /// Server emits `"3"`, `"1-3"`, or `"1,2,3"`.
    let chapters: String
    /// `"OT"` or `"NT"`.
    let testament: String

    /// Human-facing reference, e.g. `"Genesis 1–3"` or `"Matthew 1, 2"`.
    /// Uses an en dash for ranges to match the reading UI's typography.
    var display: String {
        let pretty = chapters
            .replacingOccurrences(of: "-", with: "\u{2013}")
            .replacingOccurrences(of: ",", with: ", ")
        return "\(book) \(pretty)"
    }
}
