import Foundation
import SwiftSoup

/// Converts provider-rendered passage HTML into `ParsedPassage`. One walker
/// handles both markup dialects — the class vocabularies don't collide, so
/// no per-translation mode is needed (the same parser also serves
/// `/api/crossref` responses, which are always ESV).
///
/// Design rule: **never drop content.** Unknown tags and classes render as
/// plain styled text with links preserved — that is both the safety net for
/// upstream markup drift and the guarantee that ESV's in-HTML attribution
/// (`a.copyright`) survives into the native rendering. Unrecognized classes
/// are reported via `ParsedPassage.unknownClasses` so fixtures can pin the
/// markup as fully understood.
enum PassageHTMLParser {

    /// Parse one HTML blob (one element of the `passages` array).
    static func parse(html: String) -> ParsedPassage {
        let builder = Builder()
        do {
            let doc = try SwiftSoup.parseBodyFragment(html)
            if let body = doc.body() {
                builder.walkBlockChildren(of: body)
            }
        } catch {
            // SwiftSoup is lenient; reaching here is exceptional. Degrade to
            // the raw text rather than showing nothing.
            builder.passage.blocks = [
                PassageBlock(kind: .prose, runs: [InlineRun(text: html)])
            ]
        }
        #if DEBUG
        if !builder.passage.unknownClasses.isEmpty {
            print("PassageHTMLParser: unrecognized classes:",
                  builder.passage.unknownClasses.sorted().joined(separator: ", "))
        }
        #endif
        return builder.passage
    }

    /// Parse a whole `passages` array (e.g. each chapter of a range) into
    /// one document. Footnote ids are namespaced per passage (`p0-f1-1`)
    /// because each chapter restarts its own `f1-1` numbering.
    static func parse(passages: [String]) -> ParsedPassage {
        var combined = ParsedPassage()
        for (index, html) in passages.enumerated() {
            var part = parse(html: html)
            let prefix = "p\(index)-"
            for i in part.blocks.indices {
                for j in part.blocks[i].runs.indices {
                    if let id = part.blocks[i].runs[j].footnoteID {
                        part.blocks[i].runs[j].footnoteID = prefix + id
                    }
                }
            }
            for i in part.footnotes.indices {
                part.footnotes[i].id = prefix + part.footnotes[i].id
            }
            combined.blocks += part.blocks
            combined.footnotes += part.footnotes
            combined.unknownClasses.formUnion(part.unknownClasses)
        }
        return combined
    }

    // MARK: - Walker

    private struct InlineContext {
        var verseNumber = false
        var wordsOfChrist = false
        var italic = false
        var bold = false
        var smallCaps = false
        var superscript = false
        var crossRef: String?
        var footnoteID: String?
        var link: URL?
    }

    private final class Builder {
        var passage = ParsedPassage()

        /// Every class either provider emits today. Anything outside this
        /// set still renders (pass-through) but is reported as unknown.
        static let knownClasses: Set<String> = [
            // ESV block structure
            "extra_text", "psalm-title", "block-indent", "starts-chapter",
            "footnotes", "line", "indent", "begin-line-group", "end-line-group",
            // ESV inline
            "verse-num", "chapter-num", "inline", "woc", "cf", "fn",
            "footnote", "footnote-ref", "copyright", "divine-name", "catch-word",
            // ESV footnote-tail <note> types
            "translation", "alternative",
            // YouVersion (NIV) block structure
            "cl", "s1", "yv-h", "d", "p", "q1", "q2",
            // YouVersion inline
            "yv-v", "yv-vlbl", "wj", "nd",
        ]

        static let blockTags: Set<String> = [
            "div", "p", "h1", "h2", "h3", "h4", "h5", "h6",
            "ul", "ol", "table", "blockquote", "section", "article", "hr",
        ]

        func recordClasses(of element: Element) {
            guard let classes = try? element.classNames() else { return }
            for cls in classes where !Self.knownClasses.contains(cls) {
                passage.unknownClasses.insert(cls)
            }
        }

        // MARK: Block level

        /// Walk an element's children as a sequence of blocks. Loose inline
        /// content between block children is wrapped into prose blocks so
        /// nothing is dropped.
        func walkBlockChildren(of element: Element) {
            var pendingInline: [InlineRun] = []
            for node in element.getChildNodes() {
                if let child = node as? Element, Self.blockTags.contains(child.tagName()) {
                    flushProse(&pendingInline)
                    walkBlock(child)
                } else {
                    appendInline(node: node, context: InlineContext(), into: &pendingInline)
                }
            }
            flushProse(&pendingInline)
        }

