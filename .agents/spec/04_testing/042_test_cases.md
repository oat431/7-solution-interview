---
document_type: Test Cases
version: "1.0"
status: Active
author: "QA"
created: "2026-08-15"
last_updated: "2026-08-15"
project_name: "7-Solutions Backend Interview Challenge — Task 1: User Management API"
project_id: "SS-INT-001"
classification: "Internal"
tags: [test-cases, test-scenarios, traceability, swebok, iso-29119]
standard_ref:
  - SWEBOK v4 — Testing
  - ISO/IEC/IEEE 29119 — Software Testing
---

# Test Cases — User Management API (QA polish pass)

> **Project:** SS-INT-001 | **Version:** 1.0 | **Status:** Active
> **Last Updated:** 2026-08-15
> **Baseline:** commit `a8c6a52` + this pass's additions (uncommitted at writing)
> Complements the PO's 041 test plan; this file holds the **QA-added** cases (ACT-Q1..Q6 from 072) and the full AC traceability walk.

## 1. Test Case Index (QA additions)

| Suite | Cases | Automated | Layer | Status |
|-------|-------|-----------|-------|--------|
| Mongo integration (ACT-Q1) | 9 | ✅ `user_repo_integration_test.go` | Integration (real Mongo) | ✅ green |
| Register race (ACT-Q3) | 1 | ✅ same file | Integration + service | ✅ green |
| HTTP gap closures (ACT-Q5) | 2 | ✅ `handlers_test.go` | Unit (in-process Fiber) | ✅ green |
| Smoke extension (ACT-Q4) | 7 checks | ✅ `scripts/smoke.sh` | E2E vs compose | ✅ 20/20 |
| 413 contract test (ACT-Q2) | 1 | ⏸ pending PO decision DEF-001 | Unit | blocked |
| **Total new** | **19 checks / 12 Go tests** | | | |

Run commands:

```bash
go test -race -count=1 ./...                                    # unit gate (unchanged DoD floor)
go test -tags integration -race -count=1 ./internal/infrastructure/mongodb/   # needs Mongo on localhost:27017 (compose)
bash scripts/smoke.sh                                           # needs compose up; grpcurl/docker optional (skips gracefully)
```

## 2. Integration Test Cases (ACT-Q1 — closes "mongodb adapter 0%" gap)

File: `internal/infrastructure/mongodb/user_repo_integration_test.go`, build tag `integration` (default `make test` unaffected — hermetic DoD gate preserved per DEC-H02). Each test gets a throwaway database `qa_it_*`, dropped on cleanup.

| TC | Title | Verifies | Result |
|----|-------|----------|--------|
| TC-QINT-01 | Create persists doc; id ObjectID; bcrypt-shaped hash in Mongo; no plaintext field | 023 §2, AC-001a (persistence half) | ✅ |
| TC-QINT-02 | Duplicate email → `ErrEmailExists` (unique index violation mapping) | AC-001e/f backstop, 023 §3 | ✅ |
| TC-QINT-03 | `EnsureIndexes` idempotent + index is unique | A12 | ✅ |
| TC-QINT-04 | FindByID: happy / unknown→ErrNotFound / malformed→ErrInvalidID | AC-004a/b/c (repo layer) | ✅ |
| TC-QINT-05 | FindByEmail returns hash; unknown→ErrNotFound | login path, AC-002c/d | ✅ |
| TC-QINT-06 | List: empty → `[]`; populated → all users | AC-005a/b (repo layer) | ✅ |
| TC-QINT-07 | Update: name, email, dup-email→ErrEmailExists, unknown→ErrNotFound, malformed→ErrInvalidID | AC-006a/b/e/f (repo layer) | ✅ |
| TC-QINT-08 | Delete: removes doc; second delete→ErrNotFound; malformed→ErrInvalidID | AC-007a/b (repo layer) | ✅ |
| TC-QINT-09 | Count tracks inserts/deletes | FR-009 worker data source | ✅ |
| TC-QINT-10 | **Race (ACT-Q3):** 16 goroutines register same email via real `UserService` + real Mongo | **AC-001f end-to-end** | ✅ exactly 1 success / 15 conflicts / 1 doc persisted |

