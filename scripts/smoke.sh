#!/usr/bin/env bash
# Smoke test for the User Management API.
# Runs against HOST (default http://localhost:8080). Pass/fail per step,
# exits non-zero on any failure. Requires: curl, grep, sed.
set -uo pipefail

HOST="${HOST:-http://localhost:8080}"
API="$HOST/api/v1"
PASS=0
FAIL=0
EMAIL="smoke-$(date +%s)@example.com"
PASSWORD="sm0ke-test-pass"

step() { echo ""; echo "── $1"; }

expect_status() {
  local desc="$1" want="$2" got="$3"
  if [ "$want" = "$got" ]; then
    echo "  ✅ $desc ($got)"
    PASS=$((PASS+1))
  else
    echo "  ❌ $desc: expected $want, got $got"
    FAIL=$((FAIL+1))
  fi
}

step "healthz"
code=$(curl -s -o /dev/null -w '%{http_code}' "$HOST/healthz")
expect_status "GET /healthz" 200 "$code"

step "register (expect 201)"
resp=$(curl -s -w '\n%{http_code}' -X POST "$API/auth/register" \
  -H 'Content-Type: application/json' \
  -d "{\"name\":\"Smoke Test\",\"email\":\"$EMAIL\",\"password\":\"$PASSWORD\"}")
code=$(echo "$resp" | tail -1); body=$(echo "$resp" | head -1)
expect_status "POST /auth/register" 201 "$code"
USER_ID=$(echo "$body" | grep -o '"id":"[^"]*"' | head -1 | sed 's/"id":"//;s/"//')
if [ -z "$USER_ID" ]; then echo "  ❌ could not extract user id"; FAIL=$((FAIL+1)); fi
echo "  user id: $USER_ID"
if echo "$body" | grep -q "password"; then
  echo "  ❌ response leaks password material"; FAIL=$((FAIL+1))
else
  echo "  ✅ no password material in response"; PASS=$((PASS+1))
fi

step "register duplicate (expect 409)"
code=$(curl -s -o /dev/null -w '%{http_code}' -X POST "$API/auth/register" \
  -H 'Content-Type: application/json' \
  -d "{\"name\":\"Smoke Two\",\"email\":\"$EMAIL\",\"password\":\"$PASSWORD\"}")
expect_status "POST /auth/register (dup)" 409 "$code"

step "login (expect 200 + token)"
resp=$(curl -s -w '\n%{http_code}' -X POST "$API/auth/login" \
  -H 'Content-Type: application/json' \
  -d "{\"email\":\"$EMAIL\",\"password\":\"$PASSWORD\"}")
code=$(echo "$resp" | tail -1); body=$(echo "$resp" | head -1)
expect_status "POST /auth/login" 200 "$code"
TOKEN=$(echo "$body" | grep -o '"token":"[^"]*"' | sed 's/"token":"//;s/"//')
if [ -z "$TOKEN" ]; then echo "  ❌ could not extract token"; FAIL=$((FAIL+1)); fi

step "login wrong password (expect 401)"
code=$(curl -s -o /dev/null -w '%{http_code}' -X POST "$API/auth/login" \
  -H 'Content-Type: application/json' \
  -d "{\"email\":\"$EMAIL\",\"password\":\"wrong-password\"}")
expect_status "POST /auth/login (wrong pw)" 401 "$code"

step "list users with token (expect 200)"
code=$(curl -s -o /dev/null -w '%{http_code}' "$API/users" -H "Authorization: Bearer $TOKEN")
expect_status "GET /users" 200 "$code"

step "list users without token (expect 401)"
code=$(curl -s -o /dev/null -w '%{http_code}' "$API/users")
expect_status "GET /users (no token)" 401 "$code"

step "get user by id (expect 200)"
code=$(curl -s -o /dev/null -w '%{http_code}' "$API/users/$USER_ID" -H "Authorization: Bearer $TOKEN")
expect_status "GET /users/$USER_ID" 200 "$code"

step "update name (expect 200)"
code=$(curl -s -o /dev/null -w '%{http_code}' -X PUT "$API/users/$USER_ID" \
  -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
  -d '{"name":"Smoke Updated"}')
expect_status "PUT /users/$USER_ID" 200 "$code"

step "update invalid email (expect 400)"
code=$(curl -s -o /dev/null -w '%{http_code}' -X PUT "$API/users/$USER_ID" \
  -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
  -d '{"email":"not-an-email"}')
expect_status "PUT /users/$USER_ID (bad email)" 400 "$code"

step "delete user (expect 204)"
code=$(curl -s -o /dev/null -w '%{http_code}' -X DELETE "$API/users/$USER_ID" \
  -H "Authorization: Bearer $TOKEN")
expect_status "DELETE /users/$USER_ID" 204 "$code"

