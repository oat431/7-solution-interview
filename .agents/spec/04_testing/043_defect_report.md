---
document_type: Defect Report
version: "1.0"
status: Active
author: "QA"
created: "2026-08-15"
last_updated: "2026-08-15"
project_name: "7-Solutions Backend Interview Challenge — Task 1: User Management API"
project_id: "SS-INT-001"
classification: "Internal"
tags: [defect-report, bug-report, defect-tracking, swebok, iso-29119]
standard_ref:
  - SWEBOK v4 — Testing
  - ISO/IEC/IEEE 29119 — Software Testing
---

# Defect Report — User Management API

> **Project:** SS-INT-001 | **Version:** 1.0 | **Status:** Active
> **Last Updated:** 2026-08-15
> **Review baseline:** commit `a8c6a52` (main) + dev handoff docs 072/073
> **Related:** [[073_MM_dev_to_qa_handoff]] (FINDING-D1), [[042_test_cases]], [[022_API_specification]]

## 1. Purpose

Defects found during the QA polish pass (072 ACT-Q1..Q6). All defects were reproduced live against the compose stack. Two are contract questions routed to the PO per DEC-H03 / DEC-D01 — QA does **not** change the API contract unilaterally.

## 2. Defect Register

| ID | Title | Severity | Module | Status | Assigned | Reported | Fixed |
|----|-------|---------|--------|--------|---------|---------|-------|
| DEF-001 | 413 body-limit response uses `INTERNAL_ERROR` code | 🟢 Medium | httpapi (error mapping) | ✅ Resolved — PO decision Option A | PO (DEC-PO-01) → Dev | 2026-08-15 | 2026-08-15 |
| DEF-002 | Catch-all 404 uses `NOT_FOUND` code not defined in 022 §3 | ⚪ Low | httpapi (error mapping) | ✅ Resolved — PO decision: add `NOT_FOUND` row | PO (DEC-PO-02) | 2026-08-15 | 2026-08-15 |
| DEF-003 | No `.dockerignore` — `.env` enters Docker build context | 🟢 Medium | Docker/build | ✅ Resolved — fixed + verified by SecEng (061 SEC-01) | Dev + SecEng (ACT-S3) | 2026-08-15 | 2026-08-15 |

---

## 3. DEF-001 — 413 body limit reports `INTERNAL_ERROR`

| Field | Value |
|-------|-------|
| **Defect ID** | DEF-001 |
| **Severity** | 🟢 Medium — misleading code on a client-caused error; no functional break |
| **Priority** | 🟡 P2 |
| **Status** | New — `needs-po-decision` (blocks fix: DEC-D01) |
| **Module** | httpapi — `errorHandler` (`internal/infrastructure/httpapi/errors.go`) |
| **Test Case** | TC-QINT-11 (proposed, pending PO decision — see 042) |
| **Requirement** | 022 §3 error-code contract; 013 (no AC exists for 413 — spec gap, see §3.3) |

### Description

Requests exceeding the 1 MB `BodyLimit` are rejected with HTTP **413** and the correct message, but the envelope's `code` is `INTERNAL_ERROR` — the code reserved for unhandled *server* faults. `errorHandler` maps unknown `*fiber.Error` codes to `INTERNAL_ERROR`; Fiber's 413 is such a code. Confirms 072 ACT-Q2 suspicion (FINDING-D1) — QA independently reproduced against the live compose stack.

The 022 §3 error-codes table defines **no 413 row**, so both candidate fixes are contract changes → PO decision (DEC-H03):
- **Option A:** add `REQUEST_TOO_LARGE` (413) to the 022 table
- **Option B:** map 413 → `VALIDATION_ERROR` (400 family semantics, keep table small)

### Steps to Reproduce

| Step | Action | Expected Result | Actual Result |
|------|--------|----------------|--------------|
| 1 | `docker compose up -d` (stack healthy) | API healthy | API healthy |
| 2 | Create `big.json` ≈ 1.1 MB | — | — |
| 3 | `curl -s -X POST localhost:8080/api/v1/auth/register -H 'Content-Type: application/json' --data-binary @big.json` | 413 with a client-fault code | 413 `{"error":{"code":"INTERNAL_ERROR","message":"Request Entity Too Large"}}` |

### Environment

| Field | Value |
|-------|-------|
| OS | Windows 11 |
| Runtime | Go 1.25.3, Fiber v3, Docker 29.6.2 / Compose v5.3.1 |
| Stack | `sevensolution-api-1` (healthy), commit `a8c6a52` |

### Evidence

| Type | Value |
|------|-------|
| HTTP status | `413` (correct) |
| Body | `{"error":{"code":"INTERNAL_ERROR","message":"Request Entity Too Large"}}` |
| Code path | `errorHandler` → `errors.As(err, &fe)` → default `code = "INTERNAL_ERROR"` |
| Reproduced by | Dev (073 FINDING-D1) and QA independently, 2026-08-15 |

---

## 4. DEF-002 — 404 catch-all uses undocumented `NOT_FOUND` code

