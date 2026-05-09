# Deploy at Home

**Status:** Deprecated
**Created:** 2026-05-07
**Last updated:** 2026-05-08
**Owner:** unassigned

## Why

[`deploy-aws.md`](./deploy-aws.md) covers a paid single-VM Lightsail
deployment; this spec captures the parallel path of running the same
container stack on a machine the operator already owns at home, exposed
to the public internet via Duck DNS (free dynamic DNS) and an
auto-renewing Let's Encrypt certificate. The end state is the same as
the Lightsail stack — Caddy in front, the stateless Go binary behind —
but the topology has to absorb two home-network realities the cloud
deploy didn't: the WAN IP rotates, and inbound ports come through a
consumer router rather than a cloud NAT/load-balancer. This serves
PROJECT_CONSTITUTION.md §4 (Architectural Guardrails — secrets stay
server-side; the server is stateless) by reusing the same image and
env-var posture, and §1 (Purpose — a personal-scale Bible reader) by
removing the recurring hosting cost for an operator with spare hardware.

## Goals

- [ ] One-command bring-up on a home host via `docker compose up -d`,
      reusing the same `app` image as Lightsail.
- [ ] Public HTTPS at `<subdomain>.duckdns.org`, Let's Encrypt cert
      issued and renewed automatically.
- [ ] Dynamic IP tracked: the Duck DNS A record updates within minutes
      of a WAN-IP change without operator intervention.
- [ ] Operator-facing artifacts (`compose.yaml`, `Caddyfile`,
      `.env.example`, runbook) live in `deploy/home/` alongside the
      existing `deploy/lightsail/`, not on top of it.
- [ ] No additional dependencies on the host beyond Docker + Compose.

## Non-goals

- **No HA / multi-host.** A home deploy is one box. Failure modes are
  power, ISP, and hardware — outside the scope of the app stack.
- **No remote operator access setup (VPN, Tailscale, SSH hardening).**
  The host is the operator's; how they reach it for `docker compose`
  commands is their concern.
- **No router-config automation.** Port-forwarding 443 to the host
  is a one-time manual step on the operator's router; this spec
  documents it but does not script it.
- **No CI-driven deploy.** Lightsail has a `workflow_dispatch` deploy
  job; the home deploy is operator-pulled (`docker compose pull && up
  -d`) since it sits on a residential network with no inbound SSH from
  GitHub Actions runners.
- **No revival of server-side state.** The home deploy inherits the
  stateless posture of the current binary (PROJECT_CONSTITUTION.md §4);
  no SQLite, no Litestream, no host-mount data volume.
- **No IPv6-only path.** Duck DNS supports AAAA records, but residential
  CGNAT and dual-stack quirks make IPv4 + HTTP-01 the documented
  default. IPv6 is a follow-up if the operator's ISP demands it.

## User-facing behavior

The operator (also the only user):

1. Registers a subdomain at `duckdns.org` and copies the account token.
2. Forwards TCP `80` and `443` from the router to the host's LAN IP.
   Both are required: `:80` for the HTTP-01 ACME challenge, `:443` for
   the site itself (see Decisions 2026-05-08 (HTTP-01 supersede)).
3. Drops `ESV_API_KEY`, `YOUVERSION_APP_KEY`, `DUCKDNS_SUBDOMAIN`, and
   `DUCKDNS_TOKEN` into `deploy/home/.env`.
4. Runs `docker compose up -d`. Within a few minutes, navigating to
   `https://<subdomain>.duckdns.org` from anywhere on the public
   internet returns the SPA with a valid TLS cert.
5. When the home WAN IP changes (ISP renewal, router reboot), the
   stack catches up on its own; no manual A-record edit.

## Implementation outline

