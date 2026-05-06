# Specs

Living feature specs. Each entry below points to a markdown file in
this directory. Update both the spec and this index when status
changes.

Status values: **Draft** (being shaped), **In Progress** (code is
landing), **Shipped** (user-visible), **Deprecated** (being removed).

| Spec | Status | Summary |
|---|---|---|
| [accounts](./accounts.md) | Shipped | Email + password sign up / sign in / sign out; cookie sessions; foundation for highlights & notes |
| [auto-load-daily-reading](./auto-load-daily-reading.md) | Shipped | Auto-load a daily passage on startup; toggle in settings |
| [daily-annotations](./daily-annotations.md) | In Progress | Highlights, notes, and translation picker on the Daily tab |
| [docker](./docker.md) | Shipped | Package the app as a Docker image for reproducible, portable deployment |
| [highlights](./highlights.md) | Shipped | Range-based, persistent, per-user passage highlights |
| [multi-translation](./multi-translation.md) | Shipped | Pluggable translation foundation; ESV at launch, per-user persisted preference, highlights/notes scoped per translation |
| [niv](./niv.md) | Shipped | NIV translation via YouVersion Platform API; per-translation verse-anchor dispatcher; "Powered by YouVersion" attribution |
| [notes](./notes.md) | Shipped | Private per-user written notes attached to passages |
| [passage-reader](./passage-reader.md) | Shipped | Read a chapter or contiguous passage range; ESV proxied through the server |
| [reader-ui-refresh](./reader-ui-refresh.md) | Shipped | iPad/Safari touch polish, light + dark theme, design tokens; rides a Vite/React/TS bump |
