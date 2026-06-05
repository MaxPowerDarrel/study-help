import Foundation

/// Date helpers that keep the widget aligned with the server's notion of
/// "today," which is computed in the caller's IANA timezone
/// (`internal/server/daily.go` requires a `tz` query param).
enum DateTZ {
    /// Today as `YYYY-MM-DD` in `timeZone` (defaults to the device timezone).
    static func todayString(timeZone: TimeZone = .current, now: Date = Date()) -> String {
        var cal = Calendar(identifier: .gregorian)
        cal.timeZone = timeZone
        let c = cal.dateComponents([.year, .month, .day], from: now)
        return String(format: "%04d-%02d-%02d", c.year ?? 0, c.month ?? 0, c.day ?? 0)
    }

    /// The next local midnight strictly after `now`, used as the widget's
    /// timeline reload boundary so the reading flips over at the start of the
    /// user's day.
    static func nextMidnight(after now: Date = Date(), timeZone: TimeZone = .current) -> Date {
        var cal = Calendar(identifier: .gregorian)
        cal.timeZone = timeZone
        let startOfToday = cal.startOfDay(for: now)
        return cal.date(byAdding: .day, value: 1, to: startOfToday)
            ?? now.addingTimeInterval(60 * 60)
    }
}
