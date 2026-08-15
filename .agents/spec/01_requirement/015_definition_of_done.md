---
document_type: Definition of Done (DoD)
version: "1.0"
status: Draft
author: "PO"
created: "2026-08-15"
last_updated: "2026-08-15"
project_name: "7-Solutions Backend Interview Challenge — Task 1: User Management API"
project_id: "SS-INT-001"
tags: [dod, definition-of-done, interview]
standard_ref:
  - Scrum Guide
  - Agile Practice Guide (PMI)
---

# Definition of Done — User Management API

> **Project:** SS-INT-001 | **Version:** 1.0 | **Status:** Draft
> The "release" here = **submission to the interviewer**. The DoD is the final checklist before pushing the repo.

## 1. DoD — Story (any US-xxx)

| # | Criterion |
|---|-----------|
| 1 | Implementation matches 022/023/025 |
| 2 | All ACs for the story (013) pass as unit tests |
| 3 | `gofmt` + `go vet` clean on touched files |
| 4 | No secrets, debug prints, or TODO-FIXME noise committed |

## 2. DoD — Submission (the gate that matters)

### Functional

| # | Criterion | Verified |
|---|-----------|----------|
| 1 | All 7 challenge requirements demoable (register, login→JWT, create, get, list, update, delete, logging middleware, 10s worker) | ✅ |
| 2 | Bonus 1 — Dockerfile + compose up with healthchecks | ✅ |
| 3 | Bonus 2 — Mongo behind Go interface (repository port) | ✅ |
| 4 | Bonus 3 — input validation (required fields, email format, password rules) | ✅ |
| 5 | Bonus 4 — graceful shutdown via `context.Context` + signals | ✅ |
| 6 | Bonus 5 — gRPC `.proto` + server + token metadata interceptor | ✅ |
| 7 | Bonus 6 — hexagonal layers (domain/application/infrastructure) | ✅ |

### Quality

| # | Criterion | Verified |
|---|-----------|----------|
| 8 | `go test ./... -race` green | ✅ |
| 9 | Aggregate coverage ≥80% (domain/application/http/grpc/worker) | ✅ 83–100% |
| 10 | `go vet ./...` + `gofmt -l .` empty | ✅ |
| 11 | README (031) present with: setup, JWT guide, sample requests/responses, assumptions & design decisions | ✅ |
| 12 | Smoke script (041 §6) passes against compose stack | ✅ 13/13 |
| 13 | `.env.example` only — no real secrets; `.gitignore` covers `.env` | ✅ |
| 14 | Repo pushed with clean history; no build artifacts / generated code committed unless deliberate | ☐ push pending (remote TBD) |

## 3. Quality Gates (automated where possible)

| Metric | Threshold | Tool |
|--------|-----------|------|
| Unit coverage (aggregate) | ≥80% | `go test -cover` |
| Integration suite (build tag) | green vs compose Mongo | `go test -tags integration -race ./internal/infrastructure/mongodb/` |
| Race detector | 0 races | `go test -race` |
| Vet / format | 0 issues | `go vet`, `gofmt` |
| Dependency count | ≤8 direct external modules, purpose-built only (mongo-driver, jwt, bcrypt, fiber, grpc, protobuf) | `go mod graph` (README states rationale) |

## 4. Exceptions & Notes

- **Mongo adapter coverage excluded** from the ≥80% aggregate (integration-only) — documented in 041 §2, defensible to reviewer.
- Any exception to this DoD must be listed in README "Assumptions & Design Decisions" with rationale — the reviewer reads that section.

---

> **Related:** [[041_test_plan]], [[013_acceptance_criteria]], [[031_README_developer_guide]]