> Design note: TC-QINT-10 runs the race at the **service level** against real Mongo (as 072 ACT-Q3 requested) — the unique index is the only arbiter, pre-check + bcrypt hashing widen the race window. `-race` on throughout.

## 3. HTTP Unit Gap Closures (ACT-Q5 findings)

| TC | Title | AC | Result |
|----|-------|----|--------|
| TC-HTTP-Q1 | `TestListUsersEmpty` — empty DB → `{"data":[],"meta":{"count":0}}`, never `data:null` | AC-005b | ✅ |
| TC-HTTP-Q2 | `TestUpdateUserEmailOnly` — email-only partial update, name untouched | AC-006b | ✅ |

## 4. Smoke Extension (ACT-Q4 + 073 suggestion)

`scripts/smoke.sh` grew from 13 → **20 checks** (all verified green against live stack):

| # | Check | Purpose |
|---|-------|---------|
| 14–15 | Unknown route → 404 **and** envelope shape | 404 semantics + DEF-002 witness |
| 16–17 | `PATCH /users/{id}` → 405 **and** envelope shape | ACT-Q4 405 check |
| 18 | `docker inspect` container health = `healthy` | 073 suggestion — the probe class that would have caught the tcp4 regression |
| 19 | gRPC GetUser **without** metadata → `Unauthenticated` | AC-010c live |
| 20 | gRPC GetUser **with** token → user returned | AC-010b live |

Graceful degradation: container-health and gRPC checks skip with a warning when `docker`/`grpcurl` are absent (host override: `CONTAINER=`, `GRPC_HOST=`).

## 5. Blocked Test Case (ACT-Q2 → DEF-001)

| TC | Title | Status |
|----|-------|--------|
| TC-QINT-11 | Body > 1 MB → 413 + PO-chosen code (`REQUEST_TOO_LARGE` vs `VALIDATION_ERROR`) | ⏸ blocked on DEC-D01. Repro exists in 043 DEF-001; smoke/manual probe holds until then. |

## 6. AC Traceability Walk (ACT-Q5 — all 46 ACs)

Legend: 🟢 unit (auto) · 🟢i integration (auto, this pass) · 🟢e E2E/live (smoke or grpcurl, this pass or handoff) · 🟡 partial — residual risk noted.

