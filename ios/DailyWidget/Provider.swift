import WidgetKit

/// Drives the widget timeline: fetch today's reading for the configured plan,
/// cache it, and schedule the next refresh at the user's next local midnight.
struct Provider: AppIntentTimelineProvider {
    private let api = DailyReadingAPI()

    func placeholder(in context: Context) -> DailyEntry {
        DailyEntry(
            date: Date(),
            readingDate: DateTZ.todayString(),
            planName: PlanOption.bibleYear.planName,
            passages: [
                DailyPassage(book: "Genesis", chapters: "1-3", testament: "OT"),
                DailyPassage(book: "Romans", chapters: "1", testament: "NT"),
            ],
            message: nil,
            isStale: false,
            translation: .esv
        )
    }

    func snapshot(for configuration: ConfigurationAppIntent, in context: Context) async -> DailyEntry {
        await loadEntry(for: configuration)
    }

    func timeline(for configuration: ConfigurationAppIntent, in context: Context) async -> Timeline<DailyEntry> {
        let entry = await loadEntry(for: configuration)
        // Content flips at the start of the user's day; refresh just after.
        return Timeline(entries: [entry], policy: .after(DateTZ.nextMidnight()))
    }

    private func loadEntry(for configuration: ConfigurationAppIntent) async -> DailyEntry {
        let plan = configuration.plan
        let today = DateTZ.todayString()

        do {
            let response = try await api.fetch(plan: plan, date: today)
            AppGroup.saveCache(
                CachedDaily(response: response, readingDate: today, fetchedAt: Date()),
                plan: plan
            )
            return entry(from: response, readingDate: today, plan: plan, isStale: false, configuration: configuration)
        } catch {
            // Offline / backend down / rate-limited: fall back to the cache.
            if let cached = AppGroup.loadCache(plan: plan) {
                return entry(
                    from: cached.response,
                    readingDate: cached.readingDate,
                    plan: plan,
                    isStale: true,
                    configuration: configuration
                )
            }
            return DailyEntry(
                date: Date(),
                readingDate: today,
                planName: plan.planName,
                passages: [],
                message: "Unable to load reading",
                isStale: true,
                translation: configuration.translation
            )
        }
    }

    private func entry(
        from response: DailyResponse,
        readingDate: String,
        plan: PlanOption,
        isStale: Bool,
        configuration: ConfigurationAppIntent
    ) -> DailyEntry {
        let result = response.plans.first
        return DailyEntry(
            date: Date(),
            readingDate: readingDate,
            planName: result?.name ?? plan.planName,
            passages: result?.passages ?? [],
            message: result?.message,
            isStale: isStale,
            translation: configuration.translation
        )
    }
}
