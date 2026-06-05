import SwiftUI
import WebKit

/// A `WKWebView` that loads the hosted SPA. Uses the default (persistent) data
/// store so the SPA's `localStorage` (translation, plan selection, theme, last
/// location) survives across launches — matching the browser experience.
struct WebView: UIViewRepresentable {
    let url: URL
    /// Bumping this from the parent triggers a reload (e.g. on a widget tap).
    var reloadToken: Int = 0

    func makeCoordinator() -> Coordinator { Coordinator() }

    func makeUIView(context: Context) -> WKWebView {
        let config = WKWebViewConfiguration()
        config.websiteDataStore = .default() // persistent: keeps localStorage

        let webView = WKWebView(frame: .zero, configuration: config)
        webView.allowsBackForwardNavigationGestures = true
        context.coordinator.webView = webView

        let refresh = UIRefreshControl()
        refresh.addTarget(
            context.coordinator,
            action: #selector(Coordinator.handleRefresh(_:)),
            for: .valueChanged
        )
        webView.scrollView.refreshControl = refresh

        webView.load(URLRequest(url: url))
        return webView
    }

    func updateUIView(_ webView: WKWebView, context: Context) {
        guard context.coordinator.lastReloadToken != reloadToken else { return }
        context.coordinator.lastReloadToken = reloadToken
        webView.reload()
    }

    final class Coordinator: NSObject {
        weak var webView: WKWebView?
        var lastReloadToken = 0

        @objc func handleRefresh(_ sender: UIRefreshControl) {
            webView?.reload()
            sender.endRefreshing()
        }
    }
}
