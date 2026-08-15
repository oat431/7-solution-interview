---
document_type: Spec Index
version: "1.0"
status: Draft
author: "PO"
created: "2026-08-15"
last_updated: "2026-08-15"
project_name: "7-Solutions Backend Interview Challenge — Task 1: User Management API"
project_id: "SS-INT-001"
classification: "Internal"
tags: [spec-index, interview, golang, mongodb, jwt, hexagonal]
---

# Spec Index — SevenSolution Interview Challenge

> **Scope:** Task 1 (User Management API — Go + MongoDB + JWT) only.
> Task 2 (Lottery Search System, design proposal) gets its own spec package after implementation.
> **Bonus scope:** FULL SEND — Docker/compose, Mongo interface abstraction, validation, graceful shutdown, gRPC, hexagonal architecture.

## Document Map

| # | Document | Path | Purpose |
|---|----------|------|---------|
| 011 | Business Objectives | `01_requirement/011_business_objective.md` | Evaluation criteria → our objectives & scorecard strategy |
| 012 | User Stories | `01_requirement/012_user_stories.md` | INVEST stories (REST, gRPC, worker) |
| 013 | Acceptance Criteria | `01_requirement/013_acceptance_criteria.md` | Given–When–Then per requirement |
| 015 | Definition of Done | `01_requirement/015_definition_of_done.md` | What "submittable" means |
| 022 | API Specification | `02_design/022_API_specification.md` | REST + gRPC contract (the coding contract) |
| 023 | DB Schema | `02_design/023_database_schema_DDL.md` | Mongo collection, indexes |
| 025 | Architecture | `02_design/025_software_architecture_document.md` | Hexagonal layers, ports/adapters, package tree |
| 031 | README Guide | `03_construction/031_README_developer_guide.md` | The deliverable itself: setup, JWT guide, samples, assumptions |
| 041 | Test Plan | `04_testing/041_test_plan.md` | Unit strategy, fakes, coverage targets |
| 042 | Test Cases | `04_testing/042_test_cases.md` | QA-added cases (integration, race, smoke ext.) + 46-AC traceability |
| 043 | Defect Report | `04_testing/043_defect_report.md` | DEF-001..003 (413 code, 404 code, dockerignore) |
| 072 | Polish Handoff | `07_pm/072_MM_handoff_polish.md` | MTG-H01: PO → Dev/QA/Security action items |
| 073 | Dev → QA Handoff | `07_pm/073_MM_dev_to_qa_handoff.md` | MTG-D01: polish complete, defect evidence (413 envelope), QA-relevant fixes |
| 074 | QA Polish Complete | `07_pm/074_MM_qa_polish_complete.md` | MTG-Q01: QA → PO, 46/46 AC traced, 3 defects, 2 PO decisions |

## Suggested Reading Order (for implementation)

1. **011** — what we're optimizing for (evaluation criteria)
2. **025** — architecture & package layout (everything depends on this)
3. **022 + 023** — contracts (REST, gRPC, data)
4. **012 + 013** — what "done per story" means
5. **041 + 015** — quality gates
6. **031** — the README we ship to the reviewer

## Baseline Assumptions (recorded once, referenced everywhere)

| # | Assumption | Rationale |
|---|-----------|-----------|
| A1 | Go 1.22+ (pin latest stable at implementation) | Fiber v3 minimum; modern toolchain |
| A2 | Fiber v3 for the REST adapter (fasthttp engine); gRPC stays native `grpc-go` | User-preferred framework; structured middleware, fast router; see ADR-02 |
| A3 | `go.mongodb.org/mongo-driver` (official driver, v2 line) | Explicitly required by challenge |
| A4 | `golang-jwt/jwt/v5`, HMAC **HS256**, secret from env (`JWT_SECRET`, ≥32 bytes enforced at startup) | Challenge mandates HS256 |
| A5 | `golang.org/x/crypto/bcrypt` (default cost 10) | Industry standard for password hashing |
| A6 | Structured logging via stdlib `log/slog` | Idiomatic, zero-dep, satisfies logging middleware + worker |
| A7 | Registration (`POST /auth/register`, public) and Create User (`POST /users`, JWT-protected) both exist and share the same `CreateUser` use case | Challenge lists both bullets separately; sharing avoids duplicate logic |
| A8 | No role model / no ownership restriction on update & delete — any authenticated caller may operate on any user | Challenge specifies no roles; simplicity chosen, hardening noted as production TODO in 025 |
| A9 | `PUT /users/{id}` is a partial update of `name` and/or `email` only | Matches "Update a user's name or email" |
| A10 | No pagination on list — return all users with `count` meta | Requirement says "List all users" |
| A11 | gRPC on separate port (default 50051), both `CreateUser` and `GetUser` secured via JWT bearer metadata interceptor | "Optionally secure with token metadata" → we secure it |
| A12 | Unique email index created programmatically at startup (idempotent) | Shows driver knowledge; removes manual setup step |

## Status

- [x] Review of challenge README complete
- [x] Bonus scope committed (all 6)
- [x] Spec package written (this folder)
- [x] Implementation — REST API, tests, Docker, gRPC, verified live (compose + smoke 13/13 + grpcurl)
- [ ] Task 2 spec (lottery design proposal)