        func flushProse(_ runs: inout [InlineRun]) {
            let trimmed = trimmedRuns(runs)
            if !trimmed.isEmpty {
                passage.blocks.append(PassageBlock(kind: .prose, runs: trimmed))
            }
            runs = []
        }

        func walkBlock(_ element: Element) {
            recordClasses(of: element)
            let tag = element.tagName()
            let classes = (try? element.classNames()) ?? []

            if tag == "div" && classes.contains("footnotes") {
                parseFootnotes(in: element)
                return
            }
            if tag == "hr" { return }
            if tag == "h2" || classes.contains("cl") {
                appendBlock(.passageTitle, from: element)
                return
            }
            if tag == "h3" || classes.contains("yv-h") {
                appendBlock(.heading, from: element)
                return
            }
            if tag == "h4" || tag == "h5" || tag == "h6" || classes.contains("d") {
                appendBlock(.subtitle, from: element)
                return
            }
            if tag == "p" && classes.contains("block-indent") {
                walkPoetryContainer(element)
                return
            }
            if classes.contains("q1") || classes.contains("q2") {
                appendPoetryLine(from: element, indent: classes.contains("q2") ? 1 : 0)
                return
            }
            // Container recursion: a wrapper whose children are themselves
            // blocks (YouVersion wraps the whole passage in one <div>).
            let hasBlockChildren = element.children().contains {
                Self.blockTags.contains($0.tagName())
            }
            if hasBlockChildren {
                walkBlockChildren(of: element)
                return
            }
            // Plain paragraph — and the never-drop fallback for any
            // unrecognized block tag.
            appendBlock(.prose, from: element)
        }

        /// ESV poetry: `p.block-indent` holds `span.line` / `span.indent.line`
        /// children separated by `<br/>`, with literal `&nbsp;` indentation.
        /// Each line span becomes its own block; the nbsp prefix is dropped
        /// in favor of the structural indent level.
        func walkPoetryContainer(_ container: Element) {
            for child in container.children() {
                recordClasses(of: child)
                let classes = (try? child.classNames()) ?? []
                guard classes.contains("line") else { continue } // skip br, line-group markers
                appendPoetryLine(from: child, indent: classes.contains("indent") ? 1 : 0)
            }
        }

        func appendPoetryLine(from element: Element, indent: Int) {
            var runs: [InlineRun] = []
            for node in element.getChildNodes() {
                appendInline(node: node, context: InlineContext(), into: &runs)
            }
            runs = trimmedRuns(stripLeadingIndent(runs))
            if !runs.isEmpty {
                passage.blocks.append(PassageBlock(kind: .poetryLine(indent: indent), runs: runs))
            }
        }

        func appendBlock(_ kind: PassageBlock.Kind, from element: Element) {
            var runs: [InlineRun] = []
            for node in element.getChildNodes() {
                appendInline(node: node, context: InlineContext(), into: &runs)
            }
            runs = trimmedRuns(runs)
            if !runs.isEmpty {
                passage.blocks.append(PassageBlock(kind: kind, runs: runs))
            }
        }

        // MARK: Inline level

        func appendInline(node: Node, context: InlineContext, into runs: inout [InlineRun]) {
            if let text = node as? TextNode {
                append(text: collapseWhitespace(text.getWholeText()), context: context, into: &runs)
                return
            }
            guard let element = node as? Element else { return }
            recordClasses(of: element)
            let tag = element.tagName()
            let classes = (try? element.classNames()) ?? []

            if tag == "br" {
                append(text: "\n", context: context, into: &runs)
                return
            }
            // Empty boundary/grouping markers — nothing to render.
            if classes.contains("yv-v")
                || classes.contains("begin-line-group")
                || classes.contains("end-line-group") {
                return
            }

            var ctx = context
            if tag == "b" || tag == "strong" { ctx.bold = true }
            if tag == "i" || tag == "em" { ctx.italic = true }
            if tag == "sup" { ctx.superscript = true }
            if classes.contains("verse-num") || classes.contains("chapter-num")
                || classes.contains("yv-vlbl") {
                ctx.verseNumber = true
            }
            if classes.contains("woc") || classes.contains("wj") {
                ctx.wordsOfChrist = true
            }
            if classes.contains("divine-name") || classes.contains("nd") {
                ctx.smallCaps = true
            }
            if tag == "a" {
                let href = (try? element.attr("href")) ?? ""
                if classes.contains("cf") {
                    // Bare reference list; ESV appends a trailing slash —
                    // strip it exactly as the web does (PassageArticle.tsx).
                    let q = href.replacingOccurrences(
                        of: "/+$", with: "", options: .regularExpression
                    ).trimmingCharacters(in: .whitespaces)
                    if !q.isEmpty { ctx.crossRef = q }
                } else if classes.contains("fn") {
                    if href.hasPrefix("#") { ctx.footnoteID = String(href.dropFirst()) }
                } else if let url = URL(string: href),
                          url.scheme == "http" || url.scheme == "https" {
                    ctx.link = url
                }
            }

            for child in element.getChildNodes() {
                appendInline(node: child, context: ctx, into: &runs)
            }
        }

