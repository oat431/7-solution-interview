---
document_type: README Developer Guide
version: "1.0"
status: Draft
author: "PO"
created: "2026-08-15"
last_updated: "2026-08-15"
project_name: "7-Solutions Backend Interview Challenge — Task 1: User Management API"
project_id: "SS-INT-001"
classification: "Internal"
tags: [readme, developer-guide, deliverable]
---

# README Developer Guide (deliverable content)

> **Project:** SS-INT-001 | **Version:** 1.0 | **Status:** Draft
> ⚠️ At implementation time, the `## README (repo)` section below is copied verbatim to the repository root as `README.md`. It is one of the four required deliverables (README, JWT guide, sample requests, assumptions).

## Deliverable mapping (challenge → this doc)

| Challenge deliverable | Location in repo README |
|-----------------------|------------------------|
| README with setup & execution | §Setup & Run |
| Guide: how to generate & use JWT tokens | §JWT Guide |
| Sample API requests and responses | §Sample Requests |
| Documentation of assumptions / design decisions | §Assumptions & Design Decisions |

---

## README (repo)

# User Management API

RESTful user management API in Go with MongoDB persistence and JWT authentication, built for the 7-Solutions backend challenge — with hexagonal architecture, gRPC, Docker, input validation and graceful shutdown.

## About This Submission

This repository covers **both challenge tasks** and was produced **spec-first with AI assistance** — AI personas (product owner, dev, QA, security engineer) worked from written specifications under continuous human review and direction. Every significant decision has a recorded rationale, so nothing here is "AI wrote it and we can't explain it."

- **Task 1 — User Management API** (this codebase): the full spec package lives in [`.agents/spec/`](.agents/spec) — business objectives mapped to the evaluation criteria, 46 Given–When–Then acceptance criteria, the API contract, architecture with ADRs, test plan, and the complete review/decision trail (handoff minutes MTG-H01 → MTG-P01).
- **Task 2 — Lottery Search System** (design proposal, no code): the proposal package lives in [`.agents/proposal/`](.agents/proposal) — solution architecture, data structures, storage/algorithms, performance analysis, and ADRs.

## Features

- ✅ User model: id, name, email (unique), password (bcrypt), createdAt
- ✅ Register + login → JWT (HS256), middleware-protected endpoints
- ✅ CRUD: create, get by id, list, update name/email, delete
- ✅ Official MongoDB Go driver; unique email index
- ✅ Logging middleware: method, path, status, duration (structured `slog`)
- ✅ Background goroutine: logs total user count every 10s
- ✅ Unit tests with std `testing` + hand-written fakes (≥80% coverage, `-race` clean)
- 🎁 gRPC server (`CreateUser`, `GetUser`) secured by JWT metadata
- 🎁 Hexagonal architecture (ports & adapters)
- 🎁 Docker + docker-compose, input validation, graceful shutdown

## Stack

Go 1.22+ (Fiber v3) · MongoDB (official driver) · `golang-jwt/v5` (HS256) · `bcrypt` · gRPC

## Setup & Run

### Option A — Docker Compose (recommended)

```bash
docker compose up --build
```

Starts API (:8080), gRPC (:50051), MongoDB (:27017). Healthcheck: `GET /healthz`.

### Option B — Manual

```bash
# 1. MongoDB running on localhost:27017 (or set MONGO_URI)
# 2. Configure
cp .env.example .env      # then set JWT_SECRET (>= 32 chars)
export $(cat .env | xargs)

# 3. Build & run
make run

# 4. (optional) regenerate protobuf code
make gen-proto

# 5. Tests
make test        # go test -race -cover ./...
```

### Configuration (env)

| Var | Default | Notes |
|-----|---------|-------|
| `MONGO_URI` | `mongodb://localhost:27017` | |
| `DB_NAME` | `userdb` | |
| `JWT_SECRET` | — (required) | ≥32 bytes; startup fails otherwise |
| `TOKEN_TTL` | `1h` | JWT expiry |
| `HTTP_PORT` | `8080` | |
| `GRPC_PORT` | `50051` | |
| `WORKER_INTERVAL` | `10s` | user-count logger tick |

## JWT Guide

1. **Get a token** — `POST /api/v1/auth/login` (or register first). Response: `{"token": "...", "tokenType": "Bearer", "expiresIn": 3600, ...}`.
2. **Use it** — add `Authorization: Bearer <token>` to any protected endpoint.
3. **Inspect it** — `jwt.io` or:

