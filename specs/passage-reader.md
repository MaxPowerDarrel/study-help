# Passage Reader

**Status:** Draft <!-- Draft | In Progress | Shipped | Deprecated -->
**Created:** 2026-05-02
**Last updated:** 2026-05-03
**Owner:** Darrel

## Why

The reading surface is the product. This feature delivers the core
behavior the project exists for: reading a chapter or contiguous passage
range of scripture in a focused, distraction-free way. Supports
`PROJECT_CONSTITUTION.md` §1 (purpose: a Bible reader optimized for
focused study a chapter or section at a time), §2 (in-scope: reading a
chapter or a contiguous passage range; ESV-only at v1), §3 (Study-first
UX, Respect the text, Responsiveness over richness), and §4
(server-proxied ESV — the API key never reaches the client).

## Goals

User-visible outcomes that define success. Bullets, not paragraphs.

- [ ] User can open any ESV chapter by book + chapter and read it on a focused reading surface.
- [ ] User can render a contiguous passage range within a chapter (e.g. John 3:1–21).
- [ ] Chapter-switch and range render feel instant — p50 < 150ms, p95 < 500ms warm (§3 Responsiveness over richness).

## Non-goals

Things explicitly excluded from this feature, with one-line rationale
each. Forces scope honesty — what we're *not* doing is as important as
what we are.

- Search across scripture — *the reader is for reading; search would shift the product away from focused study (§3, §6)*
- Audio playback — *separate UX surface; doesn't serve careful slow reading at v1 even though ESV offers an audio API*

## User-facing behavior

Concrete description of what the user sees and does. Not implementation.
A senior PM should be able to read this and know what shipping looks
like.

A persistent book/chapter picker pane lets the user choose where to
read; the rest of the viewport is the reading surface and renders the
passage text scrollably. Within a chapter, a range selector lets the
user narrow to a contiguous verse span. The reading surface is the
focal element — picker chrome stays out of the way (§3 Study-first
UX). Next/Previous navigation continues across book boundaries (e.g.
Malachi 4 → Matthew 1) and the relevant control is hidden at the canon
edges (no Previous on Genesis 1; no Next on Revelation 22). The user
can toggle which formatting elements appear: section headings,
footnotes, verse numbers, and the passage reference header. Toggle
state persists in browser localStorage (per device). While a passage
loads, the reading surface shows a centered spinner. On ESV
rate-limit (429) the toast says "Service is busy, try again in a
moment"; on other proxy failures or timeouts the toast says
"Something went wrong, try again".

## Implementation outline

High-level shape only:

- New package `internal/esv/` — HTTP client wrapping `api.esv.org`. Server-side only (§4: ESV API key never reaches the client).
- New handler in `internal/server` for `GET /api/passage?q=<reference>` — free-form ESV reference syntax in `q`; server validates `q` against an allow-list (book names + `chapter:verse` + ranges) and rejects malformed input with 400 before consuming an ESV API call; valid requests proxy through `internal/esv` and return the ESV HTML payload. Formatting toggles surface as query params (`include_headings`, `include_footnotes`, `include_verse_numbers`, `include_passage_references`) and are forwarded to ESV. Each successful ESV call increments an in-process counter and emits a structured log line; the counter is exposed on a `/metrics` HTTP endpoint (Prometheus-style exposition) bound to a localhost-only port (e.g. `127.0.0.1:9090`), separate from the public `/api/passage` listener, so a dashboard can scrape it without exposing internal counters publicly.
- Canon (66 books + chapter counts) is hard-coded in the React bundle; no `/api/canon` endpoint at v1.
- No new DB tables for the reader itself; highlights and notes will get their own specs and migrations.
- Client surface: React SPA in `web/` built with Vite (static build output served by the Go server). Renders the picker + reading surface and calls `/api/passage`. Must load as a static bundle inside a WebView (§4 Frontend is decoupled).
- Client persistence: localStorage access for formatting toggles sits behind a thin `ToggleStore` interface at `web/src/platform/ToggleStore.ts` (get/set/remove) so a future native-shell WebView can substitute its own implementation without touching the reading-surface component (§4 platform abstraction).

