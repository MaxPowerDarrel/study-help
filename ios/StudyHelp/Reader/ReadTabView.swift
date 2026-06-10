import SwiftUI

/// The Read tab: passage header (prev / picker / next), the rendered
/// passage, and the NIV attribution footer. Reloads whenever the location,
/// translation, or a server-sent toggle changes — the words-of-Christ
/// toggle re-renders without refetching, exactly like the web.
struct ReadTabView: View {
    @Bindable var settings: ReaderSettings
    @State private var model: ReadViewModel
    @State private var showPicker = false
    private let restore: SessionRestore

    init(settings: ReaderSettings, restore: SessionRestore = SessionRestore()) {
        self.settings = settings
        self.restore = restore
        _model = State(initialValue: ReadViewModel(location: restore.storedReadRef()))
    }

    var body: some View {
        VStack(spacing: 0) {
            header
            Divider()
            content
            AttributionFooter(translation: settings.translation)
        }
        .onChange(of: model.location) {
            restore.saveReadRef(model.location)
        }
        .task(id: loadKey) {
            await model.load(
                translation: settings.translation,
                toggles: settings.passageToggles
            )
        }
        .sheet(isPresented: $showPicker) {
            ChapterPickerView(
                current: model.location,
                currentRangeEnd: model.rangeEnd
            ) { ref, end in
                model.select(ref, rangeEnd: end)
            }
        }
    }

    /// One value that changes whenever a reload-worthy input changes.
    private var loadKey: String {
        "\(model.query)|\(settings.translation.rawValue)|" +
        "\(settings.headings)|\(settings.footnotes)|\(settings.verseNumbers)|" +
        "\(settings.passageReferences)|\(settings.crossReferences)"
    }

    private var title: String {
        model.canonical.isEmpty ? model.query : model.canonical
    }

    private var header: some View {
        HStack {
            Button {
                model.goPrevious()
            } label: {
                Image(systemName: "chevron.left")
            }
            .disabled(!model.canGoPrevious)
            .accessibilityLabel("Previous chapter")

            Spacer()

            Button {
                showPicker = true
            } label: {
                HStack(spacing: 6) {
                    Text(title).font(.headline)
                    Image(systemName: "chevron.down").font(.caption2)
                }
            }
            .buttonStyle(.plain)
            .accessibilityLabel("Choose passage")

            Spacer()

            Button {
                model.goNext()
            } label: {
                Image(systemName: "chevron.right")
            }
            .disabled(!model.canGoNext)
            .accessibilityLabel("Next chapter")
        }
        .padding(.horizontal)
        .padding(.vertical, 10)
    }

    @ViewBuilder
    private var content: some View {
        switch model.phase {
        case .idle, .loading:
            Spacer()
            ProgressView()
            Spacer()
        case .rateLimited:
            statusView("Service is busy, try again in a moment")
        case .failed:
            statusView("Could not load the passage")
        case .loaded:
            if let parsed = model.parsed {
                PassageView(
                    parsed: parsed,
                    showWordsOfChrist: settings.showWordsOfChrist,
                    contentID: model.canonical
                )
            }
        }
    }

    private func statusView(_ message: String) -> some View {
        VStack(spacing: 12) {
            Spacer()
            Text(message).foregroundStyle(.secondary)
            Button("Retry") {
                Task {
                    await model.load(
                        translation: settings.translation,
                        toggles: settings.passageToggles
                    )
                }
            }
            .buttonStyle(.bordered)
            Spacer()
        }
        .frame(maxWidth: .infinity)
    }
}
