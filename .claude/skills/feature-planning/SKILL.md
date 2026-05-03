---
name: feature-planning
description: Use this skill whenever the user wants to plan a new feature, draft or update a feature spec for study-help, check the status of in-flight work, review a spec for gaps, or runs /plan-feature or /review-spec. Each feature has a living markdown file in specs/ that captures the why, scope, decisions, and current status. New and substantively-edited specs are reviewed by the spec-reviewer sub-agent for completeness, accuracy, and ambiguity. Trigger phrases include "plan a feature", "draft a spec", "spec out X", "update the X spec", "what's the status of X", "what specs do we have", "review the X spec", "check the X spec for gaps", "/plan-feature", and "/review-spec".
---

# feature-planning skill

This skill maintains the `specs/` directory — one markdown file per feature,
each a **living document** that evolves with the feature it describes. Specs
sit between [`PROJECT_CONSTITUTION.md`](../../../PROJECT_CONSTITUTION.md)
(durable principles) and the code (implementation): they capture *what a
feature is, why it exists, what's in/out of scope, and what's been decided*.

The skill does five things, depending on the user's intent:

1. **Create** a new spec when one doesn't exist.
2. **Update** an existing spec — typically appending a decision or shifting
   status as work progresses.
3. **Report status** when the user asks what's in flight or what's been
   shipped.
4. **Implement against a spec** — read it before changing code, update it
   when implementation reveals new decisions.
5. **Review on demand** — re-run the spec-reviewer sub-agent against an
   existing spec to surface gaps the user wants checked.

New specs (Mode 1) and substantive edits to existing specs (Mode 2) are
also automatically run through the **review pass** documented below — a
sub-agent that checks for completeness, accuracy against the constitution,
and ambiguity, and surfaces clarifying questions.

Specs are not plans. Plans (under `~/.claude/plans/`) are per-conversation
scratch space. Specs are project artifacts, checked in, intended to outlive
many conversations.

## File layout

```
study-help/
├── specs/
│   ├── README.md           ← index, one row per spec
│   ├── highlights.md
│   ├── notes.md
│   └── auth-sessions.md
└── .claude/skills/feature-planning/
    ├── SKILL.md            ← this file
    └── references/
        └── spec-template.md
```

**Naming.** Spec files use **kebab-case slugs** (`highlights.md`,
`auth-sessions.md`, `passage-reader.md`). No date prefix, no number prefix —
slugs are stable identifiers, easy to link by name. Sort alphabetically.

## Mode 1 — Create a new spec

Trigger: user says "plan a feature", "draft a spec for X", "spec out X", or
runs `/plan-feature [name]`. The feature does not yet have a file in
`specs/`.

### Steps

1. **Confirm the slug.** Derive a kebab-case slug from the feature name. If
   it's ambiguous (e.g. user says "spec out the reader" — could be
   `reader.md` or `passage-reader.md`), ask before proceeding. Slugs are
   stable; renaming later is friction.

2. **Check it doesn't already exist.** `ls specs/` (or read
   `specs/README.md`). If the slug exists, switch to Mode 2 (Update) and
   tell the user.

3. **Read the constitution.** Open `PROJECT_CONSTITUTION.md` and identify
   which sections this feature relates to (in-scope items in §2, principles
   in §3, guardrails in §4, non-goals in §5). The spec must reference real
   section numbers — don't invent links.

4. **Copy the template.** Read `references/spec-template.md` and write it
   to `specs/<slug>.md` with these fields filled in:
   - `# <Feature name>` — title-case the slug.
   - **Status:** `Draft`.
   - **Created:** today's date in `YYYY-MM-DD` (use the `currentDate` from
     the conversation context, not a guess).
   - **Last updated:** same as Created.
   - **Owner:** `unassigned` unless the user names one.
   - **Why:** one paragraph from what the user told you, plus the
     constitutional reference. Don't pad it.

5. **Leave the rest deliberately incomplete.** Goals, Non-goals, User-facing
   behavior, Implementation outline, Open questions, Verification — these
   should have placeholder bullets the user fills in over time. The spec is
   meant to be iterated, not finished in one pass. Resist the urge to
   over-specify on day one.

6. **Update `specs/README.md`.** Insert a row in alphabetical order:
   `| [<slug>](./<slug>.md) | Draft | <one-line summary> |`. If
   `specs/README.md` doesn't exist yet, create it from the template at the
   bottom of this file.

7. **Show the draft.** Print the path and the rendered spec. Tell the user
   which sections are still open, and ask which they'd like to fill in
   first.

8. **Run the review pass.** See [Review pass](#review-pass) below. Mode 1
   always triggers it — even on a brand-new Draft. Catching constitutional
   conflicts and underspecified `Why` paragraphs is cheapest right now.

## Mode 2 — Update an existing spec

Trigger: user says "update the X spec", reports a decision ("we're going to
store highlights as offsets, not XPath"), or transitions a feature's
lifecycle ("highlights is shipped").

### Steps

1. **Read the current spec first.** Always. Targeted edits depend on
   knowing what's there.

2. **Identify which section to touch.** Update only that section. Do not
   rewrite the whole file.
   - **Decisions** — append a dated bullet: `- YYYY-MM-DD: Decided X
     over Y. Reason: ...` Append-only. Never rewrite or reorder past
     entries.
   - **Open questions** — when a question is resolved, do *not* delete it.
     Move it (or reference it) under Decisions with the resolution. The
     question's existence is part of the history.
   - **Status** — only on lifecycle transitions:
     - `Draft → In Progress` when the first PR for this feature lands.
     - `In Progress → Shipped` when the feature is user-visible.
     - `* → Deprecated` when the feature is being removed; add a Decision
       explaining why.
     Announce the transition to the user before making it. Don't change
     status silently. Every transition also triggers a `CLAUDE.md` sanity
     check — see [Keeping repo docs in sync](#keeping-repo-docs-in-sync).
   - **Goals / Non-goals / User-facing behavior / Implementation outline
     / Verification** — fine to edit in place as understanding sharpens.
     Make the edit small and clearly scoped.

3. **Bump `Last updated:`** to today's date on every change.

4. **Mirror status changes into `specs/README.md`.** If you changed
   **Status**, update the corresponding row in the index too.

5. **Show what you changed.** Print the diff or the relevant section so the
   user can verify.

6. **Run the review pass — only on substantive content edits.** If this
   update touched **Goals**, **Non-goals**, **User-facing behavior**,
   **Implementation outline**, or **Verification**, run the
   [Review pass](#review-pass). Do **not** run it for:
   - Decisions appends (a decision is, by definition, settled).
   - Open-questions resolutions (the resolution is now in Decisions).
   - Status transitions (Draft → In Progress, etc. — lifecycle, not
     content).

   This keeps the reviewer out of the way for routine edits while still
   catching shape changes.

## Mode 3 — Status check

Trigger: "what specs do we have", "what's in flight", "what's the status
of X".

### Steps

1. **Read `specs/README.md` first.** It's the index. Don't re-read every
   spec file unless the user asks for detail on one.

2. **Report from the index.** Group by status if helpful (Draft / In
   Progress / Shipped / Deprecated).

3. **Check for drift.** If the user asks about a specific spec, open it
   and verify its **Status** matches the README row. If they disagree,
   tell the user and offer to reconcile — pick the spec file as the
   source of truth (the README is a cache).

4. **Don't fabricate.** If a feature has no spec, say so. Don't invent
   status from memory or git history.

## Mode 4 — Implement against a spec

Trigger: user is doing implementation work on a feature that has a spec
(or asks "let's build X" where `specs/x.md` exists).

### Steps

1. **Read the spec before changing code.** It is the source of intent.
   Goals tell you what success looks like; Non-goals tell you what to
   leave out; Decisions tell you what's already been settled.

2. **If implementation contradicts the spec, stop.** Either:
   - The spec is wrong / out of date → switch to Mode 2, update it, then
     resume coding.
   - The implementation is wrong → align it with the spec.
   Do not silently let the code drift from the spec. The spec is the
   contract.

3. **Capture decisions made during implementation.** When you make a
   non-trivial choice while coding (a library, a schema, a tradeoff),
   append it to the spec's **Decisions** section before moving on.
   Otherwise the rationale gets lost between commits.

4. **Mark Status: In Progress** the first time real code lands for this
   feature. Mark Status: Shipped when it's user-visible.

5. **Refresh `CLAUDE.md` and `STACK.md` if they drifted.** When code
   lands, the repo-state snapshot in `CLAUDE.md` and the tech-choice
   list in `STACK.md` may now be wrong. Update them in the same change
   — see [Keeping repo docs in sync](#keeping-repo-docs-in-sync).

## Mode 5 — Review on demand

Trigger: user says "review the X spec", "check the X spec for gaps",
"audit the X spec", or runs `/review-spec X`. Use this when the user
explicitly wants the reviewer re-run against a spec, independent of any
edit.

### Steps

1. **Resolve the slug.** If the user gave a name, kebab-case it. Confirm
   `specs/<slug>.md` exists by reading the index. If it doesn't, say so —
   don't create one (that's Mode 1).

2. **Run the [Review pass](#review-pass).** Apply user answers and append
   logged questions exactly as documented there.

3. **Don't change Status.** Mode 5 is a quality check, not a lifecycle
   transition. The only writes are the answer-driven edits and the new
   bullets under **Open questions**.

## Review pass

This is the shared procedure that Modes 1, 2, and 5 all use. It invokes
the `spec-reviewer` sub-agent and turns its findings into either
clarifying questions for the user or new bullets under **Open
questions**.

### Steps

1. **Invoke the sub-agent.** Use the `Agent` tool with
   `subagent_type: "spec-reviewer"`. The prompt must contain the absolute
   paths for:
   - the spec being reviewed (`specs/<slug>.md`),
   - `PROJECT_CONSTITUTION.md`,
   - `.claude/skills/feature-planning/references/spec-template.md`.

   Example prompt body:

   ```
   Review the spec at <abs-path>/specs/<slug>.md.
   Constitution: <abs-path>/PROJECT_CONSTITUTION.md
   Template: <abs-path>/.claude/skills/feature-planning/references/spec-template.md
   Return your findings in the strict format defined in your agent
   instructions.
   ```

2. **Parse the output.** The agent returns three sections:
   `## Critical (ask user now)`, `## Log to Open questions`, and
   `## Notes`. Treat `- (none)` as empty.

3. **Ask Critical items via `AskUserQuestion`.** The tool accepts 1–4
   questions per call. If Critical has more than 4 items, ask the first
   4 and roll the rest into the "Log to Open questions" bucket. Each
   AskUserQuestion option should be a plausible answer or "needs more
   thought" — keep options short.

4. **Apply each answer to the spec.** Map the answer to the right
   section (Why, Goals, Non-goals, User-facing behavior, Implementation
   outline, Verification) and edit it in place. Bump `Last updated:` to
   today's date if any edit was made.

5. **Append logged questions to Open questions.** Take every bullet under
   `## Log to Open questions` (and any Critical overflow from step 3)
   and append them verbatim under the spec's `## Open questions`
   section as `- [ ] <question>` bullets. Do not ask the user about
   these.

6. **Surface the Notes.** Print the agent's `## Notes` section to the
   user as-is — it's where it flags constitution miscitations or other
   non-question observations.

7. **Summarize.** One line: "Reviewer flagged N items; you answered M,
   K logged to Open questions." Then show the diff (or the changed
   sections) so the user can verify.

### When the agent returns nothing actionable

If both Critical and Log are `- (none)`, just print the Notes (often
"No critical issues.") and stop. Don't fabricate questions to justify
the call.

## Keeping repo docs in sync

The skill maintains `specs/`, but specs don't live alone. Three other
docs at the repo root describe the same project at different layers,
and they drift if nobody minds them:

- **`PROJECT_CONSTITUTION.md`** — durable principles, guardrails, and
  non-goals. Changes via PR per its §7. Specs *cite* it; specs do not
  edit it.
- **`STACK.md`** — current backend tech choices (language, HTTP layer,
  database, query layer, migrations, sessions, etc.). Swappable; change
  via PR with a short rationale.
- **`CLAUDE.md`** — repo-state snapshot for future Claude sessions
  (package layout, commands, what exists vs. what doesn't). Updated
  whenever reality changes.

### Rules

- **Constitutional conflict → amend the constitution first.** If a
  spec's Goals, User-facing behavior, or Implementation outline
  contradicts §3 principles, §4 guardrails, or §5 non-goals, stop and
  ask the user whether the constitution should be amended (per §7)
  *before* the spec proceeds. Do not silently let a spec contradict
  the constitution. (Reinforces the Mode 1 step 3 read-and-cite rule
  and the Guardrail "Never create a spec for something the
  constitution explicitly excludes.")

- **New tech choice → update `STACK.md` in the same change.** When a
  Decision in any spec introduces a new dependency, switches a tech
  choice, or commits to a framework variant (e.g. "client is React",
  "use Redis for sessions"), also update `STACK.md` — append to the
  table, or to "Explicitly NOT chosen" if the spec rules something
  out. One-line rationale per row; STACK.md is meant to be swappable.

- **Code lands → update `CLAUDE.md` in the same change.** When Mode 4
  ships real code (new packages, new endpoints, new tooling, new env
  vars, new client surface, first tests), refresh the matching part
  of `CLAUDE.md`. The doc is a snapshot, not a wishlist; it should
  describe what is, not what was.

- **Status transitions trigger a `CLAUDE.md` sanity check.** On
  Draft → In Progress and In Progress → Shipped (Mode 2), re-read
  `CLAUDE.md` against the current repo state. If it has drifted —
  claims a layout that's no longer accurate, names files that don't
  exist, omits packages that do exist — update it. The status
  transition and the doc refresh land together.

- **Don't invent in `CLAUDE.md`.** If something is planned but not
  implemented (e.g. `sqlc` listed in `STACK.md` with no codegen output
  yet, a route documented in a spec but not wired up), describe it
  accurately as planned, or omit it. Future agents read `CLAUDE.md`
  as ground truth.

- **The skill never edits `PROJECT_CONSTITUTION.md`.** Even when the
  spec process surfaces a real reason to amend it, the skill flags
  the conflict and stops. The user opens a constitution PR
  separately, per §7. Then the spec resumes.

## Guardrails

- **Never** create a spec without confirming the slug. Slugs are
  identifiers; renaming later breaks links.
- **Never** rewrite the **Decisions** section. It is append-only and
  durable history. If a past decision was wrong, append a new dated
  entry that supersedes it — don't mutate the old one.
- **Never** delete an open question. Resolve it by moving the resolution
  into **Decisions** and crossing it off the Open Questions list (or
  leaving a pointer to the decision).
- **Never** silently change **Status**. State the transition and why,
  then make the change.
- **Never** invent constitutional references. Read
  `PROJECT_CONSTITUTION.md` and cite real section numbers, or omit the
  reference.
- **Never** copy code blocks into specs. Specs describe intent; code is
  the implementation. Pointers to packages or files are fine
  (`internal/highlights/`), full code is not.
- **Never** create a spec for something the constitution explicitly
  excludes (§5 Non-goals). Push back; that's an amendment to the
  constitution first, not a spec.
- **Never** let the `spec-reviewer` sub-agent edit files. Its tools are
  read-only by design. The skill applies edits based on the reviewer's
  findings; the agent only reports.
- **Never** skip the review pass on Mode 1, even for a Draft you think
  is obvious. It's the cheapest moment to catch a spec that conflicts
  with the constitution or names a vague Why.

## `specs/README.md` template

When `specs/README.md` doesn't yet exist, create it with this content:

```markdown
# Specs

Living feature specs. Each entry below points to a markdown file in
this directory. Update both the spec and this index when status
changes.

Status values: **Draft** (being shaped), **In Progress** (code is
landing), **Shipped** (user-visible), **Deprecated** (being removed).

| Spec | Status | Summary |
|---|---|---|
```

Insert one row per spec, alphabetically by slug.

## Worked example — creating the `highlights` spec

User: *"let's plan out highlights"*

1. Slug → `highlights`. Unambiguous, don't ask.
2. `ls specs/` → no existing file. Proceed.
3. Read constitution. Highlights is in §2 (in-scope) and §3 ("Respect the
   text" — annotations don't mutate scripture).
4. Copy template → `specs/highlights.md`. Fill:
   - **Status:** Draft
   - **Created:** 2026-05-02
   - **Last updated:** 2026-05-02
   - **Owner:** unassigned
   - **Why:** "Lets users mark passages they want to revisit. Supports
     PROJECT_CONSTITUTION.md §2 (in-scope: highlighting) and §3 (Respect
     the text — highlights annotate, never mutate)."
5. Leave Goals/Non-goals/etc. as placeholder bullets.
6. Add to README:
   `| [highlights](./highlights.md) | Draft | Range-based, persistent, per-user |`
7. Show the file. Ask the user which section they want to fill in first.

## Worked example — recording a decision

User: *"we decided to store highlights as `(start_offset, end_offset,
chapter_id)` tuples, not XPath ranges, because XPath breaks under any
HTML reflow"*

1. Read `specs/highlights.md`.
2. Append to **Decisions**:
   `- 2026-05-02: Store highlights as (start_offset, end_offset, chapter_id) tuples instead of XPath ranges. Reason: XPath breaks under any HTML reflow; offsets survive layout changes.`
3. Bump **Last updated:** to 2026-05-02.
4. If "XPath vs offsets" was an Open Question, mark it resolved with a
   pointer to the decision.
5. README status didn't change — leave it alone.
6. Show the user the appended bullet.
