---
document_type: API Specification
version: "1.0"
status: Draft
author: "PO"
created: "2026-08-15"
last_updated: "2026-08-15"
project_name: "7-Solutions Backend Interview Challenge — Task 1: User Management API"
project_id: "SS-INT-001"
tech_lead: "Candidate"
classification: "Internal"
tags: [api-specification, rest, grpc, jwt, golang]
standard_ref:
  - SWEBOK v4 — Design
  - OpenAPI Specification 3.0
---

# API Specification — User Management API

> **Project:** SS-INT-001 | **Version:** 1.0 | **Status:** Draft

## 1. API Overview

| Field | Detail |
|-------|--------|
| Base URL (REST) | `http://localhost:8080/api/v1` (env `HTTP_PORT`, default 8080) |
| gRPC endpoint | `localhost:50051` (env `GRPC_PORT`) |
| Protocol | HTTP/1.1 JSON (REST, Fiber v3) + HTTP/2 protobuf (gRPC) |
| Authentication | Bearer JWT, HS256, `Authorization: Bearer <token>` |
| Content-Type | `application/json; charset=utf-8` |
| Versioning | URL path `/api/v1/` |

## 2. Authentication

- **Issuance:** `POST /api/v1/auth/login`
- **Algorithm:** HMAC-SHA256 (HS256) — enforced at verification (any other `alg` ⇒ 401)
- **Secret:** env `JWT_SECRET`; ≥32 bytes enforced at startup (fail-fast)
- **TTL:** env `TOKEN_TTL` (default 1h)
- **Claims:** `sub` (user ID hex string), `email`, `iat`, `exp`, `iss: "sevensolutions-user-api"`

### Headers

```
Authorization: Bearer <jwt>
Content-Type: application/json
```

## 3. Common Response Formats

### Success

```json
{ "id": "665f1c2d3e4f5a6b7c8d9e0f", "name": "Ada Lovelace",
  "email": "ada@example.com", "createdAt": "2026-08-15T10:00:00Z" }
```

> User objects NEVER include password or password hash. List endpoints wrap users as `{"data":[...], "meta":{"count":N}}`.

### Error

```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Request validation failed",
    "details": [{ "field": "email", "message": "must be a valid email address" }]
  }
}
```

### Error Codes

| Code | HTTP | Trigger |
|------|------|---------|
| `VALIDATION_ERROR` | 400 | Bad field / malformed body / malformed ObjectId |
| `INVALID_ID` | 400 | Non-ObjectId `{id}` path param |
| `UNAUTHORIZED` | 401 | Missing / malformed / expired / wrong-alg token |
| `INVALID_CREDENTIALS` | 401 | Login failure (wrong email or password — identical body) |
| `USER_NOT_FOUND` | 404 | Id not in DB |
| `NOT_FOUND` | 404 | Unknown route (catch-all) |
| `METHOD_NOT_ALLOWED` | 405 | Wrong verb on known path |
| `EMAIL_ALREADY_EXISTS` | 409 | Duplicate email (pre-check + unique index) |
| `REQUEST_TOO_LARGE` | 413 | Body exceeds the configured size limit |
| `INTERNAL_ERROR` | 500 | Unhandled server error (message deliberately generic) |

## 4. REST Endpoints

### 4.1 `POST /api/v1/auth/register` — Register (public)

Request:
```json
{ "name": "Ada Lovelace", "email": "ada@example.com", "password": "s3cret-pass" }
```
Response `201`:
```json
{ "id": "665f1c2d3e4f5a6b7c8d9e0f", "name": "Ada Lovelace",
  "email": "ada@example.com", "createdAt": "2026-08-15T10:00:00Z" }
```

Validation rules:

| Field | Rule |
|-------|------|
| `name` | required, trimmed 1–100 chars |
| `email` | required, RFC-5322-ish regex + `net/mail` check |
| `password` | required, 8–72 chars (72 = bcrypt input limit) |

### 4.2 `POST /api/v1/auth/login` — Login (public)

Request: `{ "email": "ada@example.com", "password": "s3cret-pass" }`
Response `200`:
```json
{
  "token": "eyJhbGciOiJIUzI1NiIs...",
  "tokenType": "Bearer",
  "expiresIn": 3600,
  "user": { "id": "665f1c2d...", "name": "Ada Lovelace", "email": "ada@example.com", "createdAt": "2026-08-15T10:00:00Z" }
}
```
Errors: `401 INVALID_CREDENTIALS` for wrong email AND wrong password (identical body — no enumeration). `400 VALIDATION_ERROR` for malformed body.

### 4.3 `POST /api/v1/users` — Create User (JWT)

