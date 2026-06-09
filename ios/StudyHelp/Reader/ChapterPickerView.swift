import SwiftUI

/// Book + chapter picker over the shared `Canon`, with an optional end
/// chapter for ranges — the native counterpart of the web's picker pane.
struct ChapterPickerView: View {
    let current: Canon.ChapterRef
    let currentRangeEnd: Int?
    let onSelect: (Canon.ChapterRef, Int?) -> Void

    @Environment(\.dismiss) private var dismiss
    @State private var bookIndex: Int
    @State private var chapter: Int
    /// 0 means "single chapter".
    @State private var endChapter: Int

    init(
        current: Canon.ChapterRef,
        currentRangeEnd: Int?,
        onSelect: @escaping (Canon.ChapterRef, Int?) -> Void
    ) {
        self.current = current
        self.currentRangeEnd = currentRangeEnd
        self.onSelect = onSelect
        _bookIndex = State(initialValue: current.bookIndex)
        _chapter = State(initialValue: current.chapter)
        _endChapter = State(initialValue: currentRangeEnd ?? 0)
    }

    private var book: Canon.Book { Canon.books[bookIndex] }

    var body: some View {
        NavigationStack {
            Form {
                Picker("Book", selection: $bookIndex) {
                    ForEach(Canon.books.indices, id: \.self) { i in
                        Text(Canon.books[i].name).tag(i)
                    }
                }
                Picker("Chapter", selection: $chapter) {
                    ForEach(1...book.chapters, id: \.self) { c in
                        Text("\(c)").tag(c)
                    }
                }
                Picker("Through", selection: $endChapter) {
                    Text("Single chapter").tag(0)
                    if chapter < book.chapters {
                        ForEach((chapter + 1)...book.chapters, id: \.self) { c in
                            Text("Chapter \(c)").tag(c)
                        }
                    }
                }
            }
            .navigationTitle("Choose passage")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .confirmationAction) {
                    Button("Read") {
                        onSelect(
                            Canon.ChapterRef(bookIndex: bookIndex, chapter: chapter),
                            endChapter > chapter ? endChapter : nil
                        )
                        dismiss()
                    }
                }
                ToolbarItem(placement: .cancellationAction) {
                    Button("Cancel") { dismiss() }
                }
            }
            .onChange(of: bookIndex) {
                chapter = min(chapter, book.chapters)
                endChapter = 0
            }
            .onChange(of: chapter) {
                if endChapter <= chapter { endChapter = 0 }
            }
        }
    }
}