        /// Append text under a context, merging into the previous run when
        /// the attributes are identical.
        func append(text: String, context: InlineContext, into runs: inout [InlineRun]) {
            guard !text.isEmpty else { return }
            var run = InlineRun(text: text)
            run.isVerseNumber = context.verseNumber
            run.isWordsOfChrist = context.wordsOfChrist
            run.isItalic = context.italic
            run.isBold = context.bold
            run.isSmallCaps = context.smallCaps
            run.isSuperscript = context.superscript
            run.crossRef = context.crossRef
            run.footnoteID = context.footnoteID
            run.link = context.link
            if var last = runs.last, sameAttributes(last, run) {
                last.text += run.text
                runs[runs.count - 1] = last
            } else {
                runs.append(run)
            }
        }

        func sameAttributes(_ a: InlineRun, _ b: InlineRun) -> Bool {
            var a = a, b = b
            a.text = ""
            b.text = ""
            return a == b
        }

        // MARK: Footnotes tail

        /// ESV footnotes section: `div.footnotes` → `h3` ("Footnotes",
        /// skipped — section chrome, not content) and a `p` holding, per
        /// note: `span.footnote > a[id]` `[1]`, `span.footnote-ref` `23:2`,
        /// a `<note>` body, separated by `<br/>`.
        func parseFootnotes(in div: Element) {
            for paragraph in div.children() where paragraph.tagName() == "p" {
                var current: Footnote?
                for node in paragraph.getChildNodes() {
                    if let element = node as? Element {
                        recordClasses(of: element)
                        let classes = (try? element.classNames()) ?? []
                        if element.tagName() == "br" {
                            flushFootnote(&current)
                            continue
                        }
                        if classes.contains("footnote"),
                           let anchor = element.children().first(where: { $0.tagName() == "a" }) {
                            flushFootnote(&current)
                            let label = ((try? anchor.text()) ?? "")
                                .trimmingCharacters(in: CharacterSet(charactersIn: "[]"))
                            current = Footnote(
                                id: anchor.id(),
                                label: label,
                                reference: "",
                                runs: []
                            )
                            continue
                        }
                        if classes.contains("footnote-ref") {
                            current?.reference = ((try? element.text()) ?? "")
                                .trimmingCharacters(in: .whitespaces)
                            continue
                        }
                    }
                    if current != nil {
                        appendInline(node: node, context: InlineContext(), into: &current!.runs)
                    }
                }
                flushFootnote(&current)
            }
        }

        func flushFootnote(_ current: inout Footnote?) {
            if var note = current {
                note.runs = trimmedRuns(note.runs)
                passage.footnotes.append(note)
            }
            current = nil
        }

        // MARK: Text helpers

        /// HTML whitespace semantics: runs of ASCII whitespace collapse to a
        /// single space. U+00A0 (`&nbsp;`) is preserved — ESV uses it for
        /// poetry indentation and verse-number spacing.
        func collapseWhitespace(_ text: String) -> String {
            text.replacingOccurrences(
                of: "[ \\t\\n\\r]+", with: " ", options: .regularExpression
            )
        }

        /// Drop the literal `&nbsp;` indent prefix from a poetry line's
        /// first text run — indentation is carried structurally by
        /// `poetryLine(indent:)` instead.
        func stripLeadingIndent(_ runs: [InlineRun]) -> [InlineRun] {
            var runs = runs
            for i in runs.indices where !runs[i].isVerseNumber {
                runs[i].text = String(
                    runs[i].text.drop(while: { $0 == "\u{00A0}" || $0 == " " })
                )
                break
            }
            return runs.filter { !$0.text.isEmpty }
        }

        /// Trim block-edge whitespace (plain spaces and newlines, not nbsp)
        /// and drop runs that end up empty.
        func trimmedRuns(_ runs: [InlineRun]) -> [InlineRun] {
            var runs = runs.filter { !$0.text.isEmpty }
            if var first = runs.first {
                first.text = String(first.text.drop(while: { $0 == " " || $0 == "\n" }))
                runs[0] = first
            }
            if var last = runs.last {
                while last.text.hasSuffix(" ") || last.text.hasSuffix("\n") {
                    last.text = String(last.text.dropLast())
                }
                runs[runs.count - 1] = last
            }
            return runs.filter { !$0.text.isEmpty }
        }
    }
}