Pointers, not full designs. The spec is durable; the design notes can
stay in the PR.

## Open questions

Questions that need to be resolved. When answered, move the resolution
under **Decisions** and leave a pointer here (don't delete the question
— its existence is part of the history).

- [x] What are the explicit non-goals for this feature (e.g. search, cross-references, footnote toggling, audio) and the one-line rationale for each? — *resolved 2026-05-02 (see Decisions and the Non-goals section)*
- [x] How is a passage range specified by the client and validated server-side (book/chapter/verse format, max range size)? — *resolved 2026-05-02 (see Decisions)*
- [x] How is ESV API response data cached server-side to honor §3 "Responsiveness over richness" (page-turn must feel instant) without violating ESV API terms? — *resolved 2026-05-02 (see Decisions)*
- [x] How does the reader behave when the user is unauthenticated — is reading gated behind sign-in, or available anonymously with highlights/notes locked? — *resolved 2026-05-02 (see Decisions)*
- [x] What ESV API formatting options will the server request (e.g. `include-headings`, `include-footnotes`, `include-verse-numbers`) and how are they exposed (or not) to the client? — *resolved 2026-05-02 (see Decisions)*
- [x] How are chapter boundaries handled — does "next/previous chapter" cross book boundaries, and what happens at Genesis 1 / Revelation 22? — *partially resolved 2026-05-02 (see Decisions); canon-edge behavior still open*
- [x] What happens at the canon edges — before Genesis 1 (Previous) and after Revelation 22 (Next): no-op, wrap, or hide the control? — *resolved 2026-05-02 (see Decisions)*
- [x] How are ESV API terms (attribution, copyright notice, request volume caps) surfaced in the rendered HTML and in the server proxy? — *resolved 2026-05-02 (see Decisions)*
- [x] How is the book/chapter picker populated — hard-coded canon list, fetched from a server endpoint, or derived from ESV responses? — *resolved 2026-05-02 (see Decisions)*
- [x] Does the formatting-toggle state persist across sessions/devices, or reset per visit? (Persistence implies user-data storage, which intersects §4 server-authoritative.) — *resolved 2026-05-02 (see Decisions)*
- [x] What is the rate-limiting / abuse-prevention posture on `/api/passage` given reading is public (no auth gate)? — *resolved 2026-05-02 (see Decisions)*
- [x] Does the `q` passthrough sanitize or constrain input before forwarding to ESV (e.g. length cap, character allow-list), or is ESV's grammar the only validator? — *resolved 2026-05-02 (see Decisions)*
- [x] What are the verification criteria — which tests, manual flows, or metrics confirm shipping (e.g. "fetching John 3 returns ESV-formatted HTML/JSON in <300ms warm")? — *resolved 2026-05-02 (see Decisions and the Verification section)*
- [x] What loading and error states does the client render when the ESV proxy fails or rate-limits? — *resolved 2026-05-02 (see Decisions)*
- [x] Does the server return ESV's HTML payload as-is, or normalize it into a structured JSON shape the client renders? (Trade-off: ESV HTML is fast to ship but couples client rendering to ESV's markup.) — *resolved 2026-05-02 (see Decisions)*
- [x] How does the reader integrate with future highlights/notes — what passage-identifier scheme do ranges use so annotations can attach durably? — *resolved 2026-05-02 (see Decisions)*
- [x] Should the latency budget (p50 < 150ms, p95 < 500ms warm) be measured client-perceived or server-side, and where will the synthetic latency check run from? — *resolved 2026-05-03 (see Decisions)*
- [x] What defines "warm" for the latency budget — first request after deploy excluded, or a specific warm-up sequence? — *resolved 2026-05-03 (see Decisions)*
- [x] Should the server-side `q` allow-list also enforce a maximum range size (e.g. cap on number of verses) to prevent expensive ESV calls? — *resolved 2026-05-03 (see Decisions)*
- [x] How does the WebView abstraction (§4 "Platform features behind an abstraction") apply to the localStorage-backed formatting toggles — is a thin interface required at v1, or deferred until the native shell exists? — *resolved 2026-05-03 (see Decisions)*
- [x] What ESV API request volume cap applies, and does the no-rate-limit posture risk exhausting it before the "revisit if usage spikes" trigger fires? — *resolved 2026-05-03 (see Decisions)*
- [x] Should the generic error toast distinguish ESV rate-limit (429) from upstream timeout/5xx, or is one message acceptable at v1? — *resolved 2026-05-03 (see Decisions)*
- [x] Is the React SPA framework choice (plain React, Next.js static export, Vite, etc.) decided, given §4 requires a static bundle that loads in a WebView? — *resolved 2026-05-03 (see Decisions and STACK.md)*
- [x] Verification: does the suite confirm canon-edge UX (no Previous on Genesis 1, no Next on Revelation 22) and book-boundary navigation (e.g. Malachi 4 → Matthew 1)? Both are user-facing decisions made this round and currently unchecked. — *resolved 2026-05-03 (adopted as Verification line item)*
- [x] Verification: does the suite confirm `/api/passage` rejects malformed `q` with 400 *before* consuming an ESV API call? This is the central abuse-prevention claim and currently unchecked. — *resolved 2026-05-03 (adopted as Verification line item)*
- [x] Verification: does the suite confirm formatting-toggle query params (`include_headings`, `include_footnotes`, `include_verse_numbers`, `include_passage_references`) round-trip end-to-end, and that ESV attribution markup is preserved verbatim in the proxied HTML? — *resolved 2026-05-03 (adopted as Verification line item)*
- [x] Should the synthetic latency check's 3 priming requests target the *same* passage or 3 distinct passages? ESV-side caching could make the two materially different. — *resolved 2026-05-03 (see Decisions)*
- [x] Where does the `ToggleStore` interface live in `web/` (e.g. `web/src/platform/`) so the future native-shell substitution point is discoverable, or is the location left to the implementing PR? — *resolved 2026-05-03 (see Decisions)*
- [x] Is the per-request ESV counter process-local only (resets on restart), and is that acceptable for the "dashboard to read" goal — or does it need an export path (`/metrics`, log aggregation) to be useful? — *resolved 2026-05-03 (see Decisions and STACK.md)*
- [x] Does the user-facing error-toast copy ("Service is busy, try again in a moment" / "Something went wrong, try again") need i18n hooks at v1, or is English-only locked in? — *resolved 2026-05-03 (see Decisions)*
- [x] Verification covers attribution markup preservation under toggle combinations but not under 429/5xx error paths — should the error-toast Verification line also assert attribution markup is *not* required when the body never reaches the user? — *resolved 2026-05-03 (see Decisions; no new Verification line)*
- [x] Should Verification include a check that `/metrics` exposes the ESV-call counter in the agreed Prometheus exposition format (metric name, type, sample line shape) so a dashboard scrape contract is pinned before implementation drift? — *resolved 2026-05-03 (see Decisions; deliberately not pinned at spec level)*
- [x] Should the latency-check Decision pin the *measurement* reference too, or is "a fixed reference" left to the implementing PR? The priming list is named (Genesis 1, Psalm 23, John 3); the measurement reference is not — asymmetry worth resolving. — *resolved 2026-05-03 (see Decisions; deliberately not pinned at spec level)*
- [x] Should the spec name the `ToggleStore` key namespace (e.g. `reader.toggles.*`) so a future native shell substitution has a contract, or is that left to the implementing PR? — *resolved 2026-05-03 (see Decisions; deliberately not pinned at spec level)*
- [x] Is the `/metrics` surface explicitly counter-only at v1, or is the implementing PR free to add request-duration histograms? — *resolved 2026-05-03 (see Decisions)*
- [x] The toggle round-trip Verification claims attribution holds "across all 16 toggle combinations (2⁴)" but the test as described is per-toggle (4 pairs = 8 requests). Is the 16-combination claim asserted by a separate combinatorial sub-test, or is it an inference the per-toggle assertions don't actually establish? — *resolved 2026-05-03 (see Decisions; over-promise dropped from Verification line)*