step "get deleted user (expect 404)"
code=$(curl -s -o /dev/null -w '%{http_code}' "$API/users/$USER_ID" -H "Authorization: Bearer $TOKEN")
expect_status "GET /users/$USER_ID (deleted)" 404 "$code"

# ── ACT-Q4 additions (072 polish handoff) ──────────────────────────────

step "404 envelope on unknown route"
resp=$(curl -s -w '\n%{http_code}' "$HOST/api/v1/does-not-exist")
code=$(echo "$resp" | tail -1); body=$(echo "$resp" | head -1)
expect_status "GET unknown route" 404 "$code"
if echo "$body" | grep -q '"error":{"code":"NOT_FOUND"'; then
  echo "  ✅ 404 uses API error envelope"; PASS=$((PASS+1))
else
  echo "  ❌ 404 body not the API envelope: $body"; FAIL=$((FAIL+1))
fi

step "405 envelope on wrong verb (ACT-Q4)"
resp=$(curl -s -w '\n%{http_code}' -X PATCH "$API/users/$USER_ID" -H "Authorization: Bearer $TOKEN")
code=$(echo "$resp" | tail -1); body=$(echo "$resp" | head -1)
expect_status "PATCH /users/{id} (no such verb)" 405 "$code"
if echo "$body" | grep -q '"error":{"code":"METHOD_NOT_ALLOWED"'; then
  echo "  ✅ 405 uses API error envelope"; PASS=$((PASS+1))
else
  echo "  ❌ 405 body not the API envelope: $body"; FAIL=$((FAIL+1))
fi

step "container health (catches host-invisible failures, e.g. tcp4)"
CONTAINER="${CONTAINER:-sevensolution-api-1}"
if command -v docker >/dev/null 2>&1 && docker inspect "$CONTAINER" >/dev/null 2>&1; then
  health=$(docker inspect -f '{{.State.Health.Status}}' "$CONTAINER" 2>/dev/null)
  if [ "$health" = "healthy" ]; then
    echo "  ✅ container $CONTAINER is healthy"; PASS=$((PASS+1))
  else
    echo "  ❌ container $CONTAINER health: ${health:-<none>}"; FAIL=$((FAIL+1))
  fi
else
  echo "  ⚠️  docker/container not reachable — skipped (set CONTAINER= to override)"
fi

step "gRPC checks (ACT-Q4; needs grpcurl)"
GRPC_HOST="${GRPC_HOST:-localhost:50051}"
SVC="userservice.v1.UserService"
if command -v grpcurl >/dev/null 2>&1; then
  # register a dedicated live user for gRPC (the main smoke user was deleted above)
  GEMAIL="grpc-smoke-$(date +%s)@example.com"
  GID=$(curl -s -X POST "$API/auth/register" -H 'Content-Type: application/json' \
    -d "{\"name\":\"gRPC Smoke\",\"email\":\"$GEMAIL\",\"password\":\"grpc-sm0ke-pass\"}" | grep -o '"id":"[^"]*"' | head -1 | sed 's/"id":"//;s/"//')
  GTOKEN=$(curl -s -X POST "$API/auth/login" -H 'Content-Type: application/json' \
    -d "{\"email\":\"$GEMAIL\",\"password\":\"grpc-sm0ke-pass\"}" | grep -o '"token":"[^"]*"' | head -1 | sed 's/"token":"//;s/"//')

  out=$(grpcurl -plaintext -d "{\"id\":\"$GID\"}" "$GRPC_HOST" "$SVC/GetUser" 2>&1)
  if echo "$out" | grep -q 'Unauthenticated'; then
    echo "  ✅ gRPC without metadata → Unauthenticated"; PASS=$((PASS+1))
  else
    echo "  ❌ gRPC without metadata: $out"; FAIL=$((FAIL+1))
  fi

  out=$(grpcurl -plaintext -H "authorization: Bearer $GTOKEN" -d "{\"id\":\"$GID\"}" "$GRPC_HOST" "$SVC/GetUser" 2>&1)
  if echo "$out" | grep -q '"email"'; then
    echo "  ✅ gRPC GetUser with token returns the user"; PASS=$((PASS+1))
  else
    echo "  ❌ gRPC GetUser with token: $out"; FAIL=$((FAIL+1))
  fi

  # cleanup the dedicated gRPC user
  curl -s -o /dev/null -X DELETE "$API/users/$GID" -H "Authorization: Bearer $TOKEN"
else
  echo "  ⚠️  grpcurl not on PATH — skipped"
fi

echo ""
echo "═══════════════════════════════════"
echo "SMOKE RESULT: $PASS passed, $FAIL failed"
echo "═══════════════════════════════════"
[ "$FAIL" -eq 0 ]
