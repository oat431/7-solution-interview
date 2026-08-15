---
document_type: Security Test Report
version: "1.0"
status: Complete
author: "SecEng"
created: "2026-08-15"
last_updated: "2026-08-15"
project_name: "7-Solutions Backend Interview Challenge — Task 1: User Management API"
project_id: "SS-INT-001"
classification: "Internal"
tags: [security, test-report, govulncheck, jwt, docker, review]
standard_ref:
  - SWEBOK v4 — Software Security
  - CyBOK — Software Security / Authentication, Authorisation & Accountability
  - OWASP ASVS (applied subset)
---

# Security Test Report — User Management API

> **Project:** SS-INT-001 | **Version:** 1.0 | **Status:** Complete
> **Review baseline:** commit `43c3876` (main) — QA → Security handoff ([[075_MM_qa_to_security_handoff]])
> **Reviewer:** SecEng | **Date:** 2026-08-15
> **Deliverable for:** ACT-S6 ([[072_MM_handoff_polish]]), closes ACT-S1..S5

## 1. Scope & Method

Reviewed the full attack surface of Task 1: REST (Fiber v3), gRPC (grpc-go), auth (JWT HS256 + bcrypt), Mongo persistence, worker, config, Docker build/runtime, and the repo's secret hygiene. Method: static review of every production file, `govulncheck`, `go vet`, `go test -race`, live probes against the compose stack, and builder/runtime image inspection.

**Out of scope:** Task 2 (lottery design), third-party code beyond dependency scanning, penetration testing (proportionate: interview/local demo, no roles, no payments, no PII beyond demo emails).

## 2. Security Controls Verified (evidence)

| Control | Implementation | Evidence |
|---|---|---|
| AuthN — JWT alg pinned | `jwt.WithValidMethods(["HS256"])`; `none`-alg and wrong-alg rejected | `jwt.go` fresh read + existing tests (`TestVerifyRejectsNoneAlgorithm`) |
| AuthN — token lifetime | `exp` required (`WithExpirationRequired`), TTL from env (default 1h), expired rejected | `jwt.go` + `TestVerifyExpiredToken` |
| AuthN — issuer binding | `iss` validated on verify (`WithIssuer`) — rejects cross-service tokens even with a shared secret | **added this pass** + `TestVerifyRejectsWrongIssuer` |
| AuthN — secret strength | `JWT_SECRET` ≥32 bytes enforced at startup (fail-fast), env-only, never logged | `config.go` L36–38; `TestLoadMissingSecretFails`, `TestLoadShortSecretFails` |
| AuthN — bearer parsing | Case-insensitive `Bearer ` prefix, trimmed; same parser for REST + gRPC | `middleware.go`, `grpcapi/server.go` |
| AuthZ | A8 (documented): any authenticated caller may update/delete any user — accepted for the challenge, noted in README #6 | [[025_software_architecture_document]] §6 |
| Password storage | bcrypt cost 10; 8–72 char policy (72 = bcrypt input limit); hash never in responses; `password` not mutable via PUT (unknown-field rejection) | `bcrypt.go`, `domain/user.go`, `errors.go` decodeJSON |
| Login enumeration | Identical 401 body for unknown email / wrong password; **timing flattened with a dummy bcrypt compare** | **fixed this pass** — live probe before/after (see §4 SEC-02) |
| Input validation | Whitelist domain rules shared by REST + gRPC; email normalized (trim+lower); ObjectID shape checked before repo; 1 MB body limit; unknown JSON fields rejected | `domain/user.go`, `errors.go`, `app.go` |
| Injection | Mongo queries use bound values (`bson.M{...}`), no string-built filters; user input cannot inject operators or paths | `user_repo.go` full read — no `$where`, no eval, no concatenation |
| Error hygiene | Central envelope; generic 500s (no stack traces, no driver errors); 401/404/409 bodies identical in shape | `errors.go`, smoke checks |
| Logging | `slog` JSON: method/path/status/duration/request_id only; no token/password material (unit-asserted) | `middleware.go`, `TestLoggingMiddlewareNeverLogsSecrets` |
| Transport | Plain HTTP locally (documented); TLS termination at reverse proxy in production | README #15 |
| Docker runtime | Multi-stage; runtime = alpine + single stripped binary; **non-root uid 10001**; no `.env`/`.git`/source in any stage | image inspection + `docker exec id` this pass |
| Secrets in repo | `.env` never committed (git history empty), only `.env.example` (placeholder) tracked; no hardcoded secrets in code (scan) | git log + grep scan this pass |
| DoS posture | Server timeouts (read 10s / write 15s / idle 60s), 1 MB body cap, Mongo pool bounds (max 50 / min 5, 5s selection timeout) | `app.go`, `main.go` |
| Dependencies | `govulncheck` — **0 reachable vulnerabilities** after toolchain bump | §4 SEC-03 |

