# Study Plan

**Status:** Shipped
**Created:** 2026-08-16
**Last updated:** 2026-08-16
**Owner:** unassigned

## Why

The two shipped plans are both once-through-the-Bible schedules. The user
wants a heavier study cadence: the whole Old Testament across the year, the
whole New Testament **once a quarter** (four passes), and a Psalm every day —
starting Jan 1. The plan registry from [multi-plan](./multi-plan.md) already
supports a third plan, so this is a new embedded schedule plus a parser, not
new architecture.

Unlike the two existing plans, this schedule is *derived*, not transcribed
from a published source. It is generated from the canon table so the coverage
math is checkable rather than trusted.

## Goals

- [x] A third plan, id `study`, name "Study Plan", registered alongside
      `bible-year` and `hope` and selectable from the gear panel.
- [x] OT track: Genesis → Malachi (minus Psalms) exactly once, in canonical
      order, over the year — 779 chapters, 2-3 a day.
- [x] NT track: Matthew → Revelation restarted at the top of every calendar
      quarter — 4 × 260 chapters, ~3 a day.
- [x] Psalm track: Psalms 1-150 on a straight 150-day loop from Jan 1, so the
      whole Psalter lands inside the year (and about 2.4 passes total).
- [x] When the day's Psalm is 119, Psalm 119 is that day's OT reading too: the
      OT track rests and the day renders one `Psalms 119` pill, not two.
- [x] The schedule is generated (`internal/dailyreader/gen`) and its coverage
      is asserted by tests, not eyeballed.
- [x] Psalm-track pills are tagged "Psalm" rather than a second "OT".

## Non-goals

- **Making it the default plan.** `DEFAULT_PLAN_IDS` stays `["bible-year"]`;
  this plan is opt-in from the gear panel like Hope.
- **Splitting Psalm 119 into stanzas.** The daily panel renders chapter
  blocks (see [multi-plan](./multi-plan.md) on dropped verse ranges), so
  Psalm 119 is scheduled whole — which is exactly why it displaces the OT
  reading that day.
- **A weekday-aware or catch-up-heavy schedule.** Unlike Hope, this plan
  reads all 365 days. The only rest is Feb 29.
- **Per-plan pacing preferences** (start date, chapters/day, skip weekends).
  The schedule is fixed and year-agnostic like the others.

## User-facing behavior

**Picker.** "Study Plan" appears as a third checkbox in the gear panel's
"Reading plan" group. Selection persists in `localStorage` exactly as the
other plans do.

**Daily panel.** An ordinary day yields three pills, in table order: the OT
reading, the NT reading, and the Psalm. A day whose OT chapters cross a book
boundary yields a fourth pill (`Zechariah 14`, `Malachi 1` on 12/30) — the
cell holds two refs joined by `;`, one pill each.

**Pill tags.** With only this plan selected, pills are tagged by track: `OT`,
`NT`, `Psalm`. With more than one plan selected the plan name replaces the
tag, per multi-plan.

**Psalm 119 days (Apr 29, Sep 26).** One `Psalms 119` pill tagged `OT`, plus
the day's NT pill. No separate Psalm pill — the duplicate is collapsed.

**Feb 29.** An info card reading "Catch-up day — no new reading". The plan is
keyed MM/DD and built on a 365-day calendar; the leap day is a rest day rather
than a schedule shift.

## Implementation outline

**Generator — `internal/dailyreader/gen/main.go`** (`go generate ./internal/dailyreader/...`)

Walks a non-leap reference year and writes `study-plan.md`:

- Psalm for day *i* is `i % 150 + 1`.
- OT days are the days whose Psalm is not 119; the 779 non-Psalm OT chapters
  are divided across them by `total*j/n` boundaries, so each day gets the
  floor or ceil of the average and the longer days are spread evenly.
- NT chapters are divided the same way, independently within each calendar
  quarter.
- Consecutive chapters of one book collapse to `Book 1-3`; a run crossing a
  book boundary emits both refs joined by `; `.
- A `02/29` catch-up row is emitted after `02/28`.

**Plan file — `internal/dailyreader/study-plan.md`**

Generated; 366 rows of `| MM/DD | OT Reading | NT Reading | Psalm |`, with a
header comment marking it as generated. Embedded via `//go:embed`.

**Parser — `internal/dailyreader/parser_study.go`**

`parseStudy` reads the four-column table, keying by the `MM/DD` date cell.
Cells go through the shared `parsePassageCell` (extracted from the Hope
parser into `passage_cell.go` in the same change, renamed from
`parseHopeCell`/`parseHopePassage` — no behavior change). Psalm-column
passages get `Testament = "Psalm"`. `dedupePassages` collapses repeats of the
same book + chapter range within a day, keeping the first occurrence — that's
what turns the doubled Psalm 119 into a single OT-tagged pill. A cell that
doesn't parse as passages yields a `Message` (the Feb 29 row), matching Hope's
special-day handling.

**Registry / SPA**

- `internal/dailyreader/dailyreader.go` — third entry in `plans`;
  `Passage.Testament` re-documented as a track label (`"OT"`, `"NT"`,
  `"Psalm"`).
