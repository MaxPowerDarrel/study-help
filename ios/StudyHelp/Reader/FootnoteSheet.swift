import SwiftUI

/// Sheet presenting one footnote's body — the content behind a tapped
/// inline marker, already collected from the passage's footnotes tail by
/// the parser (no extra fetch).
struct FootnoteSheet: View {
    let note: Footnote
    @Environment(\.dismiss) private var dismiss

    var body: some View {
        NavigationStack {
            ScrollView {
                Text(PassageRenderer().attributedString(for: note.runs))
                    .lineSpacing(4)
                    .frame(maxWidth: .infinity, alignment: .leading)
                    .padding()
            }
            .navigationTitle(title)
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .confirmationAction) {
                    Button("Done") { dismiss() }
                }
            }
        }
        .presentationDetents([.medium])
    }

    private var title: String {
        note.reference.isEmpty ? "Footnote" : "Footnote on \(note.reference)"
    }
}
