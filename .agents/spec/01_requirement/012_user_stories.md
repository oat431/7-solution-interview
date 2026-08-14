---
document_type: User Stories
version: "1.0"
status: Draft
author: "PO"
created: "2026-08-15"
last_updated: "2026-08-15"
project_name: "7-Solutions Backend Interview Challenge — Task 1: User Management API"
project_id: "SS-INT-001"
ba_owner: "PO"
po_owner: "Candidate"
classification: "Internal"
tags: [user-stories, invest, golang, interview]
standard_ref:
  - SWEBOK v4 — Requirements
---

# User Stories — User Management API

> **Project:** SS-INT-001 | **Version:** 1.0 | **Status:** Draft
> Persona: an API consumer (curl / grpcurl / reviewer). No UI.

## Epics

| Epic | Name | Stories | Priority |
|------|------|---------|----------|
| E-01 | Authentication | US-001, US-002 | 🔴 |
| E-02 | User CRUD (REST) | US-003..US-007 | 🔴 |
| E-03 | Observability & Concurrency | US-008, US-009 | 🔴 |
| E-04 | Bonus: gRPC + Ops | US-010, US-011 | 🟡 |

---

### Epic E-01: Authentication

#### US-001: Register
**As a** new API consumer, **I want** to register with name, email and password, **so that** a user account exists for me.
- **AC:** See 013 → FR-001
- **Priority:** 🔴 | **Points:** 3 | **Epic:** E-01

#### US-002: Login
**As a** registered user, **I want** to exchange my credentials for a JWT, **so that** I can call protected endpoints.
- **AC:** See 013 → FR-002
- **Priority:** 🔴 | **Points:** 3 | **Epic:** E-01

### Epic E-02: User CRUD (REST)

#### US-003: Create User (protected)
**As an** authenticated caller, **I want** to create users via `POST /users`, **so that** user creation is available as a management operation.
- **AC:** See 013 → FR-003 | **Priority:** 🔴 | **Points:** 2 | **Epic:** E-02

#### US-004: Get User by ID
**As an** authenticated caller, **I want** to fetch one user by ID, **so that** I can read specific account data.
- **AC:** See 013 → FR-004 | **Priority:** 🔴 | **Points:** 2 | **Epic:** E-02

#### US-005: List Users
**As an** authenticated caller, **I want** to list all users, **so that** I can see the full dataset.
- **AC:** See 013 → FR-005 | **Priority:** 🔴 | **Points:** 1 | **Epic:** E-02

#### US-006: Update Name/Email
**As an** authenticated caller, **I want** to update a user's name and/or email, **so that** records stay current.
- **AC:** See 013 → FR-006 | **Priority:** 🔴 | **Points:** 3 | **Epic:** E-02

#### US-007: Delete User
**As an** authenticated caller, **I want** to delete a user, **so that** stale accounts can be removed.
- **AC:** See 013 → FR-007 | **Priority:** 🔴 | **Points:** 1 | **Epic:** E-02

### Epic E-03: Observability & Concurrency

#### US-008: Request Logging Middleware
**As a** reviewer, **I want** every HTTP request logged with method, path and execution time, **so that** middleware behavior is verifiable.
- **AC:** See 013 → FR-008 | **Priority:** 🔴 | **Points:** 2 | **Epic:** E-03

#### US-009: User Count Worker
**As an** operator, **I want** a background goroutine to log total user count every 10 seconds, **so that** dataset size is visible in logs.
- **AC:** See 013 → FR-009 | **Priority:** 🔴 | **Points:** 2 | **Epic:** E-03

### Epic E-04: Bonus — gRPC + Ops

#### US-010: gRPC CreateUser + GetUser
**As an** API consumer, **I want** gRPC endpoints for CreateUser and GetUser protected by token metadata, **so that** the polyglot bonus is demonstrated.
- **AC:** See 013 → FR-010 | **Priority:** 🟡 | **Points:** 5 | **Epic:** E-04

#### US-011: Containerization & Graceful Shutdown
**As a** reviewer, **I want** `docker compose up` to run API + MongoDB, with clean shutdown on SIGINT/SIGTERM, **so that** setup friction is zero.
- **AC:** See 013 → FR-011 | **Priority:** 🟡 | **Points:** 3 | **Epic:** E-04

---

## INVEST Check (whole backlog)

| Criterion | Check |
|-----------|-------|
| Independent | ✅ each story is a discrete endpoint/behavior |
| Negotiable | ✅ details refined in 013 |
| Valuable | ✅ each maps to a challenge requirement or bonus |
| Estimable | ✅ points above |
| Small | ✅ max 5 points (US-010) |
| Testable | ✅ every story has GWT ACs in 013 |

**Total:** 11 stories, 27 points. **Sequencing:** E-01 → E-02 → E-03 → E-04 (gRPC last per R-01 in 011).