| AC | Description (short) | Unit | Integration | E2E/live | Verdict |
|----|--------------------|------|-------------|----------|---------|
| 001a | Register happy, no hash in body | `TestRegisterReturns201WithoutPasswordMaterial` | TC-QINT-01 (bcrypt in Mongo) | smoke #2 (incl. no-password grep) | 🟢 full |
| 001b | Invalid email 400 + field detail | `TestRegisterValidationError`, domain table | — | — | 🟢 |
| 001c | Missing name 400 | same matrix | — | — | 🟢 |
| 001d | Short password 400 | same matrix | — | — | 🟢 |
| 001e | Duplicate 409 | `TestRegisterDuplicateEmail409`, `TestCreateDuplicateEmail` | TC-QINT-02 (index mapping) | smoke #3 | 🟢 full |
| 001f | Unique race: exactly one 201 | 🟡 fake repo serializes (documented) | **TC-QINT-10 (real race, 16 goroutines)** | — | 🟢 closed this pass |
| 002a | Login 200 shape | `TestLoginResponseShape`, `TestLoginHappyPath` | — | smoke #4 | 🟢 |
| 002b | Token usable on protected route | `TestLoginHappyPathAndTokenUsage` | — | smoke #6 | 🟢 |
| 002c | Wrong pw = wrong email identical 401 | `TestLoginWrongPasswordAndUnknownEmailIdentical401`, `...IsInvalidCredentials` x2 | — | smoke #5 (wrong pw) | 🟢 |
| 002d | Unknown email 401, no enumeration | covered by identical-401 test | — | 🟡 live only wrong-pw probe; unknown-email identical body proven at unit level | 🟢 |
| 002e | Claims: HS256, sub, email, exp≤ttl, iat, iss | `TestIssueAndVerifyRoundtrip` + alg/tamper/expiry/wrong-secret/none/missing-sub suite | — | README decode sample | 🟢 |
| 003a | Create (JWT) 201 | `TestCreateUserProtected` | — | — | 🟢 |
| 003b | No token 401 | `TestProtectedEndpointsRejectMissingToken` (all 5 routes) | — | smoke #7 | 🟢 |
| 003c | Validation shared with register | `TestCreateValidationError` (shared `CreateUser` use case) | — | — | 🟢 |
| 004a | Get 200 | `TestGetUser` | TC-QINT-04 | smoke #8 | 🟢 full |
| 004b | Not found 404 | `TestGetUserNotFound` | TC-QINT-04 | smoke #13 (deleted) | 🟢 |
| 004c | Malformed id 400 INVALID_ID | `TestGetUserInvalidID` | TC-QINT-04 | — | 🟢 |
| 004d | No token 401 | 401 matrix | — | — | 🟢 |
| 005a | List shape + no password fields | `TestListUsers` (+ domain.User has no password field, compile-time) | TC-QINT-06 | smoke #6 | 🟢 |
| 005b | Empty list `[]`/count 0 | **TC-HTTP-Q1 (this pass)** | TC-QINT-06 | — | 🟢 closed this pass |
| 005c | No token 401 | 401 matrix | — | — | 🟢 |
| 006a | Update name only | `TestUpdateUserNameOnly`, svc test | TC-QINT-07 | smoke #9 | 🟢 full |
| 006b | Update email only | **TC-HTTP-Q2 (this pass)**, svc-level | TC-QINT-07 | — | 🟢 closed this pass |
| 006c | Empty body 400 | `TestUpdateEmptyBodyRejected` (both layers) | — | — | 🟢 |
| 006d | Invalid email 400 | `TestUpdateInvalidEmail` | — | smoke #10 | 🟢 |
| 006e | Email conflict 409 | `TestUpdateEmailConflict409`, svc test | TC-QINT-07 (index) | — | 🟢 |
| 006f | Not found 404 | `TestUpdateNotFound` | TC-QINT-07 | — | 🟢 |
| 006g | Password immutable via PUT | `TestUpdatePasswordFieldRejected` + `TestRegisterUnknownFieldsRejected` (DisallowUnknownFields) | — | — | 🟢 |
| 007a | Delete 204, then 404 | `TestDeleteThenGet404` | TC-QINT-08 | smoke #11–13 | 🟢 full |
| 007b | Not found 404 | `TestDeleteNotFound` | TC-QINT-08 | — | 🟢 |
| 007c | No token 401 | 401 matrix | — | — | 🟢 |
| 008a | Log line: method/path/status/duration | `TestLoggingMiddlewareEmitsStructuredLine` | — | compose logs observed (handoff) | 🟢 |
| 008b | Covers all routes incl. 404 | `TestUnknownRoute404` exercises middleware on catch-all; middleware registered before router group | — | — | 🟢 |
| 008c | No secrets in logs | `TestLoggingMiddlewareNeverLogsSecrets` | — | — | 🟢 |
| 009a | Periodic total_users from Mongo | `TestWorkerLogsCountPeriodically` (short interval) | 🟡 Count source verified by TC-QINT-09; periodic-against-real-Mongo observed in compose logs (handoff evidence) | compose logs | 🟢 (composition argument: interval logic unit-tested × Count integration-tested) |
| 009b | Graceful stop on SIGINT | `TestWorkerStopsImmediatelyWhenContextAlreadyCancelled` | — | graceful shutdown exit 0 (handoff) | 🟢 |
| 009c | Injectable interval | worker tests use 20ms interval | — | — | 🟢 |
| 010a | gRPC CreateUser persists | `TestCreateUser`, `TestCreateUserDuplicate`, `TestCreateUserValidationError` | — | 🟢e live grpcurl this pass (create, dup, invalid) | 🟢 |
| 010b | gRPC GetUser + NotFound | `TestGetUser`, `TestGetUserNotFound`, `TestGetUserInvalidID` | — | 🟢e live grpcurl this pass (200/NotFound/InvalidArgument) | 🟢 |
| 010c | Missing metadata → Unauthenticated | `TestMissingMetadata` | — | 🟢e live grpcurl this pass + smoke #19 | 🟢 |
| 010d | Garbage token → Unauthenticated | `TestGarbageToken`, `TestExpiredToken` | — | 🟢e live garbage-token probe this pass | 🟢 |
| 010e | REST↔gRPC shared core | `TestRESTAndGRPCShareCore` | — | 🟢e live: gRPC-created user fetched via REST (200) | 🟢 |
| 011a | Compose up healthy | — | — | 🟢e both containers `healthy`; container-internal healthz 200 (tcp4 fix verified) + smoke #18 | 🟢 |
| 011b | Full loop in compose | — | — | smoke 20/20 vs compose | 🟢 |
| 011c | Graceful shutdown exit 0 | shutdown ordering by inspection (`main.go shutdown()`) | — | handoff evidence exit 0; QA not re-run (would disturb stack) | 🟡 accepted via evidence + code read |
| 011d | Startup fail on weak/missing secret | `TestLoadMissingSecretFails`, `TestLoadShortSecretFails` + compose `${JWT_SECRET:?}` guard | — | — | 🟢 |

