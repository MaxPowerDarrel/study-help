import Foundation

/// On-device shared storage between the app and the widget extension.
///
/// This is where the widget caches its last successful fetch so it can render
/// when the network or backend is unavailable. Per `PROJECT_CONSTITUTION.md`
/// §4, all native-client state lives here on the device — never on the server.
/// Only daily *references* are cached here, never scripture text (ToU).
enum AppGroup {
    /// Must match the App Group capability declared in both targets'
    /// `.entitlements` files.
    static let identifier = "group.com.darrelross.studyhelp"

    static var defaults: UserDefaults {
        UserDefaults(suiteName: identifier) ?? .standard
    }

    private static func key(for plan: PlanOption) -> String { "daily.\(plan.rawValue)" }

    static func saveCache(_ cached: CachedDaily, plan: PlanOption) {
        guard let data = try? JSONEncoder().encode(cached) else { return }
        defaults.set(data, forKey: key(for: plan))
    }

    static func loadCache(plan: PlanOption) -> CachedDaily? {
        guard let data = defaults.data(forKey: key(for: plan)) else { return nil }
        return try? JSONDecoder().decode(CachedDaily.self, from: data)
    }
}

/// The last good daily-reading response for a plan, with the reading date it
/// was for and when it was fetched.
struct CachedDaily: Codable {
    let response: DailyResponse
    let readingDate: String
    let fetchedAt: Date
}