- New directory `deploy/home/`:
  - `compose.yaml` — three services: `app` (image pulled from registry,
    same as Lightsail), `caddy` (TLS termination + reverse proxy,
    stock `caddy:2-alpine` digest-pinned, listening on :80 for the
    HTTP-01 ACME challenge and :443 for the site), and `duckdns`
    (`lscr.io/linuxserver/duckdns` sidecar that keeps the A record in
    sync with the home WAN IP).
  - `Caddyfile` — single site block for `<subdomain>.duckdns.org`
    with `reverse_proxy app:8080`. No `tls` directive: Caddy defaults
    to HTTP-01 (and TLS-ALPN-01).
  - No custom Caddy build. The original spec called for a
    `Caddy.Dockerfile` two-stage build with the `caddy-dns/duckdns`
    plugin baked in for DNS-01; that file was retired with the
    HTTP-01 supersede (Decisions 2026-05-08), since stock
    `caddy:2-alpine` does HTTP-01 out of the box.
  - `.env.example` — `APP_IMAGE` (digest-pinned in the form
    `ghcr.io/<you>/study-help@sha256:<digest>` — placeholder mirrors
    Lightsail's image-ref shape but uses digest pinning per
    Decisions 2026-05-08 superseding), `ESV_API_KEY`,
    `YOUVERSION_APP_KEY`, `DUCKDNS_SUBDOMAIN`, `DUCKDNS_TOKEN`.
  - Named volumes: `caddy_home_data`, `caddy_home_config`, and
    `duckdns_config`. The Caddy volumes are distinct from Lightsail's
    `caddy_data` / `caddy_config` so an operator who accidentally
    points the wrong compose file at the same host doesn't get
    silent state reuse (defense-in-depth — running both at once is
    explicitly unsupported per Non-goals, but volume isolation
    survives a sequential mistake).
  - `README.md` — runbook covering: CGNAT pre-flight
    (`dig +short myip.opendns.com @resolver1.opendns.com` returns
    the public WAN IP — `myip.opendns.com` is OpenDNS's
    well-known special record that resolves to the querier's own
    IP, which is why `+short` returns an A record rather than a
    CNAME;
    compare to the router's WAN-IP page — if they differ, the
    operator is behind CGNAT and the deploy can't work without a
    static IP or a different topology), router port-forward (443
    only), host firewall note (allow 443/tcp through ufw/firewalld
    if enabled), Duck DNS registration, the DNS-propagation wait
    (operator confirms `dig <subdomain>.duckdns.org @ns1.duckdns.org`
    — fall back to `@ns2.duckdns.org` or `@ns3.duckdns.org` if ns1
    is unreachable — returns the home IP before expecting Caddy to
    issue a cert), first-cert observation, secret rotation (edit
    `.env` and `docker compose up -d`), and the IP-update
    verification drill.
- No code changes to `internal/`, `main.go`, or the SPA. The home
  deploy is purely an operator-side packaging concern; the binary is
  the same artifact Lightsail runs.
- No changes to the repo-root `compose.yaml` (local dev) or
  `deploy/lightsail/` (cloud deploy). The three compose files stay
  independent; running more than one at once on the same host is not
  supported.

## Open questions

- [x] **HTTP-01 vs DNS-01 for Let's Encrypt.** → Resolved: DNS-01
      via `caddy-dns/duckdns` is the documented default; see
      Decisions 2026-05-07.
- [x] **Duck DNS updater: sidecar container vs host cron.** →
      Resolved: `lscr.io/linuxserver/duckdns` sidecar inside the
      compose stack; see Decisions 2026-05-07.
- [x] **What does the operator do when their ISP blocks :80?** →
      Resolved by the DNS-01 default: port 80 is no longer required
      for cert issuance; see Decisions 2026-05-07.
- [x] **Update interval.** → Resolved: 5 minutes
      (`lscr.io/linuxserver/duckdns` default, matches Duck DNS's own
      guidance); see Decisions 2026-05-08.
- [x] **Cert sharing with the local-dev `compose.yaml`.** → Resolved:
      named volumes `caddy_home_data` and `caddy_home_config`,
      distinct from Lightsail's; see Decisions 2026-05-08.
- [x] What is the documented default Duck DNS update interval? →
      Resolved: 5 minutes; see Decisions 2026-05-08.
- [x] What is the explicit Caddy data-volume name for `deploy/home/`? →
      Resolved: `caddy_home_data` / `caddy_home_config`; see Decisions
      2026-05-08.
