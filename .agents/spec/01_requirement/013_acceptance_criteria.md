---
document_type: Acceptance Criteria (ATDD/BDD)
version: "1.0"
status: Draft
author: "PO"
created: "2026-08-15"
last_updated: "2026-08-15"
project_name: "7-Solutions Backend Interview Challenge — Task 1: User Management API"
project_id: "SS-INT-001"
ba_owner: "PO"
qa_lead: "Candidate"
classification: "Internal"
tags: [acceptance-criteria, given-when-then, golang, jwt]
standard_ref:
  - SWEBOK v4 — Requirements
  - ISO/IEC/IEEE 29119 — Software Testing
---

# Acceptance Criteria — User Management API

> **Project:** SS-INT-001 | **Version:** 1.0 | **Status:** Draft
> Each criterion is unit-testable (see 041). HTTP codes reference the contract in 022.

## FR-001: Register (US-001) — `POST /api/v1/auth/register`

| AC ID | Scenario | Given | When | Then |
|-------|---------|-------|------|------|
| AC-001a | Happy path | valid name/email/password | POST register | 201; body = `{id,name,email,createdAt}`; **no password hash**; user persisted in Mongo with bcrypt hash ≠ plaintext |
| AC-001b | Invalid email | email = `"abc"` | POST register | 400 `VALIDATION_ERROR` with field detail on `email` |
| AC-001c | Missing required field | name empty | POST register | 400 `VALIDATION_ERROR` with field detail on `name` |
| AC-001d | Short password | password < 8 chars | POST register | 400 `VALIDATION_ERROR` with field detail on `password` |
| AC-001e | Duplicate email | email already exists | POST register | 409 `EMAIL_ALREADY_EXISTS` |
| AC-001f | Unique race | two concurrent registers, same email | both complete | exactly one 201; the other 409 (unique index is source of truth) |

## FR-002: Login (US-002) — `POST /api/v1/auth/login`

| AC ID | Scenario | Given | When | Then |
|-------|---------|-------|------|------|
| AC-002a | Happy path | registered user | POST login with correct credentials | 200 `{token, tokenType:"Bearer", expiresIn, user}` |
| AC-002b | Token usable | token from 002a | GET `/api/v1/users` with `Authorization: Bearer <token>` | 200 (not 401) |
| AC-002c | Wrong password | registered user | POST login, wrong password | 401 `INVALID_CREDENTIALS` — same body as wrong-email case |
| AC-002d | Unknown email | unregistered email | POST login | 401 `INVALID_CREDENTIALS` (no user enumeration) |
| AC-002e | Token shape | token decoded | inspect claims | HS256; `sub` = user id; `email` claim; `exp` ≤ now + TTL; `iat`, `iss` present |

## FR-003: Create User (US-003) — `POST /api/v1/users` (JWT)

| AC ID | Scenario | Given | When | Then |
|-------|---------|-------|------|------|
| AC-003a | Happy path | valid JWT + valid body | POST /users | 201, same body shape as 001a |
| AC-003b | No token | no Authorization header | POST /users | 401 `UNAUTHORIZED` |
| AC-003c | Validation | invalid email | POST /users | 400 `VALIDATION_ERROR` (same rules as register) |

## FR-004: Get by ID (US-004) — `GET /api/v1/users/{id}` (JWT)

| AC ID | Scenario | Given | When | Then |
|-------|---------|-------|------|------|
| AC-004a | Happy path | existing user id | GET /users/{id} | 200 `{id,name,email,createdAt}` |
| AC-004b | Not found | random valid ObjectId | GET /users/{id} | 404 `USER_NOT_FOUND` |
| AC-004c | Malformed id | id = `"zzz"` | GET /users/{id} | 400 `INVALID_ID` |
| AC-004d | No token | no header | GET /users/{id} | 401 `UNAUTHORIZED` |

## FR-005: List (US-005) — `GET /api/v1/users` (JWT)

| AC ID | Scenario | Given | When | Then |
|-------|---------|-------|------|------|
| AC-005a | Happy path | N users exist | GET /users | 200 `{data:[...], meta:{count:N}}`; every element free of password fields |
| AC-005b | Empty | no users | GET /users | 200 `{data:[], meta:{count:0}}` |
| AC-005c | No token | no header | GET /users | 401 `UNAUTHORIZED` |

## FR-006: Update Name/Email (US-006) — `PUT /api/v1/users/{id}` (JWT)

