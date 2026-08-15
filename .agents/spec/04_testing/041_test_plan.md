---
document_type: Test Plan
version: "1.0"
status: Draft
author: "PO"
created: "2026-08-15"
last_updated: "2026-08-15"
project_name: "7-Solutions Backend Interview Challenge — Task 1: User Management API"
project_id: "SS-INT-001"
classification: "Internal"
tags: [test-plan, unit-testing, golang, fakes]
standard_ref:
  - SWEBOK v4 — Testing
  - ISO/IEC/IEEE 29119 — Software Testing
---

# Test Plan — User Management API

> **Project:** SS-INT-001 | **Version:** 1.0 | **Status:** Draft
> Challenge constraint: **standard `testing` package**, mock MongoDB interactions *where appropriate*.

## 1. Scope

| In Scope | Out of Scope |
|----------|-------------|
| Unit tests (domain, application, HTTP handlers, gRPC interceptor, worker) | Load/perf testing |
| Test doubles for MongoDB + token/hash ports where needed | — |
| Build-tagged integration suite vs real Mongo (`go test -tags integration ./internal/infrastructure/mongodb/`) | — |
| Race detector + coverage gates | E2E browser testing (no UI) |

## 2. Strategy

| Level | Approach | Target |
|-------|----------|--------|
| Domain | Table-driven pure tests (`domain.NewUser` validation matrix, errors) | ≥90% |
| Application | Fake repository + real bcrypt/JWT libs with test secret; all use cases happy + error paths | ≥85% |
| HTTP | Fiber in-process tests (`app.Test` + `httptest.NewRequest`) against fake repo: status codes, envelopes, auth 401 matrix, logging middleware capture, 404/405 semantics | ≥85% |
| gRPC | Server + fake repo, interceptor auth matrix (`Unauthenticated`), `context`-based calls | ≥80% |
| Worker | Injected short interval (e.g. 20ms), fake repo counter, ctx cancel → goroutine exits | ≥80% |
| **Overall** | `go test ./... -race` green; `-cover` ≥ **80%** aggregate | ≥80% |

### Mocking decision (ADR-03)

Hand-written fake in `testutil/fake_repo.go` implementing `ports.UserRepository` — an in-memory map with duplicate-email simulation. **No third-party mock library.** The Mongo adapter itself stays thin and is excluded from unit coverage (integration-only, §6); this is the honest reading of "mock MongoDB interactions where appropriate."

## 3. Test Case Index (traces to 013)

| TC | AC | Description |
|----|----|-------------|
| TC-001 | 001a | Register happy path (201, no hash in body, bcrypt hash in fake repo) |
| TC-002 | 001b–d | Register validation matrix (bad email / empty name / short pw) |
| TC-003 | 001e–f | Duplicate email → ErrEmailExists → 409 |
| TC-004 | 002a–b | Login → valid token → protected endpoint 200 |
| TC-005 | 002c–d | Wrong password / unknown email → identical 401 |
| TC-006 | 002e | Claims decode: sub/email/exp/iat/iss, alg=HS256 |
| TC-007 | 003a–c | Create user with/without token, validation |
| TC-008 | 004a–d | Get by id: 200 / 404 / 400 bad id / 401 |
| TC-009 | 005a–c | List: populated / empty / 401 |
| TC-010 | 006a–g | Update: name-only, email-only, empty body, bad email, conflict, 404, password-rejected |
| TC-011 | 007a–c | Delete: 204→404, unknown id, 401 |
| TC-012 | 008a–c | Logging middleware: shape, coverage, no secrets |
| TC-013 | 009a–c | Worker: count logged, ctx cancel, short interval |
| TC-014 | 010a–e | gRPC: Create/Get, metadata auth matrix, REST↔gRPC shared core |
| TC-015 | 011c–d | Graceful shutdown ordering; startup fail on weak/missing secret |

## 4. Environments

| Env | How | Notes |
|-----|-----|-------|
| Unit | `go test ./...` | no external deps needed (fakes) |
| Local integration | `docker compose up` then curl script | manual verification path (§6) |
| CI (optional) | `go test -race -cover ./...` in GitHub Action | cheap bonus signal, not required |

## 5. Entry / Exit Criteria

| Gate | Criterion |
|------|-----------|
| Entry — code | package compiles; `go vet` clean |
| Exit — unit | all tests green under `-race`; aggregate coverage ≥80% |
| Exit — deliverable | §6 smoke script passes against compose stack; README samples match actual responses |

## 6. Manual Verification (complements unit suite)

Real Mongo + real HTTP smoke, run once before submission:

```bash
docker compose up -d --build
./scripts/smoke.sh        # register → login → get/list/update/delete → expect statuses
grpcurl ... CreateUser    # gRPC happy path + missing-token 401
docker compose logs api   # verify: request logs w/ duration + "total_users=N" every 10s
```

## 7. Risks

| Risk | Mitigation |
|------|-----------|
| bcrypt makes tests slow | use cost 10; keep test password set small; bcrypt calls are ~50ms — acceptable |
| Coverage gamed by adapter exclusion | coverage measured per-package with explicit adapter exclusion documented in README |
| Race in worker tests | ctx-based teardown + `-race` in CI gate |

---

> **Related:** [[013_acceptance_criteria]], [[015_definition_of_done]], [[025_software_architecture_document]]