Body/response identical to register (4.1). Shares `CreateUser` use case. `409` on duplicate email.

### 4.4 `GET /api/v1/users/{id}` — Get User (JWT)

Response `200`: user object (4.1). Errors: `400 INVALID_ID`, `404 USER_NOT_FOUND`, `401`.

### 4.5 `GET /api/v1/users` — List Users (JWT)

Response `200`:
```json
{ "data": [ { "id": "...", "name": "...", "email": "...", "createdAt": "..." } ],
  "meta": { "count": 2 } }
```
No pagination (A10). Empty DB ⇒ `{"data":[],"meta":{"count":0}}`.

### 4.6 `PUT /api/v1/users/{id}` — Update Name/Email (JWT)

Request (partial; ≥1 field):
```json
{ "name": "Ada Byron", "email": "ada.byron@example.com" }
```
- `name`: if present, same rules as register
- `email`: if present, valid + unique (conflict ⇒ 409)
- `password` key present ⇒ rejected (`400 VALIDATION_ERROR`) — password not mutable here
Response `200`: updated user. Errors: `400`, `401`, `404`, `409`.

### 4.7 `DELETE /api/v1/users/{id}` — Delete User (JWT)

Response `204` empty. Errors: `400 INVALID_ID`, `404 USER_NOT_FOUND`, `401`.

## 5. gRPC Service (Bonus)

Proto: `proto/user_service/v1/user_service.proto`

```proto
syntax = "proto3";
package userservice.v1;
option go_package = "github.com/oat431/7-solution-interview/gen/userservice/v1;userservicev1";

message User {
  string id = 1;
  string name = 2;
  string email = 3;
  string created_at = 4;   // RFC3339
}

message CreateUserRequest {
  string name = 1;
  string email = 2;
  string password = 3;
}

message GetUserRequest { string id = 1; }

service UserService {
  rpc CreateUser(CreateUserRequest) returns (User);
  rpc GetUser(GetUserRequest) returns (User);
}
```

| Call | Metadata | Errors |
|------|----------|--------|
| `CreateUser` | `authorization: Bearer <jwt>` | `InvalidArgument` (validation), `AlreadyExists` (dup email), `Unauthenticated` |
| `GetUser` | `authorization: Bearer <jwt>` | `NotFound`, `InvalidArgument` (bad id), `Unauthenticated` |

- Unary interceptor validates JWT from `authorization` metadata (same verifier as REST — one auth core)
- Same application-layer use cases as REST (AC-010e)
- Generated code via `protoc` + `protoc-gen-go` + `protoc-gen-go-grpc` (Makefile target `gen-proto`)

## 6. Operational Behaviors

| Concern | Behavior |
|---------|----------|
| Logging middleware | `slog` line per request: `method`, `path`, `status`, `duration_ms` (+ `request_id` from Fiber requestid) — wraps all routes, never logs secrets |
| User count worker | Ticker, default 10s (`WORKER_INTERVAL` env), logs `total_users=N`; stops on shutdown context |
| Graceful shutdown | `signal.NotifyContext(SIGINT, SIGTERM)` → stop accepting, drain in-flight (Fiber `ShutdownWithContext`, gRPC `GracefulStop`), cancel worker, exit 0 |
| Health | `GET /healthz` (public) → `200 {"status":"ok"}`; compose healthcheck uses it |
| Body limit | Fiber `Config.BodyLimit` 1 MB per request → 413 `REQUEST_TOO_LARGE`; unknown JSON fields rejected (password not smuggled through update) |
| Server timeouts | Fiber: read 10s / write 15s / idle 60s (Fiber default is unlimited; write covers bcrypt + Mongo round-trips) |
| Mongo pool | Driver: max pool 50 / min 5, server selection timeout 5s (bounded connections, fast fail on unreachable Mongo) |

## 7. Sample Requests (also live in README 031)

```bash
curl -s -X POST localhost:8080/api/v1/auth/register -H 'Content-Type: application/json' \
  -d '{"name":"Ada Lovelace","email":"ada@example.com","password":"s3cret-pass"}'

curl -s -X POST localhost:8080/api/v1/auth/login -H 'Content-Type: application/json' \
  -d '{"email":"ada@example.com","password":"s3cret-pass"}'   # → copy token

curl -s localhost:8080/api/v1/users -H "Authorization: Bearer $TOKEN"

grpcurl -plaintext -H 'authorization: Bearer <jwt>' -d '{"name":"Grace Hopper","email":"grace@example.com","password":"c0bol-rul3z"}' \
  localhost:50051 userservice.v1.UserService/CreateUser
```

---

> **Related:** [[023_database_schema_DDL]], [[025_software_architecture_document]], [[013_acceptance_criteria]]