## Decisions

Append-only log. Most recent at the bottom. Never rewrite past entries;
if a decision is reversed, add a new entry that supersedes it.

- 2026-05-02: Reading is public; only annotations (highlights, notes) require auth. Reason: lowers friction and keeps the §3 study-first reading experience reachable without a sign-up wall. Resolves Open question on auth gating.
- 2026-05-02: Server returns ESV's HTML payload as-is (ESV `passages/html` endpoint). Reason: fastest path to a usable reader; revisit if highlight anchoring or styling forces a structured shape. Resolves Open question on payload shape.
- 2026-05-02: Passage range is specified as a free-form ESV reference in the `q` query param (e.g. `?q=John+3:1-21`); server forwards to ESV and lets ESV's grammar enforce validity. Resolves Open question on range specification.
- 2026-05-02: No server-side cache for v1. Reason: ship the simplest path; revisit when ESV rate limits or §3 responsiveness goals demand it. Resolves Open question on caching.
- 2026-05-02: Non-goals declared for v1: search and audio playback. Reason: search would shift the product away from focused reading; audio is a separate UX surface that doesn't serve careful slow reading at v1. Cross-reference linking and footnote toggling were considered and deliberately *not* added as non-goals (left open for future scoping).
- 2026-05-02: Server requests ESV with `include-headings`, `include-footnotes`, `include-verse-numbers`, and `include-passage-references` all enabled. Each is exposed to the client as a user-controllable toggle on the reading surface, surfaced as query params on `/api/passage`. Resolves Open question on formatting options.
- 2026-05-02: Chapter Next/Previous continues across book boundaries (e.g. Malachi 4 → Matthew 1). Canon-edge behavior (before Genesis 1, after Revelation 22) is deferred — see new Open question. Partially resolves Open question on chapter boundaries.
- 2026-05-02: Anchoring scheme for highlights/notes is deferred to the future highlights spec. The reader commits to ESV HTML as the payload shape; the anchoring decision (verse IDs vs. character offsets vs. something else) belongs to the highlights spec, not this one. Resolves Open question on highlights integration.
- 2026-05-02: Client is a React SPA in `web/` with static build output served by the Go server. Reason: familiar tooling for the team and a clean static-bundle target that loads inside a WebView (§4 Frontend is decoupled).
- 2026-05-02: "Instant" latency budget is p50 < 150ms and p95 < 500ms warm for chapter-switch and range render. "Warm" excludes cold-start; ESV upstream latency counts.
- 2026-05-02: Verification at ship is a smoke test (`/api/passage?q=John+3` → 200 + HTML), a range-render test (`q=John+3:1-21`), and a synthetic latency check against the chosen budget. Manual click-through was not adopted; the suite is automation-only.
- 2026-05-02: Latency budget (p50 < 150ms, p95 < 500ms warm) stays as-is even though no cache and ESV upstream time both count. Reason: ESV's p95 is generally inside this window; if production tells us otherwise, that's the trigger to revisit the no-cache decision.
- 2026-05-02: ESV attribution and copyright are surfaced by passing ESV's HTML payload through verbatim — ESV's markup already includes the required attribution. No additional footer or wrapper UI in v1. Resolves Open question on ESV terms surfacing.
- 2026-05-02: `/api/passage` ships at v1 with no auth and no rate limit. Reason: simplest path; the ESV API key is the asset and it stays server-side. Revisit if usage spikes or abuse appears. Resolves Open question on rate-limiting/abuse posture.
- 2026-05-02: At canon edges, the Next/Previous control is hidden (no Previous on Genesis 1; no Next on Revelation 22). Reason: cleanest UX — the boundary is conveyed by absence rather than a disabled affordance. Resolves Open question on canon edges.
- 2026-05-02: Book/chapter picker is populated from a hard-coded canon list (66 books with chapter counts) shipped in the React bundle. No `/api/canon` endpoint at v1. Reason: the ESV canon is fixed; zero round-trips beat any flexibility we'd buy from a server endpoint. Resolves Open question on picker source.
- 2026-05-02: Formatting-toggle state persists client-side only (browser localStorage, per device). No server-side persistence at v1; revisit cross-device sync once sign-in lands and we have a place to store user preferences. Resolves Open question on toggle persistence.
- 2026-05-02: `/api/passage` validates `q` against a server-side allow-list (book names + `chapter:verse` + ranges) before forwarding to ESV; malformed input is rejected with 400 without consuming an ESV API call. Reason: cheapest defense against accidental abuse and reduces ESV-key exposure to garbage traffic. Resolves Open question on `q` sanitization.
- 2026-05-02: Loading state is a centered spinner; error state on ESV proxy failure/timeout is a generic error toast. Reason: standard SPA pattern; matches the simplicity bar at v1. Resolves Open question on loading/error states.
- 2026-05-03: Latency budget (p50 < 150ms, p95 < 500ms warm) is measured server-side, handler-only. Synthetic latency check runs from the same region as the deployment. Reason: cleanest attribution of regressions to our code or the ESV upstream; client/network variance is out of scope at v1. Resolves Open question on latency measurement vantage.
- 2026-05-03: localStorage-backed formatting toggles sit behind a thin client-side `ToggleStore` interface (get/set/remove) at v1, so a future native-shell WebView can substitute its own backing without touching feature code. Reason: §4 requires platform features behind an abstraction; the cost of one small interface up front is lower than retrofitting later. Resolves Open question on the WebView abstraction for toggle storage.
- 2026-05-03: The server-side `q` allow-list does not cap range size at v1 — beyond shape validation (book + chapter:verse + ranges), ESV's grammar is the only validator. Reason: matches the existing no-cache / no-rate-limit "simplest path" posture; revisit if usage spikes or abuse appears (same trigger as the rate-limit decision). Resolves Open question on `q` allow-list range size.
- 2026-05-03: "Warm" is operationalized in the synthetic latency check by firing 3 priming requests per measurement run and discarding them; subsequent requests count toward p50/p95. Reason: deterministic, cheapest way to give the budget a measurable protocol. Resolves Open question on warm-up definition.
- 2026-05-03: ESV API quota observability at v1 = per-request counter + structured log line on every ESV call; no automated alert. Reason: cheapest signal that gives us a dashboard to read; matches the broader v1 simplicity posture and the "revisit if usage spikes" trigger. Resolves Open question on ESV volume cap observability.
- 2026-05-03: Error toast distinguishes ESV rate-limit (429) from other upstream failures: 429 → "Service is busy, try again in a moment"; 5xx/timeout → generic "Something went wrong, try again". Reason: the no-cache + no-rate-limit posture makes 429 foreseeable, and a retry-friendly message reduces user-side retry storms. Resolves Open question on error toast granularity.
- 2026-05-03: React SPA tooling is Vite. Reason: fastest path to a clean static build that loads inside a WebView, minimal config, no server-only runtime APIs (§4). Recorded in STACK.md alongside the React choice. Resolves Open question on SPA framework choice.
- 2026-05-03: Three verification gaps surfaced this round are adopted as Verification line items rather than left as open questions: canon-edge UX + book-boundary navigation; allow-list 400 rejection without an ESV call; formatting-toggle round-trip + ESV attribution preserved. Resolves the three verification-gap Open questions.
- 2026-05-03: Synthetic latency check primes with 3 distinct passages (e.g. Genesis 1, Psalm 23, John 3) per measurement run, then measures against a fixed reference. Reason: per-reference ESV-side caching could otherwise under-report p95 if priming and measurement share the same passage. Resolves Open question on priming scope.
- 2026-05-03: `ToggleStore` lives at `web/src/platform/ToggleStore.ts`. Reason: a dedicated `platform/` directory makes the WebView substitution boundary discoverable and signals "thin platform abstraction" to readers. Resolves Open question on `ToggleStore` location.
- 2026-05-03: ESV-call counter is process-local (resets on restart) and exposed on a `/metrics` HTTP endpoint in Prometheus-style exposition format alongside the structured log line per call. Reason: log search alone makes ad-hoc rollups slow; a metrics endpoint keeps observability real-time without persistent state. Library choice deferred to the implementing PR; recorded in STACK.md. Resolves Open question on counter scope.
- 2026-05-03: Error-toast copy is English-only at v1; no i18n hook layer. Reason: matches the v1 simplicity posture; revisit when localization becomes a real requirement. Resolves Open question on i18n hooks.
- 2026-05-03: ESV attribution markup is not asserted in error paths because the response body is not rendered to the user when the proxy returns 429/5xx (the toast replaces the reading surface). The existing toggle-round-trip Verification line covers attribution under all 16 success-path toggle combinations. Resolves Open question on attribution under error paths.
- 2026-05-03: `/metrics` binds to a localhost-only port (e.g. `127.0.0.1:9090`), separate from the public `/api/passage` listener. Reason: prevents accidental public exposure of internal counters; matches Prometheus convention where exporters bind locally and a sidecar/in-host scraper consumes them. Resolves Open question on `/metrics` exposure posture.
- 2026-05-03: Owner assigned to Darrel.
- 2026-05-03: `/metrics` (Prometheus text exposition) sits outside §4's "Backend is a JSON API" rule because §4 is about the *application* API surface a native client consumes, not internal observability. Constitution PR #2 (`chore/constitution-metrics-carveout`) writes that scoping into §4 explicitly. Resolves the §4 alignment question raised in round-4 review.
- 2026-05-03: Three round-4 follow-ups are deliberately **not** pinned in the spec — implementing PR decides each: (a) `/metrics` scrape contract (metric name, type, exposition line shape); (b) latency-check measurement reference; (c) `ToggleStore` key namespace. Reason: each is an implementation detail that doesn't affect user-visible behavior or constitutional alignment; pinning would over-constrain the PR without buying real durability.
- 2026-05-03: `/metrics` is counter-only at v1 (the ESV-call counter); request-duration histograms deferred. Reason: the synthetic latency check covers latency observability already, and histograms add cardinality + library complexity without solving a known problem at v1. Revisit if production needs prove otherwise.
- 2026-05-03: Toggle round-trip Verification dropped the "all 16 toggle combinations (2⁴)" claim; the line now asserts only what the per-toggle pair test actually establishes (attribution present in both responses for each pair). Reason: the prior wording over-promised relative to the described test; combinatorial coverage can be added later if a real bug surfaces it.

