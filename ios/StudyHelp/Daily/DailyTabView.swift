import SwiftUI

/// The Daily tab: date navigation + daily translation picker in the
/// header, plan message cards, the pill bar, and the active pill's
/// passage — the native counterpart of the web's `DailyPanel`.
struct DailyTabView: View {
    @Bindable var settings: ReaderSettings
    @State private var model = DailyViewModel()

    var body: some View {
        VStack(spacing: 0) {
            header
            Divider()
            content
            AttributionFooter(translation: settings.translation)
        }
        .task(id: loadKey) {
            await model.load(
                translation: settings.translation,
                toggles: settings.passageToggles,
                plans: settings.selectedPlans
            )
        }
    }

    /// Reload whenever the date, translation, plan selection, or a
    /// server-sent toggle changes. (The web defers toggle changes to the
    /// next reload; refetching immediately is deliberate parity with the
    /// native Read tab.)
    private var loadKey: String {
        "\(model.selectedDate)|\(settings.translation.rawValue)|" +
        settings.selectedPlans.map(\.rawValue).joined(separator: ",") + "|" +
        "\(settings.headings)|\(settings.footnotes)|\(settings.verseNumbers)|" +
        "\(settings.passageReferences)|\(settings.crossReferences)"
    }

    private var dateBinding: Binding<Date> {
        Binding(
            get: { model.selectedDateValue },
            set: { model.setSelectedDate($0) }
        )
    }

    private var header: some View {
        VStack(spacing: 4) {
            HStack(spacing: 8) {
                Button {
                    model.offsetDate(by: -1)
                } label: {
                    Image(systemName: "chevron.left")
                }
                .accessibilityLabel("Previous day")

                DatePicker("Date", selection: dateBinding, displayedComponents: .date)
                    .labelsHidden()

                Button {
                    model.offsetDate(by: 1)
                } label: {
                    Image(systemName: "chevron.right")
                }
                .accessibilityLabel("Next day")

                if !model.isToday {
                    Button("Today") { model.goToday() }
                        .font(.callout)
                }

                Spacer()

                Picker("Translation", selection: $settings.translation) {
                    ForEach(TranslationOption.allCases, id: \.self) { t in
                        Text(t.rawValue).tag(t)
                    }
                }
                .pickerStyle(.menu)
                .accessibilityLabel("Translation")
            }
            Text(model.formattedDate)
                .font(.subheadline)
                .foregroundStyle(.secondary)
                .frame(maxWidth: .infinity, alignment: .leading)
        }
        .padding(.horizontal)
        .padding(.vertical, 8)
    }

    @ViewBuilder
    private var content: some View {
        switch model.phase {
        case .idle, .loading:
            Spacer()
            ProgressView()
            Spacer()
        case .failed:
            Spacer()
            Text("Daily reading unavailable. Try again later.")
                .foregroundStyle(.secondary)
                .padding()
            Spacer()
        case .ready:
            readyContent
        }
    }

    @ViewBuilder
    private var readyContent: some View {
        if !model.planMessages.isEmpty {
            VStack(spacing: 8) {
                ForEach(model.planMessages) { message in
                    VStack(alignment: .leading, spacing: 2) {
                        if model.planCount > 1 {
                            Text(message.planName)
                                .font(.caption.weight(.semibold))
                                .foregroundStyle(.secondary)
                        }
                        Text(message.text)
                    }
                    .frame(maxWidth: .infinity, alignment: .leading)
                    .padding(12)
                    .background(Color(.secondarySystemBackground), in: RoundedRectangle(cornerRadius: 10))
                }
            }
            .padding(.horizontal)
            .padding(.top, 8)
        }

        if model.pills.isEmpty && model.planMessages.isEmpty {
            Spacer()
            Text("No reading for this day.")
                .foregroundStyle(.secondary)
            Spacer()
        } else {
            if !model.pills.isEmpty {
                DailyPillBar(
                    pills: model.pills,
                    activeIndex: Binding(
                        get: { model.activeIndex },
                        set: { model.activeIndex = $0 }
                    ),
                    showPlanTag: model.planCount > 1
                )
            }
            activePillContent
        }
    }

    @ViewBuilder
    private var activePillContent: some View {
        if let pill = model.pills.indices.contains(model.activeIndex)
            ? model.pills[model.activeIndex] : nil {
            if pill.isLoading {
                Spacer()
                ProgressView()
                Spacer()
            } else if let error = pill.error {
                Spacer()
                Text(error).foregroundStyle(.secondary).padding()
                Spacer()
            } else if let parsed = pill.parsed {
                PassageView(
                    parsed: parsed,
                    showWordsOfChrist: settings.showWordsOfChrist,
                    contentID: "\(model.selectedDate)|\(pill.id)"
                )
            }
        } else {
            Spacer()
        }
    }
}
