import SwiftUI

/// The native reading surface: Read / Daily / Settings tabs
/// (`specs/native-reader.md`). Daily is a placeholder until phase ④ —
/// the web reader (one switch away in Settings) covers it meanwhile.
struct ReaderRootView: View {
    var settings: ReaderSettings

    var body: some View {
        TabView {
            ReadTabView(settings: settings)
                .tabItem { Label("Read", systemImage: "book") }

            dailyPlaceholder
                .tabItem { Label("Daily", systemImage: "calendar") }

            SettingsView(settings: settings)
                .tabItem { Label("Settings", systemImage: "gearshape") }
        }
    }

    private var dailyPlaceholder: some View {
        VStack(spacing: 12) {
            Image(systemName: "calendar")
                .font(.largeTitle)
                .foregroundStyle(.secondary)
            Text("The native Daily tab is coming in the next build.")
                .multilineTextAlignment(.center)
            Text("Until then, switch to the web reader in Settings for daily readings.")
                .font(.footnote)
                .foregroundStyle(.secondary)
                .multilineTextAlignment(.center)
        }
        .padding()
    }
}
