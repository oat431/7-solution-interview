# User Management API

RESTful user management API in Go with MongoDB persistence and JWT authentication, built for the 7-Solutions backend challenge — with hexagonal architecture, gRPC, Docker, input validation and graceful shutdown.

## Features

- ✅ User model: id, name, email (unique), password (bcrypt), createdAt
- ✅ Register + login → JWT (HS256), middleware-protected endpoints
- ✅ CRUD: create, get by id, list, update name/email, delete
- ✅ Official MongoDB Go driver (v2); unique email index created at startup
- ✅ Logging middleware: method, path, status, duration (+ request ID) via structured `slog`
- ✅ Background goroutine: logs total user count every 10s
- ✅ Unit tests with std `testing` + hand-written fakes (`-race` clean, core coverage 83–100%)
- 🎁 gRPC server (`CreateUser`, `GetUser`) secured by JWT metadata
- 🎁 Hexagonal architecture (ports & adapters)
- 🎁 Docker + docker-compose, input validation, graceful shutdown

## Stack

Go 1.25 · Fiber v3 (REST) · MongoDB (official driver v2) · `golang-jwt/v5` (HS256) · `golang.org/x/crypto/bcrypt` · gRPC (bonus)

## Setup & Run

### Option A — Docker Compose (recommended)

```bash
cp .env.example .env      # set a real JWT_SECRET (>= 32 chars), e.g. `openssl rand -base64 48`
docker compose up --build
```

Starts the API (`:8080`), gRPC (`:50051`) and MongoDB (`:27017`). Healthcheck at `GET /healthz`.

### Option B — Manual

```bash
# 1. MongoDB running on localhost:27017 (or set MONGO_URI)
# 2. Configure
cp .env.example .env
export $(cat .env | xargs)     # bash

# 3. Run
make run

# 4. Tests
make test                     # go test -race -cover ./...
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

1. **Get a token** — `POST /api/v1/auth/login` (register first). Response: `{"token": "...", "tokenType": "Bearer", "expiresIn": 3600, "user": {...}}`.
2. **Use it** — send `Authorization: Bearer <token>` on protected endpoints.
3. **Inspect it** — paste into [jwt.io](https://jwt.io) or:

```bash
cut -d'.' -f2 <<< "$TOKEN" | base64 -d 2>/dev/null
# {"sub":"665f1c2d...","email":"ada@example.com","iat":...,"exp":...,"iss":"sevensolutions-user-api"}
```

4. **Rules** — HS256 only (other algs rejected); expiry from `TOKEN_TTL`; missing/malformed/expired/wrong-alg ⇒ `401 UNAUTHORIZED`.

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
cmd/api/main.go              # composition root: config, wiring, servers, worker, shutdown
internal/
  domain/                    # User entity, validation rules, domain errors (stdlib-only)
  application/               # use cases + ports (interfaces), no framework imports
  infrastructure/
    mongodb/                 # MongoDB adapter (official driver v2)
    auth/                    # bcrypt + JWT (HS256) adapters
    httpapi/                 # REST adapter (Fiber v3): app, handlers, middleware, error envelope
    grpcapi/                 # gRPC adapter (bonus)
    config/                  # env config with fail-fast validation
    logger/                  # slog JSON setup
  worker/                    # 10s user-count logger
testutil/                    # hand-written fake repository + hasher (no mock lib)
proto/ + gen/                # protobuf definitions + generated code
scripts/smoke.sh             # end-to-end smoke test
```

## Testing

```bash
make test    # go test -race -cover ./...
```

Coverage across the core (domain/application/http/grpc/worker) is 83–100%. The MongoDB adapter itself is excluded from unit coverage by design: it is a thin driver wrapper verified by the smoke test against a real Mongo, while all business logic is tested against the in-memory fake behind the repository port. HTTP tests run in-process through Fiber's `app.Test` — no server needed.

## Assumptions & Design Decisions

| # | Decision | Why |
|---|----------|-----|
| 1 | Hexagonal (ports & adapters) layout | Bonus item + testability + decouples domain from Mongo |
| 2 | Fiber v3 for the REST adapter (fasthttp engine); gRPC stays native grpc-go | Preferred framework; structured middleware; custom JWT middleware (not `jwtware`) so REST and gRPC share one verifier |
| 3 | Mongo behind a `UserRepository` interface; tests use a hand-written in-memory fake (no mock library) | Challenge's "mock MongoDB where appropriate"; adapter stays thin |
| 4 | bcrypt (default cost) + HS256 JWT | Industry defaults matching the challenge constraints |
| 5 | Register (public) and Create User (JWT-protected) both exist, sharing one use case | The challenge lists both operations separately |
| 6 | No role model — any authenticated caller may update/delete any user | None specified; noted as production hardening |
| 7 | `PUT /users/{id}` is a partial update of name/email; password not mutable there | Matches "update a user's name or email" |
| 8 | List returns all users (no pagination) with `count` meta | Matches "list all users" |
| 9 | Emails normalized to lowercase at the service boundary | Consistent uniqueness + login behavior |
| 10 | gRPC on a separate port with a JWT metadata interceptor | "Optionally secure with token metadata" → secured |
| 11 | Unique email index created programmatically at startup (idempotent) | Race-safe duplicate rejection; no manual setup step |
| 12 | ID validity checked in the domain layer | Identical behavior across repository implementations |

## License

Interview submission — no license.
