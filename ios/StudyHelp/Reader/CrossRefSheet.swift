import Foundation
import Observation
import SwiftUI

/// One cross-reference lookup: fetch `/api/crossref` for the reference
/// list lifted from a tapped marker, parse with the shared pipeline, and
/// publish — the native counterpart of the web popover's loading / error /
/// busy states (`web/src/PassageArticle.tsx`). ESV-only by construction:
/// NIV passages carry no markers.
@Observable
@MainActor
final class CrossrefLookup: Identifiable {
    enum Phase: Equatable {
        case loading
        case ready
        case rateLimited
        case failed
    }

    private(set) var phase: Phase = .loading
    private(set) var canonical = ""
    private(set) var parsed: ParsedPassage?

    @ObservationIgnored private let api: PassageAPI

    init(api: PassageAPI = PassageAPI()) {
        self.api = api
    }

    func load(q: String) async {
        do {
            let resp = try await api.crossref(q: q)
            parsed = PassageHTMLParser.parse(passages: resp.passages)
            canonical = resp.canonical
            phase = .ready
        } catch PassageAPI.Failure.rateLimited {
            phase = .rateLimited
        } catch {
            phase = .failed
        }
    }
}

/// Sheet presenting the verse text behind a tapped cross-reference
/// marker. The crossref endpoint serves verse-level text with headings
/// and footnotes hard-off, so the parsed blocks render with the same
/// shared block view as the main surface.
struct CrossRefSheet: View {
    let lookup: CrossrefLookup
    let showWordsOfChrist: Bool
    @Environment(\.dismiss) private var dismiss

    var body: some View {
        NavigationStack {
            content
                .navigationTitle("Cross-references")
                .navigationBarTitleDisplayMode(.inline)
                .toolbar {
                    ToolbarItem(placement: .confirmationAction) {
                        Button("Done") { dismiss() }
                    }
                }
        }
        .presentationDetents([.medium, .large])
    }

    @ViewBuilder
    private var content: some View {
        switch lookup.phase {
        case .loading:
            ProgressView()
                .frame(maxWidth: .infinity, maxHeight: .infinity)
        case .rateLimited:
            statusText("Service is busy, try again in a moment")
        case .failed:
            statusText("Could not load cross-reference")
        case .ready:
            if let parsed = lookup.parsed {
                ScrollView {
                    VStack(alignment: .leading, spacing: 0) {
                        PassageBlocksView(blocks: parsed.blocks, showWordsOfChrist: showWordsOfChrist)
                    }
                    .frame(maxWidth: .infinity, alignment: .leading)
                    .padding(.horizontal)
                    .padding(.bottom, 24)
                }
            }
        }
    }

    private func statusText(_ message: String) -> some View {
        Text(message)
            .foregroundStyle(.secondary)
            .frame(maxWidth: .infinity, maxHeight: .infinity)
    }
}