```bash
cut -d'.' -f2 <<< "$TOKEN" | base64 -d 2>/dev/null
# {"sub":"665f1c2d...","email":"ada@example.com","iat":...,"exp":...,"iss":"sevensolutions-user-api"}
```

4. **Rules** — HS256 only; expiry from `TOKEN_TTL`; missing/malformed/expired/wrong-alg ⇒ `401 UNAUTHORIZED`.

## Sample Requests

```bash
BASE=localhost:8080/api/v1

# Register (public) → 201
curl -s -X POST $BASE/auth/register -H 'Content-Type: application/json' \
  -d '{"name":"Ada Lovelace","email":"ada@example.com","password":"s3cret-pass"}'
# → {"id":"665f1c2d3e4f5a6b7c8d9e0f","name":"Ada Lovelace","email":"ada@example.com","createdAt":"2026-08-15T10:00:00Z"}

# Login (public) → 200 {token,...}
curl -s -X POST $BASE/auth/login -H 'Content-Type: application/json' \
  -d '{"email":"ada@example.com","password":"s3cret-pass"}'

# List (JWT) → 200 {data:[...],meta:{count:N}}
curl -s $BASE/users -H "Authorization: Bearer $TOKEN"

# Get by id (JWT) → 200 / 404
curl -s $BASE/users/665f1c2d3e4f5a6b7c8d9e0f -H "Authorization: Bearer $TOKEN"

# Update name/email (JWT) → 200
curl -s -X PUT $BASE/users/665f1c2d3e4f5a6b7c8d9e0f -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"name":"Ada Byron"}'

# Delete (JWT) → 204
curl -s -X DELETE $BASE/users/665f1c2d3e4f5a6b7c8d9e0f -H "Authorization: Bearer $TOKEN"

# gRPC (bonus)
grpcurl -plaintext -H "authorization: Bearer $TOKEN" \
  -d '{"name":"Grace Hopper","email":"grace@example.com","password":"c0bol-rul3z"}' \
  localhost:50051 userservice.v1.UserService/CreateUser
```

## Project Structure

```
cmd/api/main.go              # composition root
internal/
  domain/                    # entity + errors (pure, stdlib-only)
  application/               # use cases + ports (interfaces)
  infrastructure/            # Mongo / JWT / bcrypt / Fiber HTTP / gRPC adapters
  worker/                    # 10s user-count logger
testutil/                    # hand-written fake repository
proto/ + gen/                # protobuf definitions + generated code
```

## Assumptions & Design Decisions

| # | Decision | Why |
|---|----------|-----|
| 1 | Hexagonal (ports & adapters) layout | bonus + testability + decouples domain from Mongo |
| 2 | Fiber v3 for the REST adapter (fasthttp engine); gRPC stays native grpc-go | Preferred framework; structured middleware; custom JWT middleware (not `jwtware`) so REST and gRPC share one verifier |
| 3 | Mongo behind `UserRepository` interface; tests use a hand-written in-memory fake (no mock lib) | challenge's "mock MongoDB where appropriate"; adapter stays thin |
| 4 | bcrypt (cost 10) + HS256 JWT | industry defaults matching challenge constraints |
| 5 | Register (public) and Create User (JWT) both exist, sharing one use case | challenge lists both operations |
| 6 | No role model — any authenticated caller may update/delete any user | none specified; noted as production hardening |
| 7 | `PUT /users/{id}` is partial update of name/email; password immutable via this endpoint | matches "update a user's name or email" |
| 8 | List returns all users (no pagination) with `count` meta | matches "list all users" |
| 9 | gRPC on separate port with JWT metadata interceptor | "optionally secure with token metadata" → secured |
| 10 | Unique email index created programmatically at startup | idempotent; race-safe duplicate rejection |

## Testing

```bash
make test   # go test -race -cover ./...
```

Coverage ≥80% across domain/application/http/grpc/worker. Mongo adapter excluded (integration-only) — exercised via `scripts/smoke.sh` against compose.

---

## Implementation notes (not part of repo README)

- Generate protobuf: `make gen-proto` (requires `protoc`, `protoc-gen-go`, `protoc-gen-go-grpc`); commit generated code to keep the reviewer build simple.
- Smoke script `scripts/smoke.sh` = the 041 §6 manual verification, kept in repo as a bonus artifact.
- Keep direct external deps ≤6: `mongo-driver`, `jwt/v5`, `x/crypto`, `grpc`, `protobuf`, `genproto`.

---

> **Related:** [[022_API_specification]], [[025_software_architecture_document]], [[015_definition_of_done]]