- [x] Does the home deploy pin `APP_IMAGE` to a tag/digest, or track
      `latest`? → Resolved: digest-pinned (`@sha256:...`); see
      Decisions 2026-05-08 (initial tag-pin entry) and 2026-05-08
      (superseding) — current state is the superseding entry.
- [x] How does the operator rotate `DUCKDNS_TOKEN` /
      `ESV_API_KEY` / `YOUVERSION_APP_KEY`? → Resolved: edit `.env`
      and `docker compose up -d`, documented as a one-liner in the
      runbook; see Decisions 2026-05-08.
- [x] Is there any guidance for operators on CGNAT detection? →
      Resolved: pre-flight check in the runbook (`whatismyip.com` vs
      router WAN IP); see Decisions 2026-05-08.
- [x] Should the runbook include a hard expectation about host
      firewall rules? → Resolved: one-line note covering ufw/firewalld
      and the 443/tcp allow rule; see Decisions 2026-05-08.
- [x] Does the `duckdns` sidecar need its own `/config` volume? →
      Resolved: yes, `duckdns_config` named volume; see Decisions
      2026-05-08.
- [x] **Round-3 follow-up that triggered the supersede:** the prior
      round resolved `APP_IMAGE` to `sha-<git>` tag-pinning; the
      round-3 reviewer then asked whether the Caddy base images
      (`caddy:2-builder`, `caddy:2-alpine`) should pin via the same
      scheme. Asking the same pin question for the Caddy bases
      surfaced that even tag-pinning isn't ideal for a stack the
      operator updates by hand, and the answer ("digest-pin both")
      retroactively superseded the original tag-pin decision rather
      than running in parallel with it. → Resolved: both
      digest-pinned. See Decisions 2026-05-08 (superseding).
- [x] Should `.env.example` document the full image reference? →
      Resolved: yes, mirrors Lightsail's `ghcr.io/<you>/study-help`
      shape but with `@sha256:<digest>` pinning. See Decisions
      2026-05-08 (superseding).
- [x] Should the IP-change drill instruct the operator to query
      Duck DNS authoritatively to bypass resolver caching? →
      Resolved: yes, `dig @ns1.duckdns.org`; see Verification
      and Decisions 2026-05-08.
- [x] Should the runbook spell out the DNS-propagation wait that
      precedes cert issuance? → Resolved: yes, the runbook step
      requires authoritative-resolver confirmation before expecting
      Caddy to issue a cert; see Implementation outline and
      Decisions 2026-05-08.
- [x] Should the spec reconcile "no simultaneous run" with the
      distinct-volume rationale? → Resolved: clarified in
      Implementation outline as defense-in-depth against
      sequential-reuse mistakes; simultaneous run remains
      unsupported per Non-goals. See Decisions 2026-05-08.
- [x] The IP-change drill hardcodes `@ns1.duckdns.org`; should the
      alternates be documented? → Resolved: ns1 stays primary, ns2/
      ns3 documented as fallbacks in both the runbook description
      and the IP-change drill. See Implementation outline,
      Verification, and Decisions 2026-05-08 (refinement pass).
- [x] DNS-propagation decision wording is technically loose; tighten
      to refer to the Let's Encrypt CA's resolution path? → Resolved:
      a follow-up Decision attributes DNS-01 validation to the CA
      (not Caddy); the original entry stands as written. See
      Decisions 2026-05-08 (clarifying-DNS-propagation entry).
- [x] CGNAT pre-flight: should it document an alternate or a more
      robust canonical method? → Resolved: superseded to
      `dig +short myip.opendns.com @resolver1.opendns.com`; the
      original `whatismyip.com` decision stays in the log. See
      Decisions 2026-05-08 (superseding-CGNAT entry).
- [x] Should the supersede entry call out the rationale-shift
      (consistency-with-Lightsail traded for home-stack
      reproducibility)? → Resolved: yes; captured as a discrete
      follow-up Decision rather than mutating the supersede entry.
      See Decisions 2026-05-08 (extending-digest-pin entry).
