# Considerations Before Making `study-help` Public

## Context

What to weigh before flipping this repo from private to public on GitHub. The good news from a quick audit: there are **no blockers** — no secrets in code, no secrets in git history, `.env` is gitignored, `.env.example` only holds placeholders, the Go server has zero third-party deps, and the GitHub Actions workflows reference repo secrets correctly (no fork-exposed token risks). What's left is a short list of cleanup items, plus a few decisions to make consciously rather than by default.

---

## Things to decide / fix before flipping public

### 1. Add a LICENSE file (decision needed)

There's no `LICENSE` at the root. Without one, the repo is technically "all rights reserved" — public viewers can read but not legally fork, modify, or reuse. For a personal project being showcased, that may be fine. If open source is the goal, pick one:

- **MIT** — most permissive, most common for personal projects.
- **Apache 2.0** — like MIT but with an explicit patent grant.
- **None / no LICENSE** — keeps it readable but legally closed; fine if the goal is portfolio/visibility, not contributions.

GitHub will surface the license badge on the repo page, and tools like `npm`/`go get` users may filter on it.

### 2. Personal domain references in docs

The deployment domain `study.darrel.io` appears in:

- `specs/deploy-aws.md:9`
- `specs/README.md`

Not sensitive — it's already resolvable on the public DNS — but a public repo will advertise the personal deployment URL. Options:

- Leave it (it's fine; it's where the app lives).
- Replace with a generic placeholder (`study.example.com`) and keep the real domain only in private deployment notes / GitHub Actions secrets.

### 3. Hardcoded GitHub org in `deploy/lightsail/bootstrap.sh`

Line 49: `REPO="${REPO:-MaxPowerDarrel/study-help}"`

Already overridable via env var, so not a functional problem. But if the repo moves to a different org/username when going public, the default will be stale. Either update to whatever the public path becomes, or drop the default and require `REPO=` to be set.

### 4. Git remote will need updating

Current remote: `https://github.com/MaxPowerDarrel/study-help.git`. If going public *under the same account*, no change. If transferring to a new account/org, update the remote post-transfer.

### 5. GitHub Actions and forks (worth knowing, no action required)

Once public, anyone can fork and open PRs. Two checks:

- `.github/workflows/build-image.yml` uses `secrets.GITHUB_TOKEN` (auto-scoped, safe).
- `.github/workflows/deploy.yml` uses `DEPLOY_HOST`, `DEPLOY_USER`, `DEPLOY_SSH_KEY` — these are repo secrets and **are not exposed to workflows triggered by `pull_request` from forks** by GitHub's default. So a forker can't exfiltrate them. Don't change the workflow trigger to `pull_request_target` without re-reading [GitHub's guidance on that pattern](https://securitylab.github.com/research/github-actions-preventing-pwn-requests/) — that's the foot-gun.

### 6. Issue / discussion settings

When public, GitHub will let anyone open issues by default. Decide:

- Enable **Discussions**? (good for Q&A on a portfolio project)
- Restrict issue creation to authenticated users? (default is fine; spam is rare on small repos)
- Enable **Dependabot alerts and security updates**? (free, recommended)
- Turn on **secret scanning** + **push protection**? (free for public repos, catches accidental secret commits — strongly recommend)

### 7. CodeQL / security scanning (optional, free for public repos)

GitHub offers free CodeQL scanning for public repos. Worth a one-click enable for the Go and TypeScript code.

### 8. Branch protection on `main`

CLAUDE.md already says no work lands on `main` without a PR. With the repo public, formalize that as a branch protection rule (require PR, require status checks to pass) so a stray push can't slip through.

### 9. Things to call out in the README

The current README is well-written and public-suitable. One thing to consider adding: a short note that this is a personal project, what its scope is, and whether contributions are accepted. It frames expectations for visitors and saves drive-by issue debates.

---

## What you do *not* need to worry about

- **Secrets in code or history** — clean. No `.env`, no embedded keys, test fixtures use obvious mock tokens.
- **Dependencies** — Go side is stdlib-only; web side is standard public packages.
- **Dockerfile / compose** — no private registries or baked-in creds.
- **Specs / CLAUDE.md** — no internal team names, customer data, or business secrets. They're technical decision logs that actually *help* a public reader.

---

## Suggested order of operations

1. Decide on license; add `LICENSE` if desired.
2. Decide whether to scrub `study.darrel.io` from specs.
3. Update or remove the `MaxPowerDarrel/study-help` default in `bootstrap.sh` if the path will change.
4. Flip repo to public in GitHub settings.
5. In the now-public repo's settings: enable secret scanning + push protection, Dependabot, and CodeQL. Add a branch protection rule on `main`.
6. (Optional) Add a one-paragraph "About this project" section to the README framing it as personal/portfolio.

## Verification

Not a code change, so nothing to test. Verification = visual check:

- `gh repo view --web` after flipping to confirm the public landing page looks right (license badge, language stats, README rendering).
- Open an incognito window and confirm the repo is reachable without auth.
- Trigger one CI run on a branch to confirm Actions still work post-flip (secrets behave the same).
