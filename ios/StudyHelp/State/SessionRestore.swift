import Foundation

/// "Where the user was" — the native port of `web/src/restore.ts` onto the
/// App Group `UserDefaults`: active tab, read-tab passage, and daily-tab
/// date, so reopening the app feels like resuming, not restarting. See
/// `specs/restore-last-location.md` for the design this mirrors.
struct SessionRestore {
    enum Tab: String {
        case read
        case daily
    }

    var store: UserDefaults = AppGroup.defaults

    private enum Keys {
        static let tab = "reader.active-tab"
        static let readRef = "reader.read-location"
        static let dailyDate = "reader.daily-date"
    }

    /// John 3 — the same starting passage as the web reader.
    static let defaultReadRef = Canon.ChapterRef(bookIndex: 42, chapter: 3)

    // MARK: Active tab

    func storedTab() -> Tab {
        store.string(forKey: Keys.tab).flatMap(Tab.init(rawValue:)) ?? .read
    }

    func saveTab(_ tab: Tab) {
        store.set(tab.rawValue, forKey: Keys.tab)
    }

    // MARK: Read-tab passage

    /// Falls back to John 3 when absent, corrupt, or out of canon bounds
    /// (a stale install must never restore into a crash).
    func storedReadRef() -> Canon.ChapterRef {
        guard let data = store.data(forKey: Keys.readRef),
              let ref = try? JSONDecoder().decode(Canon.ChapterRef.self, from: data),
              Canon.isValid(ref)
        else { return Self.defaultReadRef }
        return ref
    }

    func saveReadRef(_ ref: Canon.ChapterRef) {
        guard let data = try? JSONEncoder().encode(ref) else { return }
        store.set(data, forKey: Keys.readRef)
    }

    // MARK: Daily-tab date (with snap-to-today)

    /// The stored entry pairs the selected `date` with the `savedOn`
    /// calendar day. If `savedOn` is not today, the entry is expired and
    /// today wins — pinning a non-today date is meaningful within a day,
    /// stale across days.
    private struct StoredDailyDate: Codable {
        let date: String
        let savedOn: String
    }

    func storedDailyDate(today: String = DateTZ.todayString()) -> String {
        guard let data = store.data(forKey: Keys.dailyDate),
              let entry = try? JSONDecoder().decode(StoredDailyDate.self, from: data),
              entry.savedOn == today
        else { return today }
        return entry.date
    }

    func saveDailyDate(_ date: String, today: String = DateTZ.todayString()) {
        let entry = StoredDailyDate(date: date, savedOn: today)
        guard let data = try? JSONEncoder().encode(entry) else { return }
        store.set(data, forKey: Keys.dailyDate)
    }
}