- [x] The "extending-digest-pin" Decision overlaps in narrative with
      both the superseding entry and the "deliberate but not
      propagated" entry — should it be reframed or merged? →
      Resolved: leave as-is. The superseding entry stated the new
      rationale; the deliberate-divergence entry justified
      non-propagation; the extending entry adds the rationale-shift
      framing (the consistency-with-Lightsail rationale of the
      original tag-pin entry was *retired*, not silently
      sidestepped). All three say something the others don't, and
      mutating any of them to remove the overlap would violate
      append-only. See Decisions 2026-05-08 (kept-extending entry).
- [x] The Open questions list contains two entries resolving to the
      same answer (digest-pinned); should the second be reworded as
      the supersede trigger? → Resolved: reworded in place to make
      its supersede-trigger role explicit. See the "Round-3 follow-up
      that triggered the supersede" question above.
- [x] The Verification "IP-change drill" inlines an unnecessary
      cross-reference to the Let's Encrypt CA — trim or keep? →
      Resolved: trimmed. The drill is about updater behavior; the
      CA-attribution context belongs in the DNS-propagation Decision.
      See Verification.
- [x] The runbook's CGNAT pre-flight description should name
      `myip.opendns.com` explicitly so an operator understands why
      `+short` returns an A record. → Resolved: clarified inline.
      See Implementation outline (CGNAT pre-flight bullet).

## Decisions

