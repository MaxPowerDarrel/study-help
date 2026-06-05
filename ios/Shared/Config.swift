import Foundation

/// Single source of truth for where the native clients reach the backend.
///
/// The app's web view and the widget's `URLSession` both hit this origin.
/// It must be **HTTPS** (App Transport Security blocks plain HTTP / bare IPs
/// by default), which the Lightsail deployment already satisfies via Caddy
/// (see `specs/deploy-aws.md`).
enum Config {
    /// Production origin that serves the study-help SPA and JSON API.
    /// Override with `BACKEND_BASE_URL` for local development against
    /// `go run .` on your Mac (e.g. `http://<mac-lan-ip>:8080`, which then
    /// needs an ATS exception — see `ios/README.md`).
    static let backendBaseURL: URL = {
        if let raw = ProcessInfo.processInfo.environment["BACKEND_BASE_URL"],
           let url = URL(string: raw) {
            return url
        }
        return URL(string: "https://study.darrel.io")!
    }()
}
