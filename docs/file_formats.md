# MiniKV File Formats

## WAL Record
- Length prefix: uvarint
- Type: 1 byte
- Timestamp: int64 (little-endian)
- ExpiresAt: int64 (little-endian)
- Key length: uvarint
- Value length: uvarint
- Key bytes
- Value bytes
- CRC32 checksum (IEEE)

## Snapshot
- Magic: "MINIKVSN" (8 bytes)
- Version: uint32
- Timestamp: int64
- Record count: uint64
- Records:
  - Key length: uint64
  - Key bytes
  - Value length: uint64
  - Value bytes
  - ExpiresAt: int64
  - CreatedAt: int64
- Footer checksum: CRC32 of records

## MANIFEST
Text file with fields:
- `current_wal_seq`
- `last_snapshot_seq`
- `wal: <seq> "<path>"`
- `snapshot: <seq> "<path>"`

