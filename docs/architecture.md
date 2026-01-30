# MiniKV Architecture

## Overview
MiniKV is an embedded, single-node key–value database in Go with a log-structured design. Writes go to a WAL, reads are served from an in-memory index, and periodic snapshots are used for fast recovery and compaction.

## Core Components
- **DB**: public API and lifecycle management (`Open`, `Close`, `Set`, `Get`, etc.)
- **WAL Manager**: append-only log storage with rotation and CRC checks
- **Snapshot Manager**: full snapshots for recovery and compaction
- **MemIndex**: in-memory index mapping keys to entries (value + metadata)
- **Manifest**: tracks WAL segments and snapshots for recovery

## Write Path
1. Validate key/value sizes
2. Append record to WAL
3. fsync based on SyncMode
4. Update in-memory index

## Read Path
1. Lookup key in MemIndex
2. Enforce TTL (lazy delete)
3. Return value or ErrNotFound

## Recovery Path
1. Load MANIFEST
2. Load latest snapshot
3. Replay WAL segments in order
4. Rebuild index

## Compaction
- Triggered on WAL rotation or manual `Compact()` call
- Creates a new snapshot (excludes expired keys)
- Deletes WAL segments older than the snapshot
- Updates MANIFEST atomically

## Background Workers
- **SyncPeriodic**: fsync WAL every 1s
- **TTL Cleaner**: removes expired keys every 1s

