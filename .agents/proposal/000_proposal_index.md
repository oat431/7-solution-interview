---
document_type: Proposal Index
version: "1.0"
status: Draft
author: "PO"
created: "2026-08-15"
last_updated: "2026-08-15"
project_name: "7-Solutions Backend Interview Challenge — Task 2: Lottery Search System"
project_id: "SS-INT-002"
classification: "Internal"
tags: [proposal, design-document, lottery, wildcard-search, redis]
---

# Proposal Index — Lottery Search System (Task 2)

> Design exercise only — **no code implementation**, per challenge.
> Task 1 (User Management API) spec lives in `.agents/spec/`.

## Document Map

| # | Document | Purpose |
|---|----------|---------|
| 01 | Solution Proposal | The main deliverable — architecture, data structures, matching algorithm, storage choice, performance analysis, concurrency strategy (maps 1:1 to the challenge's four required sections) |
| 02 | Architecture Decision Records | ADR-01..05: storage split, positional indexing, SPOP allocation, lease/reaper recovery, scaling strategy |

## Challenge Deliverable → Document Mapping

| Challenge deliverable | Location |
|-----------------------|----------|
| Proposed solution architecture, data structures, algorithms | 01 §3–§4 |
| Recommended database/storage choice + justification | 01 §5 + ADR-01 |
| Performance analysis (efficiency + tradeoffs) | 01 §8 |
| Concurrency/distribution strategy (no duplicate simultaneous results) | 01 §7 + ADR-03 |

## Evaluation Criteria Coverage

| Criterion | Where addressed |
|-----------|-----------------|
| Feasibility | 01 §2 (problem decomposition), §8 (numbers) |
| Performance | 01 §6, §8 |
| Correctness (distribution constraint) | 01 §7 (SPOP linearization argument) + ADR-03 |
| Real-world practicality | 01 §5 (two-tier storage), §9 (failure modes) + ADR-01 |
| Creativity | 01 §2 (cardinality insight), §6 (positional indexing) |

## Headline Design (TL;DR)

- **Storage:** Valkey/Redis as the live ticket pool + PostgreSQL (or Mongo) as the system of record
- **Matching:** 60 positional digit-sets → `SINTER` the specified positions → `SPOP` allocates
- **No-duplicates:** `SPOP` is atomic — allocation is a single linearized operation, zero locks
- **Crash safety:** lease (ZSET by expiry) + reaper re-injects unconfirmed tickets

## Status

- [x] Requirements analyzed (challenge README)
- [x] Proposal written (this folder)
- [ ] Review with candidate (dry-run interview answers)
