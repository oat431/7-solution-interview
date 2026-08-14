---
document_type: Database Schema (DDL)
version: "1.0"
status: Draft
author: "PO"
created: "2026-08-15"
last_updated: "2026-08-15"
project_name: "7-Solutions Backend Interview Challenge — Task 1: User Management API"
project_id: "SS-INT-001"
tech_lead: "Candidate"
classification: "Internal"
tags: [mongodb, schema, indexes, golang]
standard_ref:
  - SWEBOK v4 — Design
  - MongoDB Data Modeling Documentation
---

# Database Schema — MongoDB

> **Project:** SS-INT-001 | **Version:** 1.0 | **Status:** Draft
> Store: MongoDB (official Go driver). Database: `userdb` (env `DB_NAME`).

## 1. Collections

Single collection — `users`.

## 2. User Document

| BSON field | Go type | BSON tag | Constraint |
|------------|---------|----------|------------|
| `_id` | `primitive.ObjectID` | `_id` | Driver-generated |
| `name` | `string` | `name` | required, 1–100 chars |
| `email` | `string` | `email` | required, valid format, **unique** |
| `password_hash` | `string` | `password_hash` | bcrypt hash (never plaintext) |
| `created_at` | `time.Time` | `created_at` | server-set at creation (UTC) |

Example document:

```json
{
  "_id": { "$oid": "665f1c2d3e4f5a6b7c8d9e0f" },
  "name": "Ada Lovelace",
  "email": "ada@example.com",
  "password_hash": "$2a$10$N9qo8uLOickgx2ZMRZoMye...",
  "created_at": { "$date": "2026-08-15T10:00:00Z" }
}
```

## 3. Indexes

| Index name | Fields | Type | Purpose |
|------------|--------|------|---------|
| `_id_` | `_id` | default | primary lookup |
| `ux_users_email` | `email: 1` | unique | enforces AC-001e/f, AC-006e (race-safe duplicate rejection) |

### Creation

Programmatic, idempotent, at startup:

```go
idx := mongo.IndexModel{
    Keys:    bson.D{{Key: "email", Value: 1}},
    Options: options.Index().SetUnique(true).SetName("ux_users_email"),
}
coll.Indexes().CreateOne(ctx, idx)
```

> Decision (A12): create in code rather than a compose `init.js` — shows driver usage, works identically in manual & container runs, idempotent by name.
> Rationale for lowercase field names: challenge is JSON-first (REST responses use camelCase via JSON tags); BSON stays snake_case per Mongo convention. Mapping happens in the adapter.

## 4. Repository Operations (port contract → adapter)

| Operation | Query / Update | Notes |
|-----------|----------------|-------|
| `Create` | `InsertOne` | translate driver `mongo.IsDuplicateKeyError` → domain `ErrEmailExists` |
| `FindByID` | `FindOne({_id})` | `mongo.ErrNoDocuments` → domain `ErrNotFound` |
| `FindByEmail` | `FindOne({email})` | used by login |
| `ListAll` | `Find({})` | no pagination (A10) |
| `Update` | `UpdateOne({_id}, {$set: ...})` | only name/email; duplicate-key → `ErrEmailExists` |
| `Delete` | `DeleteOne({_id})` | deleted count 0 ⇒ `ErrNotFound` |
| `Count` | `CountDocuments({})` | worker metric (FR-009) |

## 5. Data Lifecycle

| Concern | Policy |
|---------|--------|
| TTL / retention | none (interview scope) |
| Backup | compose volume `mongo-data` persists across restarts |
| Seeding | none required; README curl samples self-seed |
| Password storage | bcrypt (cost 10) — hash computed in application layer, domain never touches plaintext |

---

> **Related:** [[022_API_specification]], [[025_software_architecture_document]]
