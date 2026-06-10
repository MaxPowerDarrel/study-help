import SwiftUI

/// The native reading surface: Read / Daily / Settings tabs
/// (`specs/native-reader.md`).
struct ReaderRootView: View {
    enum Tab {
        case read, daily, settings
    }

    var settings: ReaderSettings
    @State private var selection: Tab

    init(settings: ReaderSettings) {
        self.settings = settings
        var initial = Tab.read
        #if DEBUG
        // Launch-time override for simulator screenshot automation, e.g.
        // `SIMCTL_CHILD_STUDYHELP_TAB=daily xcrun simctl launch …`.
        if ProcessInfo.processInfo.environment["STUDYHELP_TAB"] == "daily" {
            initial = .daily
        }
        #endif
        _selection = State(initialValue: initial)
    }

    var body: some View {
        TabView(selection: $selection) {
            ReadTabView(settings: settings)
                .tabItem { Label("Read", systemImage: "book") }
                .tag(Tab.read)

            DailyTabView(settings: settings)
                .tabItem { Label("Daily", systemImage: "calendar") }
                .tag(Tab.daily)

            SettingsView(settings: settings)
                .tabItem { Label("Settings", systemImage: "gearshape") }
                .tag(Tab.settings)
        }
        .onOpenURL { url in
            // `studyhelp://daily` — a widget tap lands on the Daily tab,
            // which always loads against the current day by default.
            if url.host == "daily" {
                selection = .daily
            }
        }
        .preferredColorScheme(colorScheme)
    }

    /// The web view keeps the web app's own theme; this override applies
    /// only to the native surface.
    private var colorScheme: ColorScheme? {
        switch settings.theme {
        case .system: return nil
        case .light: return .light
        case .dark: return .dark
        }
    }
}
