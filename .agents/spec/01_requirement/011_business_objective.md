---
document_type: Business Objectives
version: "1.0"
status: Draft
author: "PO"
created: "2026-08-15"
last_updated: "2026-08-15"
project_name: "7-Solutions Backend Interview Challenge — Task 1: User Management API"
project_id: "SS-INT-001"
sponsor: "Candidate (Interview Applicant)"
ba_owner: "PO"
classification: "Internal"
tags: [business-objectives, interview, golang, evaluation-criteria]
standard_ref:
  - BABOK v3 — Strategy Analysis
---

# Business Objectives — User Management API

> **Project:** 7-Solutions Backend Interview Challenge (Task 1)
> **Version:** 1.0 | **Status:** Draft
> **Last Updated:** 2026-08-15

---

## 1. Executive Summary

| Field | Detail |
|-------|--------|
| Purpose | Win the interview: deliver a Go + MongoDB + JWT user management API that scores maximum points against the published evaluation criteria, with all 6 bonus items implemented |
| Expected Outcome | A git repo the reviewer can `docker compose up`, exercise via curl + grpcurl, and read as clean, idiomatic, testable Go |
| Success Definition | Every requirement demonstrably working + every bonus item present + zero reviewer friction |
| Strategic Theme | Interview performance (short, decisive delivery window) |
| Target Completion | 2026-08-20 |

## 2. Strategic Alignment — Evaluation Criteria → Objectives

The challenge's evaluation criteria ARE our strategy. Each criterion maps to an objective:

| Evaluation Criterion (from README) | Objective | Weight |
|------------------------------------|-----------|--------|
| Code quality, structure, readability | OBJ-03 | High |
| Correctness & completeness of REST API | OBJ-01 | High |
| Security & implementation of JWT | OBJ-02 | High |
| Proper usage & abstraction of MongoDB | OBJ-03 | High |
| Test coverage & effective mocking | OBJ-04 | High |
| Idiomatic Go usage | OBJ-03 | Medium |
| Bonus implementations (gRPC, Docker, validation, architecture) | OBJ-05 | Bonus |

### Objective Register

| ID | Objective | Priority |
|----|-----------|----------|
| OBJ-01 | Complete & correct REST API covering all 7 challenge requirements | 🔴 |
| OBJ-02 | Secure JWT auth (HS256, middleware, no secret leaks, correct error semantics) | 🔴 |
| OBJ-03 | Clean, idiomatic, hexagonal Go with Mongo behind an interface | 🔴 |
| OBJ-04 | ≥80% unit test coverage using std `testing` with hand-written fakes | 🔴 |
| OBJ-05 | All 6 bonus items: Docker/compose, interfaces, validation, graceful shutdown, gRPC, hexagonal | 🟡 (bonus) |

## 3. Objective Cards

### OBJ-01: Complete & Correct REST API

| Field | Detail |
|-------|--------|
| Statement | All challenge operations work end-to-end against MongoDB |
| Measurable | 7/7 requirements verified with curl samples in README: model, register, login→JWT, create, get-by-id, list, update, delete, logging middleware, 10s worker |
| Achievable | Scope is small and well-defined |
| Relevant | Maps 1:1 to "Correctness and completeness" criterion |
| Time-Bound | 2026-08-20 |
| Baseline | 0 endpoints | Target | 10 endpoints + 1 worker |

### OBJ-02: Secure JWT

| Field | Detail |
|-------|--------|
| Statement | HS256-signed JWT with middleware enforcement and no security anti-patterns |
| Measurable | 401 on missing/malformed/expired/wrong-alg token; `JWT_SECRET` ≥32B enforced at startup; password hash never returned; bcrypt used; no user enumeration on login (single 401 message) |
| Relevant | "Security and implementation of JWT" criterion |
| Time-Bound | 2026-08-20 |

### OBJ-03: Clean Idiomatic Hexagonal Go

| Field | Detail |
|-------|--------|
| Statement | Domain/application/infrastructure separation; Mongo behind a repository port; Fiber v3 driving adapter |
| Measurable | Package tree matches 025; domain package has zero external imports; only purpose-built direct deps (mongo-driver, jwt, bcrypt, fiber, grpc, protobuf) |
| Relevant | "Code quality/structure", "Idiomatic Go", "Abstraction of MongoDB" |

### OBJ-04: Test Coverage & Mocking

| Field | Detail |
|-------|--------|
| Statement | Unit tests with `testing` package; Mongo interactions behind hand-written fakes |
| Measurable | `go test ./... -race` green; ≥80% coverage on domain+application+http; worker interval injectable for testability |
| Relevant | "Test coverage and effective mocking" criterion |

### OBJ-05: All 6 Bonus Items

| Field | Detail |
|-------|--------|
| Statement | Docker + compose, Mongo interfaces, validation, graceful shutdown, gRPC server, hexagonal layout — all present and working |
| Measurable | 6/6 checklist verified (see DoD 015); `docker compose up` runs API + Mongo; grpcurl sample succeeds |
| Relevant | Explicit bonus section of README |

## 4. KPI Framework (interview-adapted: binary gates, not dashboards)

| KPI | Gate | Status |
|-----|------|--------|
| KPI-01 | All 7 challenge requirements demoable via curl | ☐ |
| KPI-02 | JWT: 401 matrix (missing/malformed/expired/wrong-alg) all correct | ☐ |
| KPI-03 | `go vet` + `gofmt` clean | ☐ |
| KPI-04 | `go test ./... -race` green, coverage ≥80% | ☐ |
| KPI-05 | 6/6 bonuses verified | ☐ |
| KPI-06 | Reviewer friction = 0 (compose up → curl login → grpcurl) | ☐ |

## 5. Risks to Objectives

| ID | Risk | Probability | Impact | Mitigation |
|----|------|------------|--------|-----------|
| R-01 | gRPC scope creep eats time from required items | Medium | High | Required scope first (OBJ-01..04 green), gRPC last (OBJ-05 tail) |
| R-02 | Over-engineering: enterprise ceremony instead of lean interview code | Medium | Medium | Lean internal docs; code stays minimal; no framework sprawl |
| R-03 | Reviewer environment variance (Go version, Mongo image) | Medium | Medium | Docker compose pins versions; README documents manual path |
| R-04 | Duplicate-email race (unique index vs pre-check) | Low | High | Both: pre-check for friendly 409 + unique index as the source of truth |

## 6. Decision Log (scope choices made in spec, revisited at implementation)

| # | Decision | Owner |
|---|----------|-------|
| D-01 | Full-send bonus scope (all 6) | Candidate |
| D-02 | Hexagonal package layout (see 025) | PO + Candidate |
| D-03 | Register + Create share one use case (A7) | PO |
| D-04 | gRPC = same application layer, second adapter | PO |

---

> **Related:** [[000_spec_index]], [[012_user_stories]], [[025_software_architecture_document]]
