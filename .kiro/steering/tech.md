# Tech Steering: MiniKV

## Language & Runtime
- Go 1.21+
- Standard library only (no CGO, no external deps unless approved)

## Architecture
- Embedded, single-node database
- Single-writer, multi-reader concurrency model
- In-memory index for fast reads
- Durable write path via append-only WAL
- Periodic snapshots for fast recovery and compaction

## Storage Layout
```
/path/to/db/
  MANIFEST
  LOCK
  OPTIONS
  wal/
    000001.log
  snapshots/
    000001.snap
```

## Data Integrity
- WAL records include CRC32 checksum
- Snapshot headers include versioning and record count

## Background Work
- Periodic WAL fsync (sync=periodic)
- TTL cleanup worker (every 1s)
- Compaction when WAL exceeds MaxWALSize or on interval