- 2026-05-07: Spec created.
- 2026-05-07: **DNS-01 is the documented default for Let's Encrypt
  issuance.** Caddy uses the `caddy-dns/duckdns` plugin and the
  Duck DNS account token to satisfy the ACME challenge. Reason:
  removes the inbound-port-80 requirement (some residential ISPs
  block :80 and exposing it broadens attack surface for no benefit
  on a domain that's already locked to one host), and the same
  Duck DNS token already in `.env` does double duty for issuance
  and dynamic-IP updates. Trade: requires a custom Caddy image
  built from the `caddy:2-builder` stage; documented in
  `Caddy.Dockerfile`.
- 2026-05-07: **Duck DNS updater is a sidecar in `compose.yaml`,
  not a host cron.** Image is `lscr.io/linuxserver/duckdns`.
  Reason: keeps the entire deploy inside `docker compose` so the
  operator has one start/stop/log surface, and the updater
  survives host reboots without a separate systemd unit.
- 2026-05-07: **Renewal verification is the operator-side log-grep
  at the 60-day mark; no synthetic ACME-staging pre-flight.**
  Reason: personal-scale deploy where brief downtime from a missed
  renewal is acceptable, and Let's Encrypt rate limits aren't a
  realistic concern for a single-domain personal site.
- 2026-05-08: **Duck DNS update interval is 5 minutes.** Matches
  the `lscr.io/linuxserver/duckdns` default and Duck DNS's own
  published recommendation. Reason: ~288 calls/day is courteous to
  a free service, and a 5-minute worst-case outage after a WAN-IP
  rotation is acceptable for a personal-scale site. Tighter
  intervals risk soft rate-limiting and don't materially help.
- 2026-05-08: **`APP_IMAGE` is tag-pinned to `sha-<git>`, mirroring
  `deploy-aws.md`.** Reason: keeps reproducibility and rollback
  semantics consistent across cloud and home deploys, and the same
  CI image-build path produces the tag both stacks consume. Trade:
  the operator must explicitly edit `.env` and `docker compose pull`
  to update — there is no auto-update on the home stack (consistent
  with the "No CI-driven deploy" non-goal).
- 2026-05-08: **Caddy state lives in named volumes `caddy_home_data`
  and `caddy_home_config`.** Reason: explicitly distinct from
  Lightsail's `caddy_data` / `caddy_config` so an operator who
  ever lands the two compose files on the same host doesn't get
  silent volume reuse and a cross-stack ACME account leak.
- 2026-05-08: **CGNAT is handled as a pre-flight check in the
  runbook, not by the deploy itself.** Operator compares
  `whatismyip.com` to the router's WAN IP; if they differ, the
  deploy will not work and they need a static IP or a different
  topology. Reason: catches the failure mode at a layer where
  Caddy logs would be misleading; punts deeper workarounds (Cloudflare
  Tunnel, Tailscale Funnel) to a future spec since they replace
  Caddy + Duck DNS rather than supplement them.
- 2026-05-08: **Secret rotation is documented as a one-line runbook
  step: edit `.env`, `docker compose up -d`.** Same posture as
  Lightsail. Reason: Compose env-substitution makes rotation a file
  edit; per-secret upstream-URL pages would rot and aren't in scope
  for the runbook.
- 2026-05-08: **Host-firewall guidance is a single runbook line
  covering ufw/firewalld with a 443/tcp allow rule.** Reason: a
  router port-forward with a host firewall blocking the inbound is
  the second-most-common silent failure after CGNAT; one sentence
  prevents it without bloating the runbook into an ops manual.
- 2026-05-08: **`duckdns` sidecar persists state to a
  `duckdns_config` named volume.** Reason: matches the
  `lscr.io/linuxserver/duckdns` documented convention, persists log
  history and any internal cache across restarts, and costs nothing
  operationally — the volume holds a few hundred bytes.
- 2026-05-08 (superseding the earlier 2026-05-08 tag-pin decision):
  **Both `APP_IMAGE` and the `Caddy.Dockerfile` base images are
  digest-pinned (`@sha256:...`).** Reason: the home stack has no CI
  rollout pipeline (Non-goal: no CI-driven deploy), so the operator
  pulls and updates by hand; digest pinning gives unambiguous
  what-is-running answers and ensures a third-party `caddy:2-alpine`
  push can't change the runtime under the operator. Trade: each
  update edits two pinned digests instead of one tag string. The
  earlier 2026-05-08 tag-pin decision stands in the log as the
  initial choice; this entry supersedes it.
- 2026-05-08: **`.env.example` documents the full image reference
  shape `ghcr.io/<you>/study-help@sha256:<digest>`.** Reason:
  mirrors the registry path the Lightsail README documents while
  showing the digest-pin form chosen above, so an operator copying
  the placeholder knows both the registry namespace and the pin
  syntax.
- 2026-05-08: **DNS-propagation wait is an explicit runbook step,
  and the IP-change drill queries Duck DNS authoritatively.**
  Operator runs `dig <subdomain>.duckdns.org @ns1.duckdns.org`
  before expecting Caddy to obtain a cert (DNS-01 challenge
  validates against the record itself, but Caddy's check still
  goes through resolver TTL paths if asked indirectly), and the
  IP-change drill uses the same query so the test isn't
  misleading by hitting a stale resolver cache. Reason:
  prevents the most common false-failure mode where the cert
  issuance "didn't work" simply because DNS hadn't propagated.
- 2026-05-08: **Volume isolation between `deploy/home/` and
  `deploy/lightsail/` is defense-in-depth.** Running both compose
  files at once on the same host remains unsupported per
  Non-goals; the distinct names (`caddy_home_data`,
  `caddy_home_config`) only protect against the operator
  accidentally swapping deploys without a `docker compose down -v`
  in between. Documented to remove the apparent contradiction
  between Non-goals and the volume-naming rationale.
- 2026-05-08: **The home/Lightsail pinning divergence is
  deliberate but not propagated.** With this spec's adoption of
  `@sha256:` digest pins, the home stack now uses a stronger
  reproducibility posture than `deploy-aws.md`'s `sha-<git>`
  tag pins. We chose **not** to bring `deploy-aws.md` along
  because it is still Draft and unimplemented; the Lightsail
  artifacts (`deploy/lightsail/deploy.sh`, `.env.example`) work
  with tag pins today, and either spec may evolve independently.
  If `deploy-aws.md` ever moves toward implementation,
  re-evaluate the pin scheme then. This entry is the audit trail
  so a future reader doesn't read the divergence as an oversight.
- 2026-05-08: **Duck DNS authoritative-server fallbacks are
  documented (`ns2`/`ns3`).** The DNS-propagation runbook step and
  the IP-change drill name `@ns1.duckdns.org` as the primary
  query target with `@ns2.duckdns.org` and `@ns3.duckdns.org` as
  fallbacks. Reason: a transient outage of any single Duck DNS
  nameserver shouldn't fail the operator's drill for a non-app
  reason; Duck DNS publishes three authoritative servers
  precisely so a single-NS failure isn't load-bearing.
- 2026-05-08 (superseding the earlier 2026-05-08 CGNAT decision):
  **CGNAT pre-flight uses `dig +short myip.opendns.com
  @resolver1.opendns.com`, not `whatismyip.com`.** Reason: OpenDNS
  publishes a well-known special record that returns the caller's
  public IP, so the check has no third-party HTML dependency,
  produces a single line the operator can compare to the
  router's WAN-IP page, and survives `whatismyip.com` going dark
  or changing layout. The earlier decision's intent (catch CGNAT
  before the operator hits opaque Caddy logs) is unchanged; this
  entry supersedes only the mechanism.
