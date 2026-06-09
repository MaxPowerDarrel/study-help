import SwiftUI

/// Per-translation attribution, mirroring the web's
/// `web/src/translations/Attribution.tsx`: YouVersion's Platform ToS expects
/// a visible "Powered by YouVersion" mark whenever NIV passages render.
/// ESV needs nothing here — its attribution arrives inside the passage HTML
/// and the parser's never-drop rule carries it into the rendered text.
struct AttributionFooter: View {
    let translation: TranslationOption

    var body: some View {
        if translation == .niv {
            HStack(spacing: 4) {
                Text("Powered by")
                Link("YouVersion", destination: URL(string: "https://www.youversion.com/")!)
            }
            .font(.footnote)
            .foregroundStyle(.secondary)
            .frame(maxWidth: .infinity)
            .padding(.vertical, 8)
        }
    }
}
