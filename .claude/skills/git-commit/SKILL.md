---
name: git-commit
description: Use this skill whenever the user wants help writing a git commit message, committing staged changes, or running /git-commit in this study-help project. The skill reads the staged (or working-tree) diff, drafts a Conventional Commits message, shows it for approval, then stages and commits. Trigger phrases include "commit this", "write a commit message", "draft a commit", "commit my changes", and "/git-commit".
---

# git-commit skill

This skill turns the current diff into a single, well-formed
[Conventional Commits](https://www.conventionalcommits.org/) message
and commits it — but only after the user has approved the wording.

It does **not** push, amend, or open PRs. Those are separate actions
the user can request explicitly.

## Workflow

Follow these four steps, in order, every time this skill runs.

### 1. Read the current state

Run these in parallel:

- `git status` — see what's modified, staged, and untracked.
- `git diff --staged` — what will actually be committed if anything is
  already staged.
- `git log --oneline -10` — the repo's recent commit-message style, in
  case it diverges from this skill's defaults.

If `git diff --staged` is empty, fall back to `git diff HEAD` and tell
the user clearly: *"Nothing is staged yet. I'll draft a message based
on your working-tree changes; tell me which files to stage."*

If there are no changes at all, stop and say so. Don't create an empty
commit.

### 2. Draft a Conventional Commits message

Format:

```
<type>(<scope>): <imperative summary>

<optional body explaining the why, wrapped at 72 chars>
```

**Subject line rules:**

- ≤ 72 characters total.
- Imperative mood: "add", "fix", "rename" — not "added", "adds", "adding".
- No trailing period.
- Lowercase after the colon (unless the first word is a proper noun).

**Allowed types:**

| type      | use for                                                    |
|-----------|------------------------------------------------------------|
| `feat`    | a new user-visible capability                              |
| `fix`     | a bug fix                                                  |
| `refactor`| internal restructuring with no behavior change             |
| `docs`    | docs / comments only                                       |
| `test`    | adding or fixing tests                                     |
| `chore`   | tooling, build config, deps, housekeeping                  |
| `style`   | formatting only (no code change)                           |
| `perf`    | performance improvement                                    |
| `build`   | build system, go.mod / go.sum, Dockerfile                  |
| `ci`      | CI configuration                                           |

**Scope** is optional but encouraged. Pick the package or area
touched. In this repo, common scopes are `esv`, `auth`, `reader`,
`notes`, `highlights`, `api`, `web`. For repo-wide changes (e.g. a
top-level rename, adding `PROJECT_CONSTITUTION.md`), omit the scope.

**Body** is optional. Include one when:

- The *why* isn't obvious from the diff (a constraint, a bug
  reproduction, a decision to defer something).
- Multiple logical changes ended up in the same commit and need
  separating mentally.

Don't include a body that just restates the subject in more words.
Wrap at 72 chars. Don't write more than ~5 lines unless the change
genuinely needs it.

**Do not** add a `Co-Authored-By:` trailer. The user has explicitly
opted out.

### 3. Show the draft and wait for approval

Print the drafted message in a fenced code block, then ask: *"Commit
with this message?"*

Wait for an explicit yes. If the user requests edits, redraft from
their feedback and ask again. Do not run `git commit` until they
agree.

### 4. Stage and commit

Once approved:

- If the user already staged files in step 1, do **not** re-stage —
  just commit.
- Otherwise stage **only** the files relevant to the message, by
  explicit path. Never use `git add -A` or `git add .` — those
  silently sweep in `.env`, build artifacts, and untracked junk.
- Commit using a heredoc to preserve newlines:

```bash
git commit -m "$(cat <<'EOF'
feat(esv): add passage fetcher with retry on 429

Crossway returns 429 under burst load; the client now backs off
once before surfacing the error.
EOF
)"
```

- Run `git status` after to confirm the commit succeeded.

## Examples

### Good

```
feat(esv): add passage fetcher with retry on 429
```
Single concern, scoped, imperative, under 72 chars.

```
fix(auth): allow session cookie inside WebView wrapper

Safari WebView strips SameSite=Lax on cross-origin POSTs, which
broke sign-in from the planned iPad shell. Switching to
SameSite=None + Secure restores it without weakening the web case.
```
Bug fix with body explaining *why* — the constraint isn't visible
from the diff alone.

```
chore: bump go directive to 1.26
```
No scope (touches `go.mod` only), terse but precise.

### Bad

```
update stuff
```
No type, no scope, vague — useless in `git log`.

```
feat: I added a new function that fetches a passage from the ESV API and parses it into a struct that the rest of the codebase can consume
```
Subject is a paragraph. Move detail to the body; keep the subject
under 72 chars.

```
Fixed bug.
```
Past tense, trailing period, no scope, no specifics.

```
feat(esv): add passage fetcher and refactor logging and bump deps
```
Three unrelated concerns mashed into one commit. Either split into
three commits or pick the dominant one and leave the others for
follow-ups.

## Guardrails

- **Never** `git push`, `git commit --amend`, or `git rebase`. If the
  user wants those, they'll ask separately.
- **Never** `git add -A` / `git add .` / `git add -u`. Stage by
  explicit path.
- **Never** stage files that look like secrets: `.env`, `.env.local`,
  anything matching `*credentials*`, `*secret*`, `id_rsa`, `*.pem`.
  If one shows up in `git status`, flag it to the user before
  proceeding.
- **Never** invent context that isn't in the diff. If the *why* isn't
  clear, leave the body off rather than making something up.
- **Never** skip hooks (`--no-verify`). If a pre-commit hook fails,
  fix the underlying issue and create a *new* commit — don't amend.
