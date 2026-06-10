import SwiftUI

/// Root switch between the two reading surfaces (`specs/native-reader.md`):
/// the native SwiftUI reader (the default since parity) and the embedded
/// SPA, kept as a fallback escape hatch. While the web view is the surface
/// it carries exactly one piece of native chrome — a gear button opening
/// the sheet that hosts the switch back.
struct ContentView: View {
    @State private var settings = ReaderSettings()
    @State private var reloadToken = 0
    @State private var showModeSheet = false

    var body: some View {
        if settings.useNativeReader {
            ReaderRootView(settings: settings)
            // `studyhelp://daily` routing to the native Daily tab lands in
            // phase ⑤; until then a widget tap just foregrounds the app.
        } else {
            WebView(url: Config.backendBaseURL, reloadToken: reloadToken)
                .ignoresSafeArea(.container, edges: .bottom)
                .overlay(alignment: .topTrailing) {
                    Button {
                        showModeSheet = true
                    } label: {
                        Image(systemName: "gearshape.fill")
                            .font(.caption)
                            .foregroundStyle(.secondary)
                            .padding(8)
                            .background(.ultraThinMaterial, in: Circle())
                    }
                    .padding(.trailing, 12)
                    .accessibilityLabel("Reader settings")
                }
                .sheet(isPresented: $showModeSheet) {
                    ReaderModeSheet(settings: settings)
                }
                .onOpenURL { _ in
                    // `studyhelp://daily` — a widget tap brings the app
                    // forward. The SPA restores to the Daily tab on its own
                    // via restore.ts; reloading guarantees it reflects the
                    // current day on a cold launch that had been
                    // backgrounded across midnight.
                    reloadToken += 1
                }
        }
    }
}