| Field | Value |
|-------|-------|
| **Defect ID** | DEF-002 |
| **Severity** | ⚪ Low — shape is correct, code is undocumented |
| **Priority** | 🟢 P3 |
| **Status** | New — `needs-po-decision` |
| **Module** | httpapi — `app.go` catch-all + `errors.go` fiber.Error mapping |
| **Requirement** | 022 §3 error-code table |

### Description

Unknown routes return `{"error":{"code":"NOT_FOUND","message":"Route not found"}}` with HTTP 404. The envelope **shape** conforms to 022 §3, but `NOT_FOUND` is not in the 022 error-codes table (the only 404 code defined is `USER_NOT_FOUND`). Same for the `*fiber.Error` 404 branch in `errorHandler`. PO to decide: add a `NOT_FOUND` (404, unknown route) row to the 022 table, or reuse another code.

### Evidence (live, 2026-08-15)

```
GET http://localhost:8080/api/v1/does-not-exist
→ HTTP 404
  {"error":{"code":"NOT_FOUND","message":"Route not found"}}
```

Note: the new QA smoke checks (`scripts/smoke.sh`, ACT-Q4) currently assert the **observed** `NOT_FOUND` value; if PO picks a different code, update the smoke assertion in the same change.

---

## 5. DEF-003 — No `.dockerignore`; `.env` enters the build context

| Field | Value |
|-------|-------|
| **Defect ID** | DEF-003 |
| **Severity** | 🟢 Medium (interview/local context) — would be 🟡 High in any CI/shared-builder context |
| **Priority** | 🟡 P2 |
| **Status** | New |
| **Module** | Docker build (`Dockerfile` builder stage) |
| **Assigned To** | Dev, cross-check SecEng (072 ACT-S3) |

### Description

There is no `.dockerignore`. The builder stage's `COPY . .` pulls the entire context — including the local `.env` (real `JWT_SECRET`) and `.git/` — into the builder image layers. The final runtime stage only copies the binary, so the **final image is clean**, but:

1. builder-stage layers (image cache, `docker history` of intermediate targets, any push of a multi-target build) carry the secret;
2. every context change invalidates the `go mod download` cache layer less predictably;
3. this is exactly what 072 ACT-S3 ("no secrets in Docker layers — verify") asks to guard against.

### Fix (suggested, Dev)

Add a `.dockerignore`:

```
.env
.git
.agents
.qa_scratch
tools/
bin/
coverage.out
```

### Verification after fix

`docker build` the builder target and `docker history` / `docker save | tar -tvf` must not list `.env`; compose stack must still build and pass smoke 20/20.

**Update 2026-08-15 (075 security handoff):** escalated from inspection to demonstrated fact — `docker build --target builder` then `ls /app` shows `/app/.env` (191 bytes, real local secret), `/app/.git` (2.0M), `.agents/` and `tools/` inside the builder image. Final runtime image verified clean (binary only, non-root uid 10001). See [[075_MM_qa_to_security_handoff]].

**Update 2026-08-15 (076 security sign-off):** ✅ **Fixed and verified by SecEng.** `.dockerignore` added (`.env`, `.env.*`, `*.pem`, `.git`, `.agents`, `tools/`, `bin/`, `coverage.out`, IDE/OS noise). Rebuild + inspection: `/app` contains source/build files only — `.env`, `.git`, `.agents`, `tools/` all absent from the builder image; runtime image rebuilt, healthy, non-root uid 10001; smoke 20/20. See [[061_security_test_report]] SEC-01.

---

## 6. Defect Metrics (this pass)

| Metric | Value | Note |
|--------|-------|------|
| Defects found | 3 | 1 confirms dev's FINDING-D1; 2 new (1 spec-gap class, 1 build hygiene) |
| 🔴 Critical | 0 | |
| 🟡 High | 0 | |
| 🟢 Medium | 2 | DEF-001, DEF-003 |
| ⚪ Low | 1 | DEF-002 |
| Awaiting PO decision | 0 | all resolved 2026-08-15 (see 077_MM_po_review) |
| Regressions introduced by QA pass | 0 | unit 7/7, smoke 20/20, integration 10/10 |

## 7. Severity Definitions

| Severity | Definition |
|---------|-----------|
| 🔴 Critical | Data loss, security breach, service down |
| 🟡 High | Major feature broken, no workaround |
| 🟢 Medium | Misleading behavior or hygiene risk; functionally correct workaround exists |
| ⚪ Low | Cosmetic / documentation-level deviation |

---

## Related Documents

| Document | Relationship |
|----------|-------------|
| [[042_test_cases]] | Tests that found/verify these defects; traceability |
| [[073_MM_dev_to_qa_handoff]] | FINDING-D1 origin; ACT-Q2 |
| [[072_MM_handoff_polish]] | DEC-H03 (contract changes → PO), ACT-S3 |
| [[022_API_specification]] | The error-code contract at issue |

---

> **Template standard:** SWEBOK v4, ISO/IEC/IEEE 29119