- 2026-05-08 (clarifying the earlier 2026-05-08 DNS-propagation
  decision): **DNS-01 validation is performed by Let's Encrypt's
  CA against Duck DNS's authoritative servers; the operator's
  authoritative `dig` query observes the same answer the CA will,
  not "Caddy's resolution path."** Reason: the prior decision
  entry's parenthetical was technically loose — Caddy itself
  doesn't validate DNS-01, the CA does. The intent (use the
  authoritative `dig` query so the test isn't misled by resolver
  TTL caching) stands; this entry exists only to attribute the
  validation to the right actor.
- 2026-05-08 (extending the earlier 2026-05-08 superseding
  digest-pin decision): **The supersede was a deliberate
  rationale-shift from "consistency with `deploy-aws.md`" to
  "home-stack reproducibility."** The original tag-pin entry
  cited cross-spec consistency as its rationale; the superseding
  digest-pin entry traded that rationale away because verifiable
  what's-running on a manually-managed host is more valuable
  than cross-stack symmetry, especially when the cloud stack is
  still Draft. Captured as a discrete entry (rather than mutating
  the superseding entry) so the rationale-shift is itself part
  of the audit trail.
- 2026-05-08 (kept-extending): **The "extending-digest-pin" entry
  stays in the log despite narrative overlap with the superseding
  and deliberate-divergence entries.** Each of the three says
  something net-new (new rationale; non-propagation justification;
  rationale-shift attribution); collapsing them would either lose
  one of those framings or require mutating committed Decisions,
  which violates the append-only rule. Reason: redundancy in
  prose is the price of preserving the audit trail at the
  granularity of the questions that drove each entry.
