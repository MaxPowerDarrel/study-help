import SwiftUI

/// Native settings tab: translation, the six formatting toggles, and the
/// switch back to the embedded web reader. Mirrors the web's settings
/// surface (`web/src/toggles.ts` defaults, translation picker with
/// YouVersion attribution).
struct SettingsView: View {
    @Bindable var settings: ReaderSettings

    var body: some View {
        Form {
            Section {
                Picker("Translation", selection: $settings.translation) {
                    ForEach(TranslationOption.allCases, id: \.self) { t in
                        Text(t.rawValue).tag(t)
                    }
                }
            } footer: {
                if settings.translation == .niv {
                    AttributionFooter(translation: .niv)
                }
            }

            Section {
                ForEach(PlanOption.allCases, id: \.self) { plan in
                    Toggle(plan.planName, isOn: planBinding(for: plan))
                }
            } header: {
                Text("Reading plans")
            } footer: {
                Text("At least one plan stays selected; clearing the last one returns to Bible in One Year.")
            }

            Section("Formatting") {
                Toggle("Headings", isOn: $settings.headings)
                Toggle("Footnotes", isOn: $settings.footnotes)
                Toggle("Verse numbers", isOn: $settings.verseNumbers)
                Toggle("Passage references", isOn: $settings.passageReferences)
                Toggle("Cross-references", isOn: $settings.crossReferences)
                Toggle("Words of Christ in red", isOn: $settings.showWordsOfChrist)
            }

            Section("Appearance") {
                Picker("Theme", selection: $settings.theme) {
                    ForEach(ThemeChoice.allCases, id: \.self) { choice in
                        Text(choice.label).tag(choice)
                    }
                }
                .pickerStyle(.segmented)
            }

            Section {
                Toggle("Native reader (beta)", isOn: $settings.useNativeReader)
            } footer: {
                Text("Turning this off returns to the embedded web reader, which has its own settings.")
            }
        }
    }

    /// Checkbox semantics over the plan array, keeping `allCases` order;
    /// the never-empty fallback lives in `ReaderSettings.selectedPlans`.
    private func planBinding(for plan: PlanOption) -> Binding<Bool> {
        Binding(
            get: { settings.selectedPlans.contains(plan) },
            set: { isOn in
                var plans = settings.selectedPlans
                if isOn {
                    if !plans.contains(plan) {
                        plans = PlanOption.allCases.filter { plans.contains($0) || $0 == plan }
                    }
                } else {
                    plans.removeAll { $0 == plan }
                }
                settings.selectedPlans = plans
            }
        )
    }
}

/// The sheet behind the gear button while the web view is still the
/// default surface: just the beta switch — the web app keeps its own
/// in-page settings until the native reader is the surface
/// (`specs/native-reader.md`, User-facing behavior).
struct ReaderModeSheet: View {
    @Bindable var settings: ReaderSettings
    @Environment(\.dismiss) private var dismiss

    var body: some View {
        NavigationStack {
            Form {
                Section {
                    Toggle("Native reader (beta)", isOn: $settings.useNativeReader)
                } footer: {
                    Text("Renders passages natively instead of loading the web app. You can switch back anytime from the native Settings tab.")
                }
            }
            .navigationTitle("Reader")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .confirmationAction) {
                    Button("Done") { dismiss() }
                }
            }
        }
        .presentationDetents([.medium])
    }
}
