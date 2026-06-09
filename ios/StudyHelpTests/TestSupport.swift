import Foundation
@testable import StudyHelp

/// URLProtocol stub shared by the API and view-model tests: set `handler`
/// to script (status, body) per request, and reset it in tearDown.
final class StubURLProtocol: URLProtocol {
    static var handler: ((URLRequest) -> (Int, Data))?

    override class func canInit(with request: URLRequest) -> Bool { true }
    override class func canonicalRequest(for request: URLRequest) -> URLRequest { request }
    override func stopLoading() {}

    override func startLoading() {
        guard let url = request.url, let handler = Self.handler else {
            client?.urlProtocol(self, didFailWithError: URLError(.badURL))
            return
        }
        let (status, data) = handler(request)
        let response = HTTPURLResponse(
            url: url, statusCode: status, httpVersion: nil, headerFields: nil
        )!
        client?.urlProtocol(self, didReceive: response, cacheStoragePolicy: .notAllowed)
        client?.urlProtocol(self, didLoad: data)
        client?.urlProtocolDidFinishLoading(self)
    }
}

/// A `PassageAPI` whose session is routed through `StubURLProtocol`.
func makeStubbedPassageAPI() -> PassageAPI {
    let config = URLSessionConfiguration.ephemeral
    config.protocolClasses = [StubURLProtocol.self]
    return PassageAPI(
        baseURL: URL(string: "https://example.test")!,
        session: URLSession(configuration: config)
    )
}
