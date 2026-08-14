---
document_type: Meeting Minutes (Handoff)
version: "1.0"
status: Draft
author: "Dev"
created: "2026-08-15"
last_updated: "2026-08-15"
project_name: "7-Solutions Backend Interview Challenge — Task 1: User Management API"
project_id: "SS-INT-001"
classification: "Internal"
tags: [handoff, meeting-minutes, dev, qa, polish, defect-evidence]
standard_ref:
  - SWEBOK v4 — Cross-Cutting (Project Management)
---

# Meeting Minutes — Dev → QA Handoff (Polish Pass Complete)

> **Meeting ID:** MTG-D01 | **Type:** Handoff
> **Facilitator/Scribe:** Dev | **Date:** 2026-08-15

## Attendees

| Role | Persona | Status |
|------|---------|--------|
| Backend Dev | 👨‍💻 Dev | ✅ Present (facilitator) |
| QA Engineer | 🧪 QA | ⬜ Receiver — action items below |
| Product Owner | 🎯 PO | 📎 FYI — one decision request (FINDING-D1) |

## Context (baseline being handed over)

Polish pass (072 ACT-D1..D5) is **complete, verified, and committed** at `a8c6a52`
on `main` (pushed). Baseline evidence, re-runnable by QA:

- `gofmt -l .` clean, `go vet ./...` clean, `go build ./...` clean
- `go test -race -count=1 ./...` — 7/7 packages green
- `bash scripts/smoke.sh` — **13 passed, 0 failed** (REST, vs live compose stack)
- Container `sevensolution-api-1` — `healthy`; in-container `wget localhost:8080/healthz` → 200
- gRPC: `grpcurl` without metadata → `Unauthenticated` (A11 enforced)

## Dev deliverables this pass (closes 072 Dev action items)

| 072 ID | Status | What landed |
|--------|--------|-------------|
| ACT-D1 | ✅ Done | `fiber.Config`: ReadTimeout 10s / WriteTimeout 15s / IdleTimeout 60s (Fiber default was unlimited) |
| ACT-D2 | ✅ Done | Mongo client: `SetMaxPoolSize(50)`, `SetMinPoolSize(5)`, `SetServerSelectionTimeout(5s)` |
| ACT-D3 | ⏭️ Skipped (optional) | `golangci-lint` not installed on dev machine; revisit if wanted |
| ACT-D4 | ⏭️ Skipped (optional) | Per-request context timeout judged unnecessary once server timeouts existed — would complicate shutdown for no gain |
| ACT-D5 | ✅ Done | Package tree unchanged; 025/README needed no structural updates |

Additional clean-code pass (per user's Clean Code notes, Ch. 4):

- Removed `timeNow`/`timeSinceMS` dead test indirection in `middleware.go` — comment claimed test determinism that no test used
- Inlined `disallowUnknownFields` into `decodeJSON` (`errors.go`) — comment mislabeled stdlib `encoding/json` as "the Fiber body decoder"
- Validation kept in the **domain layer** (not Fiber `StructValidator` / `go-playground`): one rule set shared by REST + gRPC, exact 022 error envelope. Reviewed against the official Fiber validation guide; decision documented in README assumptions table

## 🔴 Bug fixed this pass (regression from Fiber migration)

**Docker healthcheck was failing since `2cac38e`.** Fiber v3 defaults to
`ListenerNetwork: tcp4` (IPv4-only); the in-container busybox `wget localhost`
resolves to `::1` → connection refused → compose reported `unhealthy`, while
host-side probes (IPv4 port forward) kept passing — which is why smoke 13/13
never caught it. Fixed with `fiber.ListenConfig{ListenerNetwork: fiber.NetworkTCP}`
(restores the dual-stack listener the pre-Fiber net/http server had).

**QA relevance:** the whole class of "container-internal" failures is invisible
to host-only smoke probes → see suggested smoke addition below.

## New findings for QA (with evidence)

### FINDING-D1 — ACT-Q2 CONFIRMED: 413 body-limit response uses `INTERNAL_ERROR` code

072 ACT-Q2 suspected this; now confirmed with a live probe against the compose
stack (~1.1 MB body, limit is 1 MB):

```
curl -s -X POST http://localhost:8080/api/v1/auth/register \
  -H 'Content-Type: application/json' --data-binary @big.json
→ HTTP 413
  {"error":{"code":"INTERNAL_ERROR","message":"Request Entity Too Large"}}
```

`errorHandler` maps unknown `*fiber.Error` codes to `INTERNAL_ERROR`. Status
413 is correct; the **code value is misleading** (claims server fault for a
client size violation). 022 §3 error-codes table does not define a 413 row →
per **DEC-H03 this is a PO decision** (e.g. new `REQUEST_TOO_LARGE` code vs
reuse `VALIDATION_ERROR`), NOT something Dev will change unilaterally.

**QA action:** file in 043 defect report with the repro above; tag as
`needs-po-decision`.

### FINDING-D2 — envelope shape spot-check (info, no action)

400/401/404/409 paths all return the exact `{"error":{code,message,details?}}`
envelope (smoke + unit tests assert exact shape, not substrings).

## Suggested additions to QA's open 072 items

- **ACT-Q4 (extend):** besides 405/404/gRPC checks, have `smoke.sh` assert
  container health: `docker inspect -f '{{.State.Health.Status}}' sevensolution-api-1`
  = `healthy`. That is the only probe class that would have caught the tcp4
  regression above.
- **Optional:** slow-client check for the new ReadTimeout — open a socket,
  send a partial request, expect the connection closed within ~10s. Low
  priority; documents ACT-D1 behavior end-to-end.
- **ACT-Q1/Q3 unchanged** — Mongo integration suite and the AC-001f race
  scenario remain QA's biggest coverage gap (mongodb adapter 0% by design).

## Decisions requested from PO

| ID | Question | Why PO |
|----|----------|--------|
| DEC-D01 | 413 error code: add `REQUEST_TOO_LARGE` (new code) vs map to `VALIDATION_ERROR`? | Error envelope is the locked 022 contract (DEC-H03) |

## Verification commands (re-run baseline)

```bash
make test               # go test -race -cover ./...
bash scripts/smoke.sh   # 13 checks vs live stack
docker compose ps       # api healthy
```

## Parking Lot

| # | Item | Owner | When |
|---|------|-------|------|
| 1 | `golangci-lint` + `.golangci.yml` + Makefile `lint` target (ACT-D3) | Dev | post-handoff, optional |
| 2 | Per-request context timeout middleware (ACT-D4) | — | closed as wontfix unless QA finds a gap |
| 3 | Task 2 — Lottery Search System design proposal | PO | next session with user |

---

> **Related:** [[000_spec_index]], [[072_MM_handoff_polish]], [[022_API_specification]], [[041_test_plan]], [[015_definition_of_done]]