- `web/src/daily/plans.ts` — `PlanID` gains `"study"`, catalog gains
  `{ id: "study", name: "Study Plan" }`. The settings checkbox group and the
  daily pill row are driven off that catalog, so no component changed.
- `web/src/api.ts`, `web/src/daily/useDailyTab.ts` — the track-label union
  widens to `"OT" | "NT" | "Psalm" | ""`.

## Open questions

- **Daily volume.** About 6 chapters a day (2-3 OT + ~3 NT + 1 Psalm), roughly
  25-30 minutes. That is the requested shape, but it is roughly double the
  Bible-in-One-Year load; if it proves unsustainable, the natural dial is the
  NT cadence (twice a year instead of quarterly), which is a generator
  constant, not a structural change.
- **Psalm 119 tag.** The collapsed pill keeps the OT-column tag, so on those
  two days the Psalm track shows no pill of its own. The alternative — tagging
  it "Psalm" — loses the signal that the OT track is resting. Revisit if it
  reads as a missing psalm.
- **Quarter boundaries vs. calendar quarters.** The NT restarts on Jan 1, Apr
  1, Jul 1, Oct 1, so the four passes are 90/91/92/92 days long and the daily
  NT load varies slightly by quarter. Even quarters of 91.25 days would drift
  off month boundaries and read worse.

## Decisions

- **2026-08-16** — Psalms are covered by the Psalm track, not the OT track.
  "Read the entire Old Testament" is satisfied across the two tracks together;
  scheduling Psalms twice would have added ~150 duplicate chapters to the OT
  pace for no reading benefit.
- **2026-08-16** — On Psalm 119 days the OT track *rests* rather than shifting
  the whole schedule. Chapters are allocated to the 363 non-119 days up front,
  so the plan still finishes Malachi on 12/31.
- **2026-08-16** — The schedule is generated and committed, not computed at
  request time. Keeps `dailyreader` a parser package (same shape as the other
  two plans), keeps the schedule diffable in review, and lets tests assert
  coverage against the canon table.
- **2026-08-16** — Feb 29 is a catch-up day. The MM/DD keying that makes the
  plan year-agnostic can't absorb a 366th reading day without desynchronising
  the tracks.
- **2026-08-16** — `Passage.Testament` carries a third value `"Psalm"` rather
  than a new field. It is a display tag end to end (server JSON → pill label);
  a parallel `Track` field would have duplicated it.
- **2026-08-16** — The Hope parser's cell helpers were renamed and moved to
  `passage_cell.go` rather than copied. Both plans need the same
  segment-splitting and canon-normalising rules.

## Related

- [`multi-plan.md`](./multi-plan.md) — the plan registry, `?plans=` query, and
  pill-tagging rules this plan plugs into unchanged.
- [`auto-load-daily-reading.md`](./auto-load-daily-reading.md) — the daily-tab
  load flow.
- [`PROJECT_CONSTITUTION.md`](../PROJECT_CONSTITUTION.md) — §1 (study-first),
  §3 (plans schedule reading; scripture text is never mutated).

## Verification

**Go tests** (`internal/dailyreader/parser_study_test.go`)

- Every calendar date Jan 1 - Dec 31 plus 02/29 has a row; no row is empty.
- 01/01 is Genesis 1-2 / Matthew 1-2 / Psalm 1.
- 12/30 crosses a book boundary into two OT passages.
- 02/29 is a message-only row with markdown emphasis stripped.
- The NT restarts at Matthew 1 on 01/01, 04/01, 07/01, 10/01 and lands on
  Revelation 22 on 03/31, 06/30, 09/30, 12/31.
- OT coverage: the OT track, minus its two Psalm 119 days, equals the 779
  non-Psalm OT chapters in canonical order, element by element.
- NT coverage: 1040 chapters, four identical Matthew → Revelation passes.
- Psalm track: exactly one Psalm per reading day; all 150 scheduled.
- Psalm 119 days: exactly one `Psalms 119` passage, tagged OT, no other OT or
  Psalm passage, NT still present.
- `dedupePassages` keeps the first occurrence.
- `TestRoundTripEveryRowValidates` now covers `study`, so every ref in the
  generated file round-trips through `canon.ValidateQuery`.

**Server test** — `TestDailyHandlerStudyPlan`: `?plans=study&date=2026-01-01`
returns three passages with the OT / NT / Psalm track labels.

**Manual smoke**
1. `cd web && npm install && npm run build && cd .. && ESV_API_KEY=… YOUVERSION_APP_KEY=… go run .`
2. Daily tab → gear → check "Study Plan", uncheck the others.
3. Expect three pills (OT / NT / Psalm) and confirm each loads.
4. Date-pick `2026-04-29` → a single `Psalms 119` pill tagged OT plus the NT pill.
5. Date-pick `2026-12-30` → four pills (`Zechariah 14`, `Malachi 1`, NT, Psalm).
6. Date-pick a leap-year Feb 29 (e.g. `2028-02-29`) → catch-up info card.
