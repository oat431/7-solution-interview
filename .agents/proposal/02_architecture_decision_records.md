---
document_type: ADR (Architecture Decision Records)
version: "1.0"
status: Active
author: "PO"
created: "2026-08-15"
last_updated: "2026-08-15"
project_name: "7-Solutions Backend Interview Challenge — Task 2: Lottery Search System"
project_id: "SS-INT-002"
architect: "PO (candidate)"
classification: "Internal"
tags: [adr, architecture-decisions, wildcard-search, allocation, redis]
standard_ref:
  - SWEBOK v4 — Architecture
  - ISO/IEC/IEEE 42010 — Architecture Description
---

# ADR — Lottery Search System

## ADR Index

| ADR | Title | Status |
|-----|-------|--------|
| ADR-01 | Two-tier storage: Redis pool + relational ledger | ✅ Accepted |
| ADR-02 | Positional digit-set indexing (60 sets) over per-pattern materialization | ✅ Accepted |
| ADR-03 | SPOP-based atomic allocation (Lua claim script), no application locks | ✅ Accepted |
| ADR-04 | Lease + reaper reservation lifecycle | ✅ Accepted |
| ADR-05 | Scale-out by pool sharding on ticket_id hash | ✅ Accepted (when needed) |

---

## ADR-01: Two-tier storage — Redis pool + relational ledger

**Status:** ✅ Accepted | **Date:** 2026-08-15

**Context.** The system needs both (a) fast wildcard matching with atomic allocation over 10M tickets, and (b) durable, auditable ticket lifecycle records.

**Decision.** Valkey/Redis owns the *hot availability pool* (matching, allocation, leases). PostgreSQL (or MongoDB) owns the *system of record* (ticket ledger, status, audit, reporting). The pool is derived data, rebuildable from the ledger.

**Consequences.**
- ✅ Matching/allocation is memory-fast and linearizable (single-threaded command execution).
- ✅ Ledger keeps ACID guarantees for business truth without being on the search path.
- ⚠️ Two systems to operate; pool↔ledger consistency is eventual (lease state authoritative in pool).
- ⚠️ RAM cost (~2–2.5 GB) traded for latency and concurrency simplicity.

**Alternatives considered.** PostgreSQL-only with `SKIP LOCKED` (simple ops, slower claims, awkward interior-wildcard indexes); MongoDB-only (weaker claim semantics); Elasticsearch (overkill).

---

## ADR-02: Positional digit-set indexing over per-pattern materialization

**Status:** ✅ Accepted | **Date:** 2026-08-15

**Context.** 11⁶ ≈ 1.77M possible patterns. Materializing a set per pattern costs 10M × 2⁶ = 640M memberships (~13+ GB) — bounded but wasteful. Patterns are conjunctions of ≤6 positional predicates.

**Decision.** Maintain 60 sets `idx:pos{i}:{d}` (position × digit). A query intersects only the specified positions; wildcards contribute nothing. Optional `prefix:`/`suffix:` fast paths for the two README shapes (O(1)) are a later optimization, not the foundation.

**Consequences.**
- ✅ One uniform index serves prefix, suffix, and scattered patterns.
- ✅ 60M memberships ≈ 1.9 GB — 7× cheaper than per-pattern materialization.
- ✅ Query cost = O(k × min set size) ≈ O(10⁶) worst case, tens of ms.
- ⚠️ Intersection materializes a temp set per query (SINTERSTORE) — bounded by the smallest input set.

**Alternatives considered.** Per-pattern sets (fast, too much RAM); reversed-column LIKE in SQL (works for prefix/suffix only); inverted-index engines (Lucene-style, unnecessary complexity).

---

## ADR-03: SPOP-based atomic allocation — no application locks

**Status:** ✅ Accepted | **Date:** 2026-08-15

**Context.** The core correctness requirement: the same ticket must never be returned to two users simultaneously. Lock-based designs (DB row locks, distributed locks) add coordination, deadlock risk, and latency on the hottest path.

**Decision.** Allocation *is* a data-structure operation: `SPOP` atomically removes-and-returns one member of the match set. The full claim (intersect → pop → lease → stage) runs as one **Lua script**, so the whole sequence executes as a single atomic server step.

**Consequences.**
- ✅ Duplicate allocation is impossible by construction — two concurrent identical queries each pop *different* members.
- ✅ No locks, no deadlocks, no distributed coordination; ~100K+ allocations/s.
- ✅ Correctness argument is easy to state and to prove in an interview ("single-threaded Redis serializes the two scripts").
- ⚠️ Couples the no-duplicate guarantee to Redis semantics — a reimplementation on another store must reproduce atomic pop (e.g., PG `SKIP LOCKED` variant).

**Alternatives considered.** `SELECT … FOR UPDATE SKIP LOCKED` on the ledger (correct, slower, DB contention); application-level distributed locks (lock management burden, failure modes); optimistic CAS with retries (contention storms on hot patterns).

---

## ADR-04: Lease + reaper reservation lifecycle

**Status:** ✅ Accepted | **Date:** 2026-08-15

**Context.** SPOP removes the ticket from the pool. Without a way back, a crashed client permanently destroys inventory. "At the same time" also implies exclusivity should be time-bounded.

**Decision.** Every allocation carries a lease (ZSET `leases` scored by expiry, default 10 min). Confirm → SOLD; cancel → re-inject; expiry → reaper re-injects (SADD back to the six positional sets + ledger → AVAILABLE). Re-injection is idempotent.

**Consequences.**
- ✅ Crashes self-heal; no permanent inventory loss; bounded exclusivity window.
- ✅ Reaper is a trivial idempotent loop (no distributed coordination).
- ⚠️ A ticket is briefly double-offerable across a lease boundary (after expiry, before re-injection) — acceptable: exclusivity is defined per lease window.

**Alternatives considered.** Permanent removal (inventory leak on crash); two-phase commit with ledger (heavy, unnecessary); compensating jobs per allocation (more moving parts than one reaper).

---

## ADR-05: Scale-out by pool sharding on ticket_id hash

**Status:** ✅ Accepted (deferred until needed) | **Date:** 2026-08-15

**Context.** A single Redis instance already exceeds the challenge scale by 100×. Growth (100M+ tickets, higher QPS) needs a defined path.

**Decision.** When needed, shard the pool by `hash(ticket_id) % N`. Each ticket has exactly one home shard, so per-shard `SPOP` preserves global no-duplicate guarantees without cross-shard transactions. Pattern queries become scatter-gather (intersect per shard; pop on the owning shard).

**Consequences.**
- ✅ Linear allocation scaling; uniqueness preserved by the one-home-shard invariant.
- ✅ Rebuilds stay shard-local (ledger partitioned by the same hash).
- ⚠️ Intersection cost multiplies by shard count for patterns; mitigated by the prefix/suffix fast paths and by shard-local caps.

**Alternatives considered.** Redis Cluster (native, but cross-slot scripts are restricted — our Lua touches several keys per query); centralized coordinator shard (hot spot); moving matching to a search engine (extra system).

---

> **Related:** [[000_proposal_index]], [[01_solution_proposal]]
