---

# MiniKV — Specification Document

**Version:** v0.1
**Status:** Design / Implementation-ready

---

## 1. Overview

**MiniKV** is a lightweight, embedded key–value database designed for learning, experimentation, and real-world use in small systems.

It prioritizes:

- simplicity
- correctness
- persistence
- composability with other systems (MiniSearchDB, FYI Notes)

MiniKV is **single-node**, **single-writer**, and optimized for **local application embedding**, not for distributed or multi-tenant use.

---

## 2. Goals

### Primary Goals

- Provide a durable key–value store with predictable performance
- Teach core database concepts:
  - WAL
  - snapshots
  - compaction
  - TTL

- Interoperate cleanly with MiniSearchDB and FYI Notes
- Be implementable in multiple languages using the same spec

### Secondary Goals

- Reasonable performance for small–medium workloads (≤ millions of keys)
- Clear file formats and deterministic behavior
- Easy inspection and debugging

---

## 3. Non-Goals (Explicit)

MiniKV will **not**:

- be distributed or replicated
- implement clustering or sharding
- support complex data structures (lists, sets, sorted sets)
- provide built-in authentication or networking
- guarantee linearizable reads across processes

---

## 4. Target Use Cases

- Metadata store for MiniSearchDB
- Parsed FYI Notes storage
- App configuration and feature flags
- Simple caching with TTL
- Learning database internals

---

## 5. Data Model

### Key

- Type: **byte array**
- Max size: implementation-defined (recommended ≤ 1 KB)
- Comparison: **lexicographic byte order**

### Value

- Type: **byte array**
- Max size: implementation-defined (recommended ≤ 10 MB)
- Interpreted by caller (JSON, binary, text, etc.)

### Record

```
(key, value, metadata)
```

### Metadata

- `created_at` (optional)
- `expires_at` (optional, Unix timestamp)
- `tombstone` flag (for deletes)

---

## 6. API Specification (Language-Agnostic)

### Core API

```text
Open(path, options) → DB
Close()

Get(key) → value | NotFound
Set(key, value)
Delete(key)

Exists(key) → bool
```

---

### Extended Operations

```text
SetNX(key, value) → bool        // set if not exists
Incr(key) → int64               // value must be integer
```

---

### TTL Support

```text
SetWithTTL(key, value, ttl)
TTL(key) → duration | NotFound | NoTTL
```

Behavior:

- Expired keys behave as deleted
- Expiration may be lazy or background-cleaned

---

### Iteration

```text
Scan(prefix, limit) → [(key, value)]
```

Requirements:

- Results sorted lexicographically by key
- Snapshot consistency during scan

---

### Transactions (Optional v0.2)

```text
Begin()
Commit()
Rollback()
```

v0.1 guarantees **atomic single-key operations only**.

---

## 7. Consistency & Concurrency Model

- **Single writer**
- **Multiple readers**
- Readers see a consistent snapshot
- Read-your-writes is guaranteed within the same goroutine/thread after commit
- Cross-thread immediate visibility is **best-effort**

---

## 8. Storage Architecture

MiniKV uses a **log-structured design**.

### Components

```
WAL (append-only)
Snapshots (periodic full state)
In-memory index (key → position)
```

---

## 9. On-Disk Layout

```
kv/
  MANIFEST
  wal/
    wal_000001.log
    wal_000002.log
  snapshots/
    snapshot_000010.kv
```

---

## 10. Write-Ahead Log (WAL)

### Purpose

- Durability
- Crash recovery
- Sequential writes only

### Record Types

- SET
- DELETE
- EXPIRE

### WAL Record Format (binary)

```
| record_len u32 LE |
| record_type u8    |
| key_len uvarint   |
| key_bytes         |
| value_len uvarint | (SET only)
| value_bytes       | (SET only)
| expires_at i64 LE | (optional, -1 if none)
| crc32 u32 LE      |
```

### Record Types

| Value | Meaning |
| ----- | ------- |
| 1     | SET     |
| 2     | DELETE  |

---

## 11. Snapshot Format

### Purpose

- Fast startup
- WAL truncation

### Snapshot Contents

- Full sorted keyspace
- Includes expiration timestamps
- Excludes expired keys

### Snapshot File Structure

```
| magic "MINIKVSN" |
| version u32     |
| record_count u64|
| repeated records|
```

Each record:

```
| key_len uvarint |
| key_bytes       |
| value_len uvarint |
| value_bytes     |
| expires_at i64 LE |
```

---

## 12. Startup & Recovery

On startup:

1. Load latest snapshot (if any)
2. Replay WAL files in order
3. Discard expired keys
4. Rebuild in-memory index

Guarantee:

- No acknowledged write is lost
- Duplicate WAL records are safe to replay

---

## 13. Compaction

### WAL Compaction

Triggered when:

- WAL size exceeds threshold
- Manual compaction requested

Steps:

1. Write snapshot
2. fsync snapshot
3. Update MANIFEST
4. Delete old WAL files

---

## 14. Memory Index

In-memory structure:

```
map[key] → {
  value_position,
  expires_at
}
```

Alternative implementations allowed:

- skip list
- B-tree
- sorted array + binary search

---

## 15. Expiration (TTL)

- Expiration timestamps stored per key
- Lazy deletion allowed
- Optional background cleaner
- Expired keys removed during:
  - Get
  - Scan
  - Snapshot

---

## 16. Error Handling

Errors must be explicit:

- `ErrNotFound`
- `ErrKeyExpired`
- `ErrInvalidValue`
- `ErrCorruptWAL`
- `ErrReadOnly`

Corruption handling:

- WAL CRC mismatch → stop replay, fail startup
- Snapshot corruption → fallback to WAL replay

---

## 17. Performance Targets (v0.1)

- Get: O(1) average
- Set: append-only write + map update
- Startup (100k keys): < 1 second
- Memory overhead: proportional to key count

---

## 18. Interoperability

### With MiniSearchDB

- Store:
  - document metadata
  - synonyms
  - boosts
  - index statistics

### With FYI Notes

- Persist parsed notes
- Version notes via overwrite
- TTL for temporary notes

---

## 19. Observability

Minimal metrics:

- key_count
- wal_size
- snapshot_count
- last_compaction_time

Optional logging hooks.

---

## 20. Acceptance Criteria (MVP)

MiniKV v0.1 is complete when:

- Data survives crash + restart
- WAL replay works correctly
- TTL expiration behaves correctly
- Scan returns ordered results
- Can store metadata for MiniSearchDB

---

## 21. Future Extensions (Deferred)

- Secondary indexes
- Multi-key transactions
- Network protocol (Redis RESP-like)
- Pluggable storage engines
- Compression
- Encryption-at-rest

---

## 22. Design Philosophy (Important)

MiniKV is:

- **boring on purpose**
- **explicit over clever**
- **small enough to re-implement**

The goal is understanding and composability, not feature accumulation.

---

If you want, next I can:

- write the **MiniKV file format spec (byte-level, hex examples)** like we did for MiniSearchDB,
- design a **shared WAL library** used by MiniKV _and_ MiniSearchDB,
- or give you a **Go-first package layout + milestone plan**.

You’re building a really elegant little storage ecosystem here — this is exactly how real systems evolve.