- 2026-05-08 (HTTP-01 supersede; supersedes the 2026-05-07 DNS-01
  decision): **HTTP-01 is the documented ACME challenge type, not
  DNS-01.** During operator first-cert testing on 2026-05-08, the
  DNS-01 path consistently failed with `SERVFAIL` on the CA's TXT
  lookup of `_acme-challenge.<sub>.duckdns.org`. letsdebug.net
  reproduced the failure and surfaced a `TXTDoubleLabel` finding:
  the `caddy-dns/duckdns` plugin was setting the TXT at
  `_acme-challenge.<sub>.duckdns.org.<sub>.duckdns.org` (a
  malformed name produced by appending the FQDN onto the zone)
  rather than calling Duck DNS's update API for the apex sub.
  Pinning Caddy's challenge resolvers to `1.1.1.1`/`8.8.8.8`
  cleared an earlier SOA-walk SERVFAIL but couldn't reach the
  plugin-side bug. Switching to HTTP-01 sidesteps every Duck DNS
  TXT moving part: Caddy listens on :80, Boulder fetches an inbound
  challenge token over plaintext HTTP, no plugin or TXT involved.
  Caddy then redirects all :80 traffic to :443 once the cert is
  issued, so no plaintext content is ever served. Trade: the
  operator now forwards both 443 and 80 from the router (the
  original "ISP blocks :80" question moves from "resolved by
  DNS-01" to "operator's problem; documented as a CGNAT-style
  pre-flight" — out of scope here since the spec assumes a
  residential ISP that doesn't block :80) and the host firewall
  must allow both. The custom Caddy build (`Caddy.Dockerfile`,
  `xcaddy build --with caddy-dns/duckdns`) is retired in favor of
  stock `caddy:2-alpine`. The earlier 2026-05-07 DNS-01 decision
  stands in the log as the initial choice and as the audit trail
  for why the stack was structured the way it was; the prose in
  Implementation outline and User-facing behavior is updated to
  reflect the new state, while the original DNS-01 reasoning is
  preserved here.
- 2026-05-08 (deprecated): **The home self-hosted deploy is
  deprecated; `deploy/home/` is removed.** After the HTTP-01
  supersede above, operator testing surfaced a second, independent
  DuckDNS reliability failure: Caddy correctly served the HTTP
  challenge to Boulder's primary perspective, but Boulder's
  secondary multi-perspective validation consistently timed out
  resolving the A record on DuckDNS's NS pool (`ns1`–`ns9.duckdns.org`,
  several of which were intermittently unreachable from the
  operator's vantage point and from public recursive resolvers).
  Combined with the earlier `caddy-dns/duckdns` plugin bug
  (TXT-double-label, see HTTP-01 supersede entry), DuckDNS is
  untenable as the ACME backstop for this stack: one failure was
  a plugin/libdns bug, the other is upstream NS infrastructure we
  can't fix from this side. The `deploy/lightsail/` cloud variant
  remains the supported deploy path; local development continues
  to use `go run .` and the repo-root `compose.yaml`. The
  `deploy/home/` operational artifacts (`Caddyfile`,
  `compose.yaml`, `README.md`, `.env`, `.env.example`) are
  deleted; this spec is kept as the historical record of why the
  path was attempted and dropped, matching the precedent set by
  `notes`, `accounts`, `highlights`, and `daily-annotations`.

## Verification

- [ ] **First-cert drill.** Fresh host, fresh Duck DNS subdomain.
      `docker compose up -d`, then watch `docker compose logs -f caddy`
      for the first ACME challenge to succeed. Expect a valid
      Let's Encrypt cert within ~60s of DNS resolving to the home IP.
- [ ] **Public-reachability smoke.** From a tethered phone (off the
      home LAN), open `https://<subdomain>.duckdns.org`, fetch a
      passage, switch translations. SPA loads, API responds, no cert
      warnings.
- [ ] **IP-change drill.** Force a router WAN-IP renewal (or simulate
      by pointing the Duck DNS A record at `0.0.0.0` and waiting for
      the updater to overwrite it). Confirm via
      `dig <subdomain>.duckdns.org @ns1.duckdns.org` (fall back to
      `@ns2` or `@ns3` if ns1 is transiently unavailable; querying
      authoritatively bypasses resolver-side TTL caching) that the A
      record returns to the correct IP within ~5 minutes.
- [ ] **Reboot drill.** `sudo reboot` the host. `docker compose ps`
      after boot shows all services up; the site is reachable without
      manual intervention.
- [ ] **Renewal sanity.** Caddy renews ~30 days before expiry; this
      can't be verified at deploy time, but the runbook should include
      the log grep (`certificate obtained successfully`) the operator
      runs at the 60-day mark to confirm renewal happened.

## Related

- [`deploy-aws.md`](./deploy-aws.md) — paid cloud variant of the same
  shape (Caddy + Compose + the same image). The home deploy diverges
  on dynamic-DNS handling and the absence of CI-driven deploy.
- [`docker.md`](./docker.md) — image packaging this spec consumes
  unchanged.
- [`PROJECT_CONSTITUTION.md`](../PROJECT_CONSTITUTION.md) — §1
  (purpose: personal-scale reader), §4 (server is stateless; secrets
  stay server-side).
- External: [Duck DNS](https://www.duckdns.org/),
  [Caddy automatic HTTPS](https://caddyserver.com/docs/automatic-https),
  [`caddy-dns/duckdns`](https://github.com/caddy-dns/duckdns).