## 3. Verification Commands (re-runnable)

```bash
govulncheck ./...                          # 0 reachable (go1.25.13 toolchain)
go vet ./... && go test -race ./...        # clean / green
bash scripts/smoke.sh                      # 20/20 vs compose
docker build --target builder -t chk . && docker run --rm --entrypoint sh chk -c 'ls -la /app'
                                           # no .env / .git / .agents / tools
```

## 4. Findings Register (this pass)

| ID | Finding | Severity | Status |
|----|---------|----------|--------|
| SEC-01 | No `.dockerignore` — local `.env` (real `JWT_SECRET`), `.git` history, `.agents/`, `tools/` entered the **builder image** (QA FINDING-S1 / DEF-003) | 🟢 Med | ✅ **Fixed + verified** — `.dockerignore` added; builder `/app` re-inspected: all absent; smoke 20/20 |
| SEC-02 | Login timing oracle: unknown email ~4 ms vs wrong-password ~62 ms (**~15× delta**, live-measured) — user enumeration vector (QA FINDING-S2) | 🟢 Med | ✅ **Fixed + verified** — dummy bcrypt compare on unknown email; live re-probe: 58.6 ms vs 58.0 ms; unit guard `TestLoginUnknownEmailRunsDummyCompare` |
| SEC-03 | 19 reachable Go **stdlib** vulnerabilities (crypto/tls, net/http, net/url, x509, …) pinned by `go 1.25.3` in `go.mod` — incl. TLS paths reachable via Mongo driver | 🟡 High (supply-chain) | ✅ **Fixed + verified** — `go.mod` → `go 1.25.13`; `govulncheck` re-run: **0 reachable**; full suite green on 1.25.13 |
| SEC-04 | JWT `iss` claim set but never validated on verify — weak token-confusion defense if the secret were ever shared across services | ⚪ Low | ✅ **Fixed + verified** — `jwt.WithIssuer` + `jwt.WithExpirationRequired`; new negative test |
| SEC-05 | No rate limiting on public endpoints (register/login) | 🟢 Med (prod) | 📄 **Documented** — README #14; accepted for challenge scope (072 parking lot: PO-owned, post-interview) |
| SEC-06 | Compose Mongo runs without auth, host port exposed | 🟢 Med (prod) | 📄 **Documented** — README #16; accepted for local demo |
| SEC-07 | gRPC reflection enabled (schema disclosure) | ⚪ Low | 📄 **Documented** — README #17; JWT-gated, disable in prod |
| SEC-08 | No security headers (`X-Content-Type-Options` etc.) on API responses | ⚪ Info | 📄 **Noted** — JSON API only; nosniff/no-store worth adding before any browser-facing deployment; no action for interview |

**Not found (scanned):** hardcoded secrets/keys, `os.Exec`/`eval`/`panic` paths, SQL/NoSQL injection patterns, path traversal, unsafe deserialization, secrets in request logs, password hash exposure, unknown-field smuggling.

## 5. Residual Risk Statement

- **Acceptable for submission.** No critical/high exploitable risk ships. All three medium findings were fixed and re-verified with live evidence this pass; remaining items are documented production hardenings (rate limiting, Mongo auth, TLS, reflection), all explicitly called out in the README assumptions table an interviewer can read.
- Supply-chain posture is now clean at build time; the running compose image was rebuilt from the fixed tree (`sevensolution-api` rebuilt, healthy, smoke 20/20).
- One non-reachable vulnerability remains in a required module (govulncheck: "not called by your code") — acceptable, tracked by the toolchain's patch cadence.

## 6. Sign-off

| ACT (072) | Status | Evidence |
|-----------|--------|----------|
| ACT-S1 govulncheck | ✅ Closed | SEC-03 — 0 reachable after toolchain bump |
| ACT-S2 login timing | ✅ Closed | SEC-02 — fixed, live re-probe + unit test |
| ACT-S3 secrets in layers / non-root / strip | ✅ Closed | SEC-01 + non-root uid 10001 + `-ldflags "-s -w"` verified |
| ACT-S4 JWT hardening | ✅ Closed | §2 AuthN rows + SEC-04 hardening added |
| ACT-S5 README production gaps | ✅ Closed | README #14–17 |
| ACT-S6 security test report | ✅ Closed | This document |

---

> **Related:** [[075_MM_qa_to_security_handoff]], [[043_defect_report]], [[072_MM_handoff_polish]], [[022_API_specification]], [[025_software_architecture_document]]
