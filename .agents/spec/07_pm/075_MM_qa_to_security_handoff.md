---
document_type: Meeting Minutes (Handoff)
version: "1.0"
status: Draft
author: "QA"
created: "2026-08-15"
last_updated: "2026-08-15"
project_name: "7-Solutions Backend Interview Challenge — Task 1: User Management API"
project_id: "SS-INT-001"
classification: "Internal"
tags: [handoff, meeting-minutes, qa, security, polish]
standard_ref:
  - SWEBOK v4 — Cross-Cutting (Project Management)
---

# Meeting Minutes — QA → Security Handoff

> **Meeting ID:** MTG-S01 | **Type:** Handoff
> **Facilitator/Scribe:** QA | **Date:** 2026-08-15

## Attendees

| Role | Persona | Status |
|------|---------|--------|
| QA Engineer | 🧪 QA | ✅ Present (facilitator) |
| Security Engineer | 🔒 SecEng | 📎 Receiver — ACT-S1..S6 status + QA security-relevant findings below |
| Backend Dev | 👨‍💻 Dev | 📎 FYI — DEF-003 fix owner |
| Product Owner | 🎯 PO | 📎 FYI — pending until security sign-off |

## Context

QA polish pass is complete and pushed (`43c3876` on `main`, repo `oat431/7-solution-interview` — **public**). QA hands over to Security **before** PO per process; this document is Security's evidence pack: what QA verified on their behalf, what is open in their lane, and pre-chewed findings.

## What QA verified for the security lane (evidence, not claims)

| Concern (072 origin) | QA evidence | Status |
|---|---|---|
| ACT-S3 — `.env` never committed | `git log --all -- .env` empty; only `.env.example` tracked (placeholder secret); `.gitignore` covers `.env` | ✅ verified |
| ACT-S3 — final image carries no secrets/source | Built runtime image inspected: contains only `/usr/local/bin/api` (+ alpine base); no `.env`, no source, no `.git` | ✅ verified |
| ACT-S3 — non-root user | `docker compose exec api id` → `uid=10001(app) gid=10001(app)` | ✅ verified |
| ACT-S3 — `-ldflags` strip | Dockerfile: `CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w"` | ✅ verified in code |
| ACT-S4 — JWT hardening (unit level) | Full suite green: alg pinned (`WithValidMethods` HS256), `none`-alg rejected, tamper/wrong-secret/expired/missing-sub rejected; secret ≥32 bytes fail-fast at startup (`TestLoadMissingSecretFails`, `TestLoadShortSecretFails`); token never logged (`TestLoggingMiddlewareNeverLogsSecrets`) | ✅ unit-verified; SecEng may re-read per ACT-S4 wording |
| Logging — no secrets in request logs | `TestLoggingMiddlewareNeverLogsSecrets`; middleware logs method/path/status/duration/request_id only | ✅ verified |

## 🔴 Open security findings from the QA pass (pre-chewed)

### FINDING-S1 — DEF-003 upgraded with hard evidence: `.env` IS inside the builder image

QA reproduced and escalated 043 DEF-003 from code inspection to **demonstrated fact**:

```
docker build --target builder -t qa-builder-check .
docker run --rm --entrypoint sh qa-builder-check -c 'ls -la /app'
→ -rwxr-xr-x  /app/.env          (191 bytes — the real local JWT_SECRET)
→ drwxr-xr-x  /app/.git          (2.0M full history)
→ /app/.agents, /app/tools also present
```

There is no `.dockerignore`, so `COPY . .` in the builder stage ingests the whole context. Blast radius today: **builder-stage layers only** — the final image is clean (verified above), so the shipped artifact is safe. But any `docker history`/`docker save` of the builder target, layer-cache sharing, or a multi-target push leaks the real secret. With the repo now **public**, this is the single most worthwhile hygiene fix left.

**Suggested fix (043 DEF-003):** add `.dockerignore` (`.env`, `.git`, `.agents`, `tools/`, `bin/`, `coverage.out`, `.qa_scratch`), rebuild, and re-run QA's two verification probes.

### FINDING-S2 — Login timing side-channel (ACT-S2 context)

Confirmed in code: `AuthService.Login` returns `ErrInvalidCredentials` **before** any bcrypt work when the email is unknown (`auth_service.go` L34-37) — unknown-email responses are ~50-100ms faster than wrong-password ones. Low severity for an interview (no rate limiting either way, A8-notes), but SecEng owns the document-or-fix call per ACT-S2 (dummy `bcrypt.CompareHashAndPassword` against a precomputed hash flattens it in ~3 lines).

### FINDING-S3 — Public repo posture check

`oat431/7-solution-interview` is **public** (interview requirement). QA verified no secret material in history (above). SecEng to confirm comfort with: reflection-enabled gRPC in the shipped config (`grpcurl` discoverable, still JWT-gated — QA verified `Unauthenticated` without token live), and compose Mongo without auth (already flagged for README assumptions per ACT-S5).

## Security action items — carried status from 072

| ID | Item | QA-observed status |
|----|------|--------------------|
| ACT-S1 | `govulncheck ./...` — fix or document (DEC-H04) | ⬜ Open — not run in QA pass (not QA tooling) |
| ACT-S2 | Login timing side-channel | ⬜ Open — FINDING-S2 gives exact location + fix sketch |
| ACT-S3 | Secrets in layers / non-root / ldflags | 🟢 Mostly verifiable by QA — see table above; **remaining:** the `.dockerignore` fix (DEF-003) then re-verify |
| ACT-S4 | JWT hardening claims on fresh read | 🟡 QA unit evidence supplied — SecEng fresh-read remains |
| ACT-S5 | Production gaps in README assumptions (rate limiting, roles A8, plain HTTP local, Mongo no-auth) | ⬜ Open — README currently documents roles (A8) but not rate limiting / TLS termination / Mongo-auth |
| ACT-S6 | Deliverable: 061 security test report notes | ⬜ Open |

## QA environment handed over (re-runnable)

```bash
docker compose ps                 # api + mongo healthy at handoff time
bash scripts/smoke.sh             # 20/20 — includes container-health + gRPC auth probes
go test -race ./...               # unit gate
go test -tags integration -race ./internal/infrastructure/mongodb/   # real-Mongo suite
```

## Decisions requested

None from Security directly — QA's two PO decisions (DEC-Q01/Q02, 074) remain queued until Security sign-off. If SecEng wants the 413 code fixed as part of their pass, coordinate with PO via DEC-Q01.

## Parking Lot

| # | Item | Owner | When |
|---|------|-------|------|
| 1 | `.dockerignore` fix + builder re-verification | Dev (fix) + SecEng (verify, ACT-S3 closure) | this pass |
| 2 | ACT-S5 README assumptions additions | SecEng + Dev | this pass |
| 3 | QA → PO handoff (074 decisions) | QA | after security sign-off |

---

> **Related:** [[000_spec_index]], [[074_MM_qa_polish_complete]], [[072_MM_handoff_polish]], [[043_defect_report]]
