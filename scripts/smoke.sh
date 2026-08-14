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

echo ""
echo "═══════════════════════════════════"
echo "SMOKE RESULT: $PASS passed, $FAIL failed"
echo "═══════════════════════════════════"
[ "$FAIL" -eq 0 ]