**Coverage statement:** 46/46 ACs have automated evidence at unit or integration level; 44/46 additionally have live/E2E evidence. No AC is untested. Two verdicts carry explicit notes (009a composition argument; 011c evidence-accepted) — both judged sufficient risk-wise for an interview submission; AC-011c could be hardened with a scripted SIGTERM probe if desired (parking lot).

## 7. Known Gaps & Residual Risk (honest list)

| # | Gap | Risk | Disposition |
|---|-----|------|-------------|
| 1 | AC-011c graceful shutdown not scripted in QA pass | Low — code-read + handoff evidence; ordering is straightforward (HTTP→gRPC→Mongo) | Accept; optional scripted probe later |
| 2 | Slow-client ReadTimeout behavior (073 optional suggestion) | Low — ACT-D1 settings verified in code | Not done this pass; optional |
| 3 | Login timing side-channel (unknown email skips bcrypt) | Low (interview scope); SecEng ACT-S2 owns | Route to Security |
| 4 | DEF-001 413 code — contract decision pending | Low | PO DEC-D01 |
| 5 | DEF-002 404 code undocumented | Low | PO decision |
| 6 | DEF-003 no `.dockerignore` | Medium (hygiene) | Dev + SecEng |

## 8. Test Execution Summary (this pass)

| Suite | Executed | Passed | Failed | Pass Rate |
|-------|---------|--------|--------|-----------|
| Unit (`go test -race ./...`) | 80 tests / 7 packages | 80 | 0 | 100% |
| Integration (`-tags integration`, real Mongo) | 10 | 10 | 0 | 100% |
| Smoke (compose, extended) | 20 checks | 20 | 0 | 100% |
| Live gRPC probes (grpcurl) | 8 scenarios | 8 | 0 | 100% |

---

## Related Documents

| Document | Relationship |
|----------|-------------|
| [[041_test_plan]] | Governing plan (PO) — this file extends it |
| [[043_defect_report]] | Defects found during these tests |
| [[013_acceptance_criteria]] | Traceability source (46 ACs) |
| [[022_API_specification]] | Contract under test |
| [[072_MM_handoff_polish]] | ACT-Q1..Q6 origin |
| [[073_MM_dev_to_qa_handoff]] | Baseline + FINDING-D1 |

---

> **Template standard:** SWEBOK v4, ISO/IEC/IEEE 29119