## Verification

How we'll know it's working: tests, manual flows, metrics, screenshots.

- [ ] Smoke test: `GET /api/passage?q=John+3` returns 200 with ESV HTML.
- [ ] Range render test: `GET /api/passage?q=John+3:1-21` returns the right verses.
- [ ] Latency check: synthetic measurement (3 priming requests against 3 distinct passages — e.g. Genesis 1, Psalm 23, John 3 — discarded per run) confirms server-side p50 < 150ms and p95 < 500ms warm against the production ESV upstream.
- [ ] Canon-edge & book-boundary navigation: clicking Previous on Genesis 1 has no effect (control hidden); clicking Next on Revelation 22 has no effect (control hidden); Next from Malachi 4 advances to Matthew 1.
- [ ] Allow-list rejection: malformed `q` (e.g. `q=!!!`, `q=Booga+99`, empty `q=`) returns 400 from `/api/passage` and the ESV upstream request count for that test is 0.
- [ ] Toggle round-trip — asserted per toggle individually: for each of `include_headings`, `include_footnotes`, `include_verse_numbers`, `include_passage_references`, fetch the same reference twice (`=true` / `=false`) and assert the `false` response omits the corresponding markup category (section-heading element / footnote marker / verse-number marker / passage-reference header) while the `true` response includes it. Selectors pinned in the test fixture by the implementing PR. ESV attribution markup remains present in both responses for each pair.
- [ ] ESV-call counter: a successful request to `/api/passage` with valid `q` increments the in-process counter by exactly 1; a 400-rejected request leaves the counter unchanged.
- [ ] Error-toast rendering: a simulated upstream 429 surfaces the exact string "Service is busy, try again in a moment"; a simulated upstream 5xx/timeout surfaces "Something went wrong, try again".

## Related

- Other specs this depends on or extends: `[name](./name.md)`
- Constitution sections: `PROJECT_CONSTITUTION.md §1`, `§2`, `§3`, `§4`
- External references: ESV API docs (https://api.esv.org/docs/)