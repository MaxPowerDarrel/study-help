# <Feature name>

**Status:** Draft <!-- Draft | In Progress | Shipped | Deprecated -->
**Created:** YYYY-MM-DD
**Last updated:** YYYY-MM-DD
**Owner:** unassigned

## Why

One paragraph. The problem this solves and the user behavior it enables.
Link to the constitutional principle(s) it serves, e.g. "supports
PROJECT_CONSTITUTION.md §2 (in-scope: highlights) and §3 (Respect the
text)."

## Goals

User-visible outcomes that define success. Bullets, not paragraphs.

- [ ] ...
- [ ] ...

## Non-goals

Things explicitly excluded from this feature, with one-line rationale
each. Forces scope honesty — what we're *not* doing is as important as
what we are.

- ... — *why not*
- ... — *why not*

## User-facing behavior

Concrete description of what the user sees and does. Not implementation.
A senior PM should be able to read this and know what shipping looks
like.

## Implementation outline

High-level shape only:

- Which packages / files (`internal/...`)
- Which API endpoints (`GET /api/...`)
- Which DB tables / migrations
- Which client surfaces

Pointers, not full designs. The spec is durable; the design notes can
stay in the PR.

## Open questions

Questions that need to be resolved. When answered, move the resolution
under **Decisions** and leave a pointer here (don't delete the question
— its existence is part of the history).

- [ ] ...
- [ ] ...

## Decisions

Append-only log. Most recent at the bottom. Never rewrite past entries;
if a decision is reversed, add a new entry that supersedes it.

- YYYY-MM-DD: ...

## Verification

How we'll know it's working: tests, manual flows, metrics, screenshots.

- [ ] ...

## Related

- Other specs this depends on or extends: `[name](./name.md)`
- Constitution sections: `PROJECT_CONSTITUTION.md §X`
- External references: links, RFCs, ESV API docs, etc.
