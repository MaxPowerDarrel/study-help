import SwiftUI

/// The whole app is the hosted SPA in a web view. The native shell exists only
/// to make the home-screen widget possible (widgets can't be web) — it adds no
/// second reader. See `PROJECT_CONSTITUTION.md` §1 and `specs/native-ios.md`.
struct ContentView: View {
    @State private var reloadToken = 0

    var body: some View {
        WebView(url: Config.backendBaseURL, reloadToken: reloadToken)
            .ignoresSafeArea(.container, edges: .bottom)
            .onOpenURL { _ in
                // `studyhelp://daily` — a widget tap brings the app forward.
                // The SPA restores to the Daily tab on its own via restore.ts;
                // reloading guarantees it reflects the current day on a cold
                // launch that had been backgrounded across midnight.
                reloadToken += 1
            }
    }
}