| AC ID | Scenario | Given | When | Then |
|-------|---------|-------|------|------|
| AC-006a | Update name only | existing user | PUT `{"name":"New"}` | 200 updated user; email unchanged |
| AC-006b | Update email only | existing user, email unused | PUT `{"email":"x@y.z"}` | 200 updated user |
| AC-006c | Empty body | existing user | PUT `{}` | 400 `VALIDATION_ERROR` (at least one of name/email) |
| AC-006d | Invalid email | existing user | PUT `{"email":"bad"}` | 400 `VALIDATION_ERROR` |
| AC-006e | Email conflict | email owned by another user | PUT `{"email": existing}` | 409 `EMAIL_ALREADY_EXISTS` |
| AC-006f | Not found | unknown id | PUT | 404 `USER_NOT_FOUND` |
| AC-006g | Password immutability | body includes `password` | PUT | field rejected/ignored — password cannot be changed via this endpoint |

## FR-007: Delete (US-007) — `DELETE /api/v1/users/{id}` (JWT)

| AC ID | Scenario | Given | When | Then |
|-------|---------|-------|------|------|
| AC-007a | Happy path | existing user | DELETE /users/{id} | 204; subsequent GET → 404 |
| AC-007b | Not found | unknown id | DELETE | 404 `USER_NOT_FOUND` |
| AC-007c | No token | no header | DELETE | 401 `UNAUTHORIZED` |

## FR-008: Logging Middleware (US-008)

| AC ID | Scenario | Given | When | Then |
|-------|---------|-------|------|------|
| AC-008a | Log shape | any request | handled | one structured log line: `method`, `path`, `status`, `duration` |
| AC-008b | Covers all routes | any endpoint | handled | middleware wraps every registered route (incl. 404s) |
| AC-008c | No secrets | request with Authorization header | logged | log line contains **no** token/password material |

## FR-009: User Count Worker (US-009)

| AC ID | Scenario | Given | When | Then |
|-------|---------|-------|------|------|
| AC-009a | Periodic log | N users, worker running | 10s elapses | log line with `total_users: N` (query hit Mongo, not a cache) |
| AC-009b | Graceful stop | SIGINT received | shutdown starts | worker goroutine exits cleanly (no goroutine leak, no panic) |
| AC-009c | Testability | interval injectable | tick with short interval | worker observable in tests without sleeping 10s |

## FR-010: gRPC (US-010)

| AC ID | Scenario | Given | When | Then |
|-------|---------|-------|------|------|
| AC-010a | CreateUser | valid metadata token + valid request | grpc CreateUser | `User{id,name,email,createdAt}` returned; persisted in Mongo |
| AC-010b | GetUser | valid token + existing id | grpc GetUser | `User` returned; unknown id → `codes.NotFound` |
| AC-010c | Missing token | no `authorization` metadata | any grpc call | `codes.Unauthenticated` |
| AC-010d | Invalid token | garbage token | any grpc call | `codes.Unauthenticated` |
| AC-010e | Shared core | REST + gRPC both deployed | create via REST, get via gRPC | same user visible (one application layer) |

## FR-011: Docker & Graceful Shutdown (US-011)

| AC ID | Scenario | Given | When | Then |
|-------|---------|-------|------|------|
| AC-011a | Compose up | clean machine with Docker | `docker compose up -d` | API + Mongo healthy (healthchecks pass) |
| AC-011b | Full loop in compose | compose stack up | register → login → get via curl | works end-to-end |
| AC-011c | Graceful shutdown | server running | SIGINT/SIGTERM | in-flight requests complete; HTTP + gRPC servers + worker shut down; exit code 0 |
| AC-011d | Startup validation | `JWT_SECRET` unset or < 32 bytes | start API | process refuses to start with clear error |

## AC Summary

| Requirement | ACs | 🔴 | Notes |
|-------------|-----|-----|-------|
| FR-001 Register | 6 | 6 | |
| FR-002 Login | 5 | 5 | |
| FR-003 Create | 3 | 3 | |
| FR-004 Get | 4 | 4 | |
| FR-005 List | 3 | 3 | |
| FR-006 Update | 7 | 7 | |
| FR-007 Delete | 3 | 3 | |
| FR-008 Logging | 3 | 3 | |
| FR-009 Worker | 3 | 3 | |
| FR-010 gRPC | 5 | 5 | 🟡 bonus |
| FR-011 Ops | 4 | 4 | 🟡 bonus |
| **Total** | **46** | | Traceable to test plan 041 |

---

> **Related:** [[012_user_stories]], [[022_API_specification]], [[041_test_plan]]
