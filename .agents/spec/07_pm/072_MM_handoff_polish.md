---
document_type: Meeting Minutes (Handoff)
version: "1.0"
status: Approved
author: "PO"
created: "2026-08-15"
last_updated: "2026-08-15"
project_name: "7-Solutions Backend Interview Challenge — Task 1: User Management API"
project_id: "SS-INT-001"
classification: "Internal"
tags: [handoff, meeting-minutes, polish, dev, qa, security]
standard_ref:
  - SWEBOK v4 — Cross-Cutting (Project Management)
---

# Meeting Minutes — Polish Handoff (PO → Dev / QA / Security)

> **Meeting ID:** MTG-H01 | **Type:** Handoff
> **Facilitator/Scribe:** PO | **Date:** 2026-08-15

## Attendees

| Role | Persona | Status |
|------|---------|--------|
| Product Owner | 🎯 PO | ✅ Present (facilitator) |
| Backend Dev | 👨‍💻 Full-Stack | ✅ Present — takes code polish |
| QA Engineer | 🧪 QA | ✅ Present — takes test coverage |
| Security Engineer | 🔒 SecEng | ✅ Present — takes security review |

## Context (shared baseline for all three)

Task 1 is **implemented, verified, and committed** at `F:\interview\SevenSolution`
(module `github.com/oat431/7-solution-interview`). Latest commit `2cac38e` (Fiber v3 migration).

Verified live before handoff: `go test -race` 7/7 packages green (coverage: application 91.1%, domain 96.6%, httpapi 91.3%, grpcapi 88.2%, auth 87%, worker 83.3%; mongodb adapter 0% by design), `go vet` clean, smoke **13/13** vs real Mongo, grpcurl gRPC verified, graceful shutdown exit 0.

**Contracts that are NOT re-openable** (product intent, spec at `.agents\spec\`):
- 022 API contract: routes, status codes, error envelope `{error:{code,message,details}}`
- 013 Acceptance Criteria (46 ACs) and 011 evaluation-criteria mapping
- Assumptions A1–A12 and ADR-01..07 (025) — incl. Fiber v3, custom JWT middleware (one shared verifier with gRPC, NOT `jwtware`), hexagonal layout, HS256 + bcrypt, no roles (A8)
- Challenge requirements: std `testing` package, official Mongo driver, logging middleware, 10s worker

**Boundary:** you own HOW (quality), not WHAT (behavior). Any proposed contract change goes back to PO first.

## Decisions Made (recorded, not re-openable)

| ID | Decision | Rationale |
|----|----------|-----------|
| DEC-H01 | Polish passes run in parallel: Dev (code), QA (coverage), Sec (review) | Independent workstreams, single codebase — coordinate via git commits, one persona commits at a time |
| DEC-H02 | All polish must keep `go test -race ./...` green and the smoke script 13/13 | Non-negotiable quality floor (041/015) |
| DEC-H03 | Findings that would change the API contract, auth model, or architecture → report to PO, do not implement | Product owns WHAT/WHY |
| DEC-H04 | `govulncheck` findings → fix or document with rationale in README assumptions | Interview reviewers read the assumptions table |

## Action Items — Dev 👨‍💻

| ID | Action | Notes |
|----|--------|-------|
| ACT-D1 | Fiber server timeouts: set `ReadTimeout`/`WriteTimeout`/`IdleTimeout` in `fiber.Config` (currently only BodyLimit; the old net/http server had ReadHeaderTimeout) | config in `httpapi/app.go`; env-configurable or sensible defaults |
| ACT-D2 | Mongo driver connection pool: explicit `SetMaxPoolSize`/`SetMinPoolSize` (+ server selection timeout) in `connectMongo` | currently driver defaults |
| ACT-D3 | Optional: `golangci-lint` (checklist §Project Setup) with `.golangci.yml` + Makefile `lint` target | keep the enabled set lean; must be clean before commit |
| ACT-D4 | Optional: per-request context timeout middleware | only if it doesn't complicate shutdown |
| ACT-D5 | Re-read 025 package tree after changes; update docs if structure changes | docs-first convention |

## Action Items — QA 🧪

| ID | Action | Notes |
|----|--------|-------|
| ACT-Q1 | **Biggest gap:** Mongo adapter integration suite (0% coverage by design) — env-gated or `//go:build integration` tests against real Mongo: all 7 repo ops, unique-index duplicate → `ErrEmailExists` mapping, invalid ID → `ErrInvalidID` | `internal/infrastructure/mongodb/` |
| ACT-Q2 | Verify **>1MB body** behavior: Fiber returns 413; our `errorHandler` currently maps unknown `*fiber.Error` → 500 — likely needs a 413 → `VALIDATION_ERROR` mapping + test | suspected bug, confirm first |
| ACT-Q3 | Race test for AC-001f: N goroutines registering the same email → exactly one 201 (service level, fake repo cannot race — use the mongo integration suite or a mutex-free scenario note) | document the gap honestly if not testable at unit level |
| ACT-Q4 | Extend `scripts/smoke.sh`: 405 check, 404 envelope check, gRPC grpcurl checks (Create/Get + Unauthenticated) | currently REST-only |
| ACT-Q5 | AC traceability walk: 013's 46 ACs → test coverage map; flag uncovered ones | 041 table is the starting point |
| ACT-Q6 | Deliverables: additions to 042 test cases, or 043 defect report if Q2 confirmed | |

## Action Items — Security 🔒

| ID | Action | Notes |
|----|--------|-------|
| ACT-S1 | Run `govulncheck ./...`; fix or document each finding | DEC-H04 |
| ACT-S2 | Login timing side-channel: when email not found, bcrypt compare is skipped — consider a dummy compare to flatten timing | low severity for interview; document either way |
| ACT-S3 | Confirm no secrets in Docker layers (.env not copied — verify image contents), non-root user, `-ldflags` strip | Dockerfile already non-root |
| ACT-S4 | Confirm JWT hardening: alg pinning, secret length fail-fast, TTL sanity, token never in logs | tests exist — verify claims on a fresh read |
| ACT-S5 | Document production gaps in README assumptions: no rate limiting, no role model (A8), plain HTTP locally (TLS terminates at reverse proxy in real deployment), compose Mongo without auth | reviewers read this table |
| ACT-S6 | Deliverable: 061 security test report notes (or append to README assumptions) | |

## Parking Lot (deferred, not for this polish pass)

| # | Item | Owner | When |
|---|------|-------|------|
| 1 | Task 2 — Lottery Search System design proposal | PO | after polish pass (next session with user) |
| 2 | Push to GitHub (`oat431/7-solution-interview`) | User decision | whenever user says |
| 3 | Rate limiting + roles/ownership model | PO | post-interview hardening, only if user wants |
| 4 | OTel / Swagger / helmet / CORS | — | skipped by tier decision (see Fiber checklist scoping) |

---

> **Related:** [[000_spec_index]], [[011_business_objective]], [[013_acceptance_criteria]], [[022_API_specification]], [[025_software_architecture_document]], [[041_test_plan]], [[015_definition_of_done]]
