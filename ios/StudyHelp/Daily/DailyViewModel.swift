import Foundation
import Observation

/// State machine for the Daily tab — the native port of the web's
/// `useDailyTab` (`web/src/daily/useDailyTab.ts`): one `/api/daily-reading`
/// fetch builds the pills and plan messages, then every pill's passages
/// load concurrently and land as they finish. A fetch-id guard drops
/// stale results when the date, translation, or plan selection changes
/// mid-flight.
@Observable
@MainActor
final class DailyViewModel {
    struct Pill: Identifiable, Equatable {
        let planID: String
        let planName: String
        let passage: DailyPassage
        /// nil while loading (when `error` is also nil).
        var parsed: ParsedPassage?
        var error: String?

        var id: String { "\(planID)|\(passage.book)|\(passage.chapters)" }
        var isLoading: Bool { parsed == nil && error == nil }
    }

    struct PlanMessage: Identifiable, Equatable {
        let planID: String
        let planName: String
        let text: String
        var id: String { "\(planID)|\(text)" }
    }

    enum Phase: Equatable {
        case idle
        case loading
        case ready
        case failed
    }

    private(set) var phase: Phase = .idle
    private(set) var pills: [Pill] = []
    private(set) var planMessages: [PlanMessage] = []
    /// Plans contributing to this load — pills carry their testament tag
    /// for a single plan, their plan name when several contribute
    /// (mirrors `DailyPanel`'s `showPlanTagOnPills`).
    private(set) var planCount = 0
    /// Index into `pills`; -1 on info-card-only days.
    var activeIndex = -1

    /// `YYYY-MM-DD` in the device timezone. Session restore replaces the
    /// default in phase ⑥.
    var selectedDate = DateTZ.todayString()

    @ObservationIgnored private let dailyAPI: DailyReadingAPI
    @ObservationIgnored private let passageAPI: PassageAPI
    @ObservationIgnored private var fetchID = 0

    init(dailyAPI: DailyReadingAPI = DailyReadingAPI(), passageAPI: PassageAPI = PassageAPI()) {
        self.dailyAPI = dailyAPI
        self.passageAPI = passageAPI
    }

    var isToday: Bool { selectedDate == DateTZ.todayString() }

    func goToday() { selectedDate = DateTZ.todayString() }

    func offsetDate(by days: Int) {
        guard let date = Self.parse(selectedDate) else {
            selectedDate = DateTZ.todayString()
            return
        }
        var cal = Calendar(identifier: .gregorian)
        cal.timeZone = .current
        if let shifted = cal.date(byAdding: .day, value: days, to: date) {
            selectedDate = DateTZ.todayString(now: shifted)
        }
    }

    var selectedDateValue: Date {
        Self.parse(selectedDate) ?? Date()
    }

    func setSelectedDate(_ date: Date) {
        selectedDate = DateTZ.todayString(now: date)
    }

    /// "Tuesday, June 9" — the daily header's date line.
    var formattedDate: String {
        guard let date = Self.parse(selectedDate) else { return selectedDate }
        let fmt = DateFormatter()
        fmt.setLocalizedDateFormatFromTemplate("EEEEMMMMd")
        return fmt.string(from: date)
    }

    func load(
        translation: TranslationOption,
        toggles: PassageToggles,
        plans: [PlanOption]
    ) async {
        fetchID += 1
        let id = fetchID
        phase = .loading
        pills = []
        planMessages = []
        activeIndex = -1

        let daily: DailyResponse
        do {
            daily = try await dailyAPI.fetch(plans: plans, date: selectedDate)
        } catch {
            if id == fetchID { phase = .failed }
            return
        }
        guard id == fetchID else { return }

        var newPills: [Pill] = []
        var messages: [PlanMessage] = []
        for plan in daily.plans {
            if let message = plan.message, !message.isEmpty {
                messages.append(PlanMessage(planID: plan.id, planName: plan.name, text: message))
            }
            for passage in plan.passages {
                newPills.append(Pill(planID: plan.id, planName: plan.name, passage: passage))
            }
        }
        pills = newPills
        planMessages = messages
        planCount = daily.plans.count
        activeIndex = newPills.isEmpty ? -1 : 0
        phase = .ready

        // Load every pill concurrently; each lands as it finishes, like the
        // web's per-pill loads. The group keeps it structured so `load`
        // returns only once the day is fully settled.
        await withTaskGroup(of: Void.self) { group in
            for index in newPills.indices {
                group.addTask { [weak self] in
                    await self?.loadPill(at: index, fetchID: id,
                                         translation: translation, toggles: toggles)
                }
            }
        }
    }

    private func loadPill(
        at index: Int,
        fetchID id: Int,
        translation: TranslationOption,
        toggles: PassageToggles
    ) async {
        let queries = Self.passageQueries(for: pills[index].passage)
        do {
            // Comma-separated chapters fan out into one request per piece
            // (the server accepts only single chapters or hyphen ranges);
            // results concatenate in query order.
            let responses = try await withThrowingTaskGroup(
                of: (Int, PassageResponse).self
            ) { group in
                for (i, q) in queries.enumerated() {
                    group.addTask { [passageAPI] in
                        (i, try await passageAPI.passage(q: q, translation: translation, toggles: toggles))
                    }
                }
                var ordered = [PassageResponse?](repeating: nil, count: queries.count)
                for try await (i, resp) in group { ordered[i] = resp }
                return ordered.compactMap { $0 }
            }
            guard id == fetchID else { return }
            let combined = responses.flatMap(\.passages)
            pills[index].parsed = PassageHTMLParser.parse(passages: combined)
        } catch {
            guard id == fetchID else { return }
            if case PassageAPI.Failure.rateLimited = error {
                pills[index].error = "Service is busy, try again in a moment"
            } else {
                pills[index].error = "Something went wrong, try again"
            }
        }
    }

    /// One `?q=` per comma piece — direct port of the web's
    /// `passageQueries` (`useDailyTab.ts`).
    static func passageQueries(for passage: DailyPassage) -> [String] {
        passage.chapters
            .split(separator: ",")
            .map { $0.trimmingCharacters(in: .whitespaces) }
            .filter { !$0.isEmpty }
            .map { "\(passage.book) \($0)" }
    }

    private static func parse(_ yyyymmdd: String) -> Date? {
        let fmt = DateFormatter()
        fmt.calendar = Calendar(identifier: .gregorian)
        fmt.timeZone = .current
        fmt.dateFormat = "yyyy-MM-dd"
        return fmt.date(from: yyyymmdd)
    }
}
