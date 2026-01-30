# MiniKV — Product Requirements Document (Go Implementation)

**Version:** v1.0
**Status:** Implementation-ready
**Target Language:** Go 1.21+
**Last Updated:** 2026-01-30

---

## Table of Contents

1. [Executive Summary](#1-executive-summary)
2. [Goals & Success Metrics](#2-goals--success-metrics)
3. [Target Audience & Use Cases](#3-target-audience--use-cases)
4. [Functional Requirements](#4-functional-requirements)
5. [Technical Architecture](#5-technical-architecture)
6. [API Specification (Go)](#6-api-specification-go)
7. [Storage Engine Design](#7-storage-engine-design)
8. [Concurrency & Thread Safety](#8-concurrency--thread-safety)
9. [Performance Requirements](#9-performance-requirements)
10. [Error Handling & Recovery](#10-error-handling--recovery)
11. [Observability & Debugging](#11-observability--debugging)
12. [Testing Strategy](#12-testing-strategy)
13. [Package Structure](#13-package-structure)
14. [Implementation Roadmap](#14-implementation-roadmap)
15. [Future Enhancements](#15-future-enhancements)

---

## 1. Executive Summary

**MiniKV** is an embedded, single-node key-value database written in pure Go. It provides durable storage with predictable performance for applications that need local persistence without the complexity of distributed databases.

### Core Value Proposition

- **Zero dependencies**: Pure Go, no CGO, no external libraries for core functionality
- **Crash-safe**: WAL-based durability with configurable sync modes
- **Simple API**: Familiar Redis-like operations with Go idioms
- **Embeddable**: Single `import`, works like `sql.DB`
- **Observable**: Built-in metrics and inspection tools
- **Educational**: Clean architecture for learning database internals

### Target Performance (v1.0)

- **Writes**: 50,000+ ops/sec (batched, sync=periodic)
- **Reads**: 200,000+ ops/sec (in-memory index)
- **Startup**: <1s for 1M keys
- **Memory**: ~100 bytes overhead per key
- **Storage**: Log-structured with automatic compaction

---

## 2. Goals & Success Metrics

### Primary Goals

1. **Durability**: Zero data loss on crash after acknowledged write
2. **Correctness**: Pass 100% of Redis-compatible test cases (for supported ops)
3. **Performance**: Meet or exceed BoltDB for single-key operations
4. **Simplicity**: Complete implementation in <3000 LOC
5. **Composability**: Integrate seamlessly with MiniSearchDB and FYI Notes

### Success Metrics

| Metric | Target (v1.0) |
|--------|---------------|
| Write throughput (batched) | ≥50K ops/sec |
| Read throughput | ≥200K ops/sec |
| P99 write latency (sync=always) | <5ms |
| P99 read latency | <100μs |
| Startup time (1M keys) | <1s |
| Memory overhead/key | <100 bytes |
| Test coverage | ≥85% |
| Zero data loss | 100% of crash tests |

### Non-Goals (v1.0)

- Network protocol or client-server mode
- Distributed replication or clustering
- Complex data types (lists, sets, hashes)
- Multi-version concurrency control (MVCC)
- Built-in authentication or encryption
- Cross-platform file locking

---

## 3. Target Audience & Use Cases

### Primary Users

1. **Go Developers**: Building applications needing local persistence
2. **Students**: Learning database internals and storage engines
3. **Researchers**: Prototyping new indexing or storage algorithms

### Use Cases

#### Core Use Cases

1. **Application Metadata Store**
   - Configuration management
   - Feature flags
   - User preferences
   - Session storage

2. **MiniSearchDB Backend**
   - Document metadata
   - Index statistics
   - Synonym mappings
   - Field boosts

3. **FYI Notes Storage**
   - Parsed note content
   - Note versioning
   - Tag indexes
   - Temporary drafts (with TTL)

4. **Caching Layer**
   - API response caching
   - Computed results
   - Rate limiting counters
   - Temporary data with TTL

5. **Development/Testing**
   - Fast integration test fixtures
   - Mock data storage
   - Reproducible test environments

---

## 4. Functional Requirements

### 4.1 Core Operations (MUST HAVE)

| Operation | Description | Atomicity |
|-----------|-------------|-----------|
| `Get(key)` | Retrieve value by key | Atomic |
| `Set(key, value)` | Store key-value pair | Atomic |
| `Delete(key)` | Remove key | Atomic |
| `Exists(key)` | Check key existence | Atomic |

### 4.2 Extended Operations (MUST HAVE)

| Operation | Description | Notes |
|-----------|-------------|-------|
| `SetNX(key, value)` | Set if not exists | Returns bool success |
| `Incr(key)` | Increment integer value | Atomic counter |
| `Decr(key)` | Decrement integer value | Atomic counter |
| `IncrBy(key, delta)` | Increment by delta | Atomic |

### 4.3 TTL Support (MUST HAVE)

| Operation | Description |
|-----------|-------------|
| `SetWithTTL(key, value, ttl)` | Set with expiration |
| `TTL(key)` | Get remaining TTL |
| `Expire(key, ttl)` | Set expiration on existing key |
| `Persist(key)` | Remove expiration |

**TTL Behavior**:
- Expired keys behave as deleted
- Lazy deletion on read
- Background cleaner runs every 1 second
- Expired keys excluded from snapshots

### 4.4 Iteration (MUST HAVE)

| Operation | Description |
|-----------|-------------|
| `Scan(prefix, limit)` | Iterate keys with prefix |
| `ScanRange(start, end, limit)` | Range scan |
| `Keys(pattern)` | List keys matching pattern |
| `Count()` | Total key count |

**Iteration Guarantees**:
- Lexicographically sorted results
- Snapshot-consistent view
- No torn reads during concurrent writes

### 4.5 Batch Operations (MUST HAVE)

```go
type Batch interface {
    Set(key, value []byte)
    Delete(key []byte)
    SetWithTTL(key, value []byte, ttl time.Duration)
    Write() error        // Atomic commit
    Discard()           // Rollback
}
```

**Requirements**:
- All operations in batch succeed or fail atomically
- Single WAL write for entire batch
- Batch size limited to 100MB

### 4.6 Atomic Operations (SHOULD HAVE)

| Operation | Description | Use Case |
|-----------|-------------|----------|
| `CompareAndSwap(key, old, new)` | CAS operation | Lock-free algorithms |
| `GetAndSet(key, value)` | Atomic swap | State transitions |

### 4.7 Range Operations (COULD HAVE)

| Operation | Description |
|-----------|-------------|
| `DeleteRange(start, end)` | Bulk delete |
| `CountRange(start, end)` | Count keys in range |

---

## 5. Technical Architecture

### 5.1 System Components

```
┌─────────────────────────────────────────┐
│           Application Code              │
└────────────────┬────────────────────────┘
                 │
┌────────────────▼────────────────────────┐
│          MiniKV Public API              │
│  (Get, Set, Delete, Batch, Scan)        │
└────────────────┬────────────────────────┘
                 │
┌────────────────▼────────────────────────┐
│         In-Memory Index                 │
│   (map[string]*Entry or *btree.BTree)   │
└─────┬──────────────────────────────┬────┘
      │                              │
┌─────▼──────────┐          ┌────────▼─────┐
│  WAL Manager   │          │  Snapshot    │
│  (Append-only) │          │  Manager     │
└─────┬──────────┘          └────────┬─────┘
      │                              │
┌─────▼──────────────────────────────▼─────┐
│           File System Layer               │
│  (wal/*.log, snapshots/*.snap)            │
└───────────────────────────────────────────┘
```

### 5.2 Directory Layout

```
/path/to/db/
├── MANIFEST               # Current active WAL/snapshot pointers
├── LOCK                   # Process-level file lock
├── OPTIONS                # DB configuration
├── wal/
│   ├── 000001.log        # WAL segments
│   ├── 000002.log
│   └── 000003.log
└── snapshots/
    ├── 000005.snap       # Periodic full snapshots
    └── 000010.snap
```

### 5.3 Data Flow

#### Write Path
```
Set(key, value)
    ↓
[Validate input]
    ↓
[Append to WAL]
    ↓
[fsync (if sync=always)]
    ↓
[Update in-memory index]
    ↓
[Return success]
```

#### Read Path
```
Get(key)
    ↓
[Check in-memory index]
    ↓
[Check if expired]
    ↓
[Return value or NotFound]
```

#### Startup Path
```
Open(path)
    ↓
[Acquire file lock]
    ↓
[Read MANIFEST]
    ↓
[Load latest snapshot]
    ↓
[Replay WAL segments]
    ↓
[Build in-memory index]
    ↓
[Start background workers]
    ↓
[Return DB handle]
```

---

## 6. API Specification (Go)

### 6.1 Core Types

```go
package minikv

import (
    "context"
    "time"
)

// DB is the main database handle
type DB struct {
    // unexported fields
}

// Options configures database behavior
type Options struct {
    // DirPath is the database directory
    DirPath string

    // SyncMode controls fsync behavior
    SyncMode SyncMode

    // MaxWALSize triggers compaction when exceeded
    MaxWALSize int64 // default: 64MB

    // MaxMemoryBytes limits in-memory index size
    MaxMemoryBytes int64 // default: 1GB

    // EnableMetrics enables Prometheus metrics
    EnableMetrics bool

    // Logger for structured logging
    Logger Logger

    // ReadOnly opens database in read-only mode
    ReadOnly bool

    // SnapshotInterval controls automatic snapshot frequency
    SnapshotInterval time.Duration // default: 1 hour
}

type SyncMode int

const (
    SyncAlways   SyncMode = iota // fsync every write (durable, slow)
    SyncPeriodic                  // fsync every 1s (balanced)
    SyncManual                    // app controls via Sync() (fast, risky)
)

// Common errors
var (
    ErrNotFound      = errors.New("minikv: key not found")
    ErrKeyExpired    = errors.New("minikv: key expired")
    ErrInvalidValue  = errors.New("minikv: invalid value")
    ErrCorruptWAL    = errors.New("minikv: corrupt WAL")
    ErrReadOnly      = errors.New("minikv: database is read-only")
    ErrKeyTooLarge   = errors.New("minikv: key exceeds max size")
    ErrValueTooLarge = errors.New("minikv: value exceeds max size")
    ErrClosed        = errors.New("minikv: database is closed")
    ErrBatchTooBig   = errors.New("minikv: batch size exceeds limit")
)

// Size limits
const (
    MaxKeySize   = 1024        // 1KB
    MaxValueSize = 10 * 1024 * 1024  // 10MB
    MaxBatchSize = 100 * 1024 * 1024 // 100MB
)
```

### 6.2 Database Lifecycle

```go
// Open opens or creates a database at the specified path
func Open(opts Options) (*DB, error)

// Close closes the database and releases resources
// Flushes pending WAL writes and stops background workers
func (db *DB) Close() error

// Sync forces a fsync of pending writes
func (db *DB) Sync() error

// Compact triggers manual compaction
// Creates snapshot and truncates WAL
func (db *DB) Compact() error
```

### 6.3 Core Operations

```go
// Get retrieves the value for the given key
// Returns ErrNotFound if key doesn't exist or is expired
func (db *DB) Get(key []byte) ([]byte, error)

// GetWithContext retrieves with cancellation support
func (db *DB) GetWithContext(ctx context.Context, key []byte) ([]byte, error)

// Set stores a key-value pair
// Overwrites existing value if key exists
func (db *DB) Set(key, value []byte) error

// SetWithContext stores with cancellation support
func (db *DB) SetWithContext(ctx context.Context, key, value []byte) error

// Delete removes a key
// No error if key doesn't exist
func (db *DB) Delete(key []byte) error

// Exists checks if key exists and is not expired
func (db *DB) Exists(key []byte) bool
```

### 6.4 Extended Operations

```go
// SetNX sets key only if it doesn't exist
// Returns true if value was set, false if key already exists
func (db *DB) SetNX(key, value []byte) (bool, error)

// Incr increments the integer value of key by 1
// Creates key with value 1 if it doesn't exist
// Returns ErrInvalidValue if value is not an integer
func (db *DB) Incr(key []byte) (int64, error)

// Decr decrements the integer value of key by 1
func (db *DB) Decr(key []byte) (int64, error)

// IncrBy increments the integer value of key by delta
func (db *DB) IncrBy(key []byte, delta int64) (int64, error)

// CompareAndSwap performs atomic compare-and-swap
// old can be nil to check for non-existence
func (db *DB) CompareAndSwap(key, old, new []byte) (bool, error)

// GetAndSet atomically sets new value and returns old value
// Returns ErrNotFound if key doesn't exist
func (db *DB) GetAndSet(key, value []byte) ([]byte, error)
```

### 6.5 TTL Operations

```go
// SetWithTTL sets key with expiration
func (db *DB) SetWithTTL(key, value []byte, ttl time.Duration) error

// TTL returns remaining time to live
// Returns -1 if key has no expiration
// Returns ErrNotFound if key doesn't exist
func (db *DB) TTL(key []byte) (time.Duration, error)

// Expire sets expiration on existing key
// Returns false if key doesn't exist
func (db *DB) Expire(key []byte, ttl time.Duration) (bool, error)

// Persist removes expiration from key
// Returns false if key doesn't exist
func (db *DB) Persist(key []byte) (bool, error)
```

### 6.6 Iteration

```go
// Iterator represents a key-value iterator
type Iterator interface {
    // Next advances to the next key
    Next() bool

    // Key returns current key (valid until next Next() call)
    Key() []byte

    // Value returns current value (valid until next Next() call)
    Value() []byte

    // Error returns any iteration error
    Error() error

    // Close releases iterator resources
    Close() error
}

// Scan returns iterator over keys with given prefix
// limit controls maximum keys to return (0 = unlimited)
func (db *DB) Scan(prefix []byte, limit int) Iterator

// ScanRange returns iterator over keys in range [start, end)
func (db *DB) ScanRange(start, end []byte, limit int) Iterator

// Keys returns all keys matching glob pattern
// Pattern syntax: * (any chars), ? (single char)
func (db *DB) Keys(pattern string) ([][]byte, error)

// Count returns total number of non-expired keys
func (db *DB) Count() int64
```

### 6.7 Batch Operations

```go
// Batch represents an atomic write batch
type Batch interface {
    // Set adds a key-value pair to batch
    Set(key, value []byte) error

    // SetWithTTL adds key with expiration to batch
    SetWithTTL(key, value []byte, ttl time.Duration) error

    // Delete adds key deletion to batch
    Delete(key []byte)

    // Write atomically commits all operations
    // All operations succeed or all fail
    Write() error

    // Discard abandons the batch
    Discard()

    // Size returns current batch size in bytes
    Size() int
}

// NewBatch creates a new write batch
func (db *DB) NewBatch() Batch
```

### 6.8 Observability

```go
// Stats returns database statistics
type Stats struct {
    KeyCount         int64
    WALSize          int64
    WALSegmentCount  int
    SnapshotCount    int
    LastCompaction   time.Time
    MemoryUsageBytes int64

    // Operation counters
    TotalWrites      uint64
    TotalReads       uint64
    TotalDeletes     uint64
    TotalScans       uint64

    // Timing histograms (P50, P95, P99)
    WriteLatencyP50  time.Duration
    WriteLatencyP95  time.Duration
    WriteLatencyP99  time.Duration
    ReadLatencyP50   time.Duration
    ReadLatencyP95   time.Duration
    ReadLatencyP99   time.Duration
}

func (db *DB) Stats() Stats

// DumpKeys writes all keys to writer for debugging
func (db *DB) DumpKeys(w io.Writer) error
```

---

## 7. Storage Engine Design

### 7.1 WAL Record Format

```
┌─────────────────────────────────────────────────────┐
│ Record Length (4 bytes, little-endian uint32)       │
├─────────────────────────────────────────────────────┤
│ Record Type (1 byte)                                │
│   1 = SET, 2 = DELETE, 3 = EXPIRE, 4 = BATCH_START │
│   5 = BATCH_END                                     │
├─────────────────────────────────────────────────────┤
│ Timestamp (8 bytes, Unix nanoseconds)               │
├─────────────────────────────────────────────────────┤
│ Key Length (varint)                                 │
├─────────────────────────────────────────────────────┤
│ Key Data (variable)                                 │
├─────────────────────────────────────────────────────┤
│ Value Length (varint, SET only)                     │
├─────────────────────────────────────────────────────┤
│ Value Data (variable, SET only)                     │
├─────────────────────────────────────────────────────┤
│ Expires At (8 bytes, Unix nanoseconds, -1 = never)  │
├─────────────────────────────────────────────────────┤
│ CRC32 Checksum (4 bytes, IEEE polynomial)           │
└─────────────────────────────────────────────────────┘
```

### 7.2 Snapshot Format

```
┌──────────────────────────────────────────┐
│ Magic Number: "MINIKVSN" (8 bytes)       │
├──────────────────────────────────────────┤
│ Format Version (4 bytes, uint32)         │
├──────────────────────────────────────────┤
│ Snapshot Timestamp (8 bytes, Unix ns)    │
├──────────────────────────────────────────┤
│ Record Count (8 bytes, uint64)           │
├──────────────────────────────────────────┤
│ ┌──────────────────────────────────────┐ │
│ │ Key Length (varint)                  │ │
│ ├──────────────────────────────────────┤ │
│ │ Key Data                             │ │
│ ├──────────────────────────────────────┤ │
│ │ Value Length (varint)                │ │
│ ├──────────────────────────────────────┤ │
│ │ Value Data                           │ │
│ ├──────────────────────────────────────┤ │
│ │ Expires At (8 bytes, -1 = never)     │ │
│ └──────────────────────────────────────┘ │
│            (repeated for each record)     │
├──────────────────────────────────────────┤
│ Footer CRC32 (4 bytes)                   │
└──────────────────────────────────────────┘
```

### 7.3 MANIFEST File Format

```yaml
version: 1
current_wal_sequence: 3
last_snapshot_sequence: 2
wal_segments:
  - sequence: 1
    size: 1048576
  - sequence: 2
    size: 2097152
  - sequence: 3
    size: 524288
snapshots:
  - sequence: 2
    timestamp: 1706630400
    key_count: 50000
```

### 7.4 In-Memory Index

**Option 1: HashMap (Default)**
```go
type Entry struct {
    Value     []byte
    ExpiresAt int64  // Unix nanoseconds, -1 = never
    CreatedAt int64
}

type MemIndex struct {
    mu      sync.RWMutex
    data    map[string]*Entry
    size    int64  // total memory usage
}
```

**Option 2: B-Tree (for ordered scans)**
```go
import "github.com/google/btree"

type BTreeIndex struct {
    mu   sync.RWMutex
    tree *btree.BTree
}
```

**Trade-offs**:
- HashMap: O(1) reads, but requires sorting for scans
- B-Tree: O(log n) reads, but efficient ordered scans
- **Default choice**: HashMap (simpler, faster reads)

### 7.5 Compaction Strategy

**Trigger Conditions** (any of):
- WAL size exceeds `MaxWALSize`
- Manual `Compact()` call
- Scheduled periodic compaction

**Compaction Process**:
1. Create new snapshot file with increasing sequence number
2. Write all non-expired keys in sorted order
3. fsync snapshot file
4. Update MANIFEST atomically
5. Delete WAL segments older than snapshot
6. fsync MANIFEST

**Concurrency**:
- Reads continue during compaction (from in-memory index)
- Writes continue to new WAL segment
- Snapshot creation happens in background goroutine

---

## 8. Concurrency & Thread Safety

### 8.1 Concurrency Model

**Guarantees**:
- **Single-writer**: Only one write at a time (protected by mutex)
- **Multiple-readers**: Read operations don't block each other
- **Read-your-writes**: Same goroutine sees its writes immediately
- **Snapshot isolation**: Scans see consistent point-in-time view

**Lock Hierarchy** (prevent deadlocks):
```
1. DB.mu (protects overall state)
2. Index.mu (protects in-memory index)
3. WAL.mu (protects WAL writer)
```

### 8.2 Goroutine Safety

All public APIs are goroutine-safe:
```go
// Safe concurrent usage
var wg sync.WaitGroup
for i := 0; i < 100; i++ {
    wg.Add(1)
    go func(id int) {
        defer wg.Done()
        db.Set([]byte(fmt.Sprintf("key-%d", id)), []byte("value"))
    }(i)
}
wg.Wait()
```

### 8.3 Context Support

All operations support cancellation via context:
```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

value, err := db.GetWithContext(ctx, []byte("key"))
if err == context.DeadlineExceeded {
    // handle timeout
}
```

### 8.4 Background Workers

**WAL Syncer** (if SyncMode=SyncPeriodic):
- Runs every 1 second
- Calls fsync on WAL file
- Bounded by backpressure if falling behind

**TTL Cleaner**:
- Runs every 1 second
- Scans subset of keys for expired entries
- Bounded to 10ms per iteration

**Compaction Worker**:
- Triggers on threshold or schedule
- Runs in background goroutine
- Throttled to avoid I/O saturation

---

## 9. Performance Requirements

### 9.1 Throughput Targets

| Operation | Target (ops/sec) | Conditions |
|-----------|------------------|------------|
| Get | 200,000+ | In-memory index |
| Set (batched, sync=periodic) | 50,000+ | Batch size 100 |
| Set (sync=always) | 5,000+ | Single fsync per write |
| Delete | 50,000+ | Same as Set |
| Scan (1000 keys) | 1,000+ | Sequential read |

### 9.2 Latency Targets

| Operation | P50 | P95 | P99 |
|-----------|-----|-----|-----|
| Get | <10μs | <50μs | <100μs |
| Set (sync=periodic) | <50μs | <200μs | <500μs |
| Set (sync=always) | <500μs | <2ms | <5ms |
| Scan (prefix) | <100μs | <500μs | <1ms |

### 9.3 Resource Limits

| Resource | Limit |
|----------|-------|
| Memory overhead per key | <100 bytes |
| Max concurrent readers | 1000 |
| Max WAL file size | 64MB (default) |
| Max snapshot size | Unlimited |
| File descriptors | 3-10 |

### 9.4 Scalability

| Dataset Size | Startup Time | Memory Usage |
|--------------|--------------|--------------|
| 10K keys | <10ms | ~1MB |
| 100K keys | <100ms | ~10MB |
| 1M keys | <1s | ~100MB |
| 10M keys | <10s | ~1GB |

---

## 10. Error Handling & Recovery

### 10.1 Error Categories

**User Errors** (don't corrupt DB):
- `ErrNotFound`: Key doesn't exist
- `ErrKeyTooLarge`: Key exceeds max size
- `ErrValueTooLarge`: Value exceeds max size
- `ErrReadOnly`: Write attempted on read-only DB

**System Errors** (require recovery):
- `ErrCorruptWAL`: WAL checksum mismatch
- `ErrDiskFull`: Out of disk space
- `ErrIOError`: Filesystem I/O error

**Panic Conditions** (unrecoverable):
- Index corruption (programming bug)
- Deadlock (programming bug)

### 10.2 Recovery Strategies

**Corrupt WAL Record**:
```
1. Stop replaying at corrupt record
2. Log error with record position
3. Truncate WAL at corrupt position
4. Continue with partial recovery
5. Return warning to user
```

**Corrupt Snapshot**:
```
1. Delete corrupt snapshot
2. Fall back to older snapshot
3. Replay all WAL segments
4. Log warning
```

**Disk Full**:
```
1. Stop accepting writes
2. Return ErrDiskFull
3. Allow reads to continue
4. Wait for space or manual intervention
```

### 10.3 Crash Consistency

**Crash Recovery Process**:
```
1. Open database directory
2. Acquire file lock (fail if held)
3. Read MANIFEST
4. Load latest valid snapshot
5. Replay WAL segments in order
6. Verify checksums
7. Rebuild in-memory index
8. Remove partial writes (truncate at last valid record)
9. Ready for operations
```

**Guarantees**:
- All acknowledged writes are durable
- No torn writes (checksums detect partial records)
- Idempotent replay (same WAL can be replayed multiple times)

### 10.4 Testing for Crash Safety

**Fault injection**:
- Random crashes during writes
- Disk full simulation
- Corrupt byte injection
- Power loss simulation (on Linux: sync + hard reboot)

**Property testing**:
- Write N records → crash → recover → verify N records
- Sequential writes → verify order preserved
- Batch writes → verify atomicity

---

## 11. Observability & Debugging

### 11.1 Metrics (Prometheus Format)

```go
// Counters
minikv_operations_total{op="get|set|delete|scan"}
minikv_errors_total{type="not_found|io_error|corrupt"}
minikv_bytes_written_total
minikv_bytes_read_total

// Gauges
minikv_keys_total
minikv_wal_size_bytes
minikv_memory_usage_bytes
minikv_snapshot_age_seconds

// Histograms
minikv_operation_duration_seconds{op="get|set|delete|scan"}
minikv_batch_size_bytes
```

### 11.2 Structured Logging

```go
type Logger interface {
    Debug(msg string, fields ...Field)
    Info(msg string, fields ...Field)
    Warn(msg string, fields ...Field)
    Error(msg string, fields ...Field)
}

// Example log output (JSON)
{
  "timestamp": "2026-01-30T10:30:45Z",
  "level": "info",
  "msg": "compaction completed",
  "wal_size_before": 67108864,
  "wal_size_after": 0,
  "snapshot_size": 52428800,
  "keys_compacted": 1000000,
  "duration_ms": 1234
}
```

### 11.3 CLI Tool

```bash
# Inspect database
minikv-cli info /path/to/db
# Output:
# Keys: 1,000,000
# WAL Size: 64 MB
# Snapshots: 2
# Last Compaction: 2 hours ago

# Dump all keys
minikv-cli dump /path/to/db > keys.txt

# Get specific key
minikv-cli get /path/to/db mykey

# Set key
minikv-cli set /path/to/db mykey myvalue

# Scan with prefix
minikv-cli scan /path/to/db --prefix="user:"

# Trigger compaction
minikv-cli compact /path/to/db

# Verify integrity
minikv-cli verify /path/to/db
```

### 11.4 Debug API

```go
// DebugInfo returns detailed internal state
func (db *DB) DebugInfo() map[string]interface{}

// VerifyIntegrity checks data consistency
func (db *DB) VerifyIntegrity() error

// DumpWAL writes WAL contents for inspection
func (db *DB) DumpWAL(w io.Writer) error
```

---

## 12. Testing Strategy

### 12.1 Unit Tests

**Coverage targets**:
- Core operations: 100%
- WAL encoding/decoding: 100%
- Snapshot format: 100%
- Error paths: 90%

**Key test cases**:
```go
TestBasicOperations         // CRUD
TestTTLExpiration          // TTL behavior
TestConcurrentReads        // Goroutine safety
TestBatchAtomicity         // All-or-nothing
TestWALRecovery           // Crash recovery
TestSnapshotCompaction    // Compaction correctness
TestIteratorConsistency   // Scan isolation
```

### 12.2 Integration Tests

```go
TestFullLifecycle          // Open → Write → Close → Reopen
TestLargeDataset          // 1M keys
TestCrashRecovery         // Kill process mid-write
TestDiskFull              // Handle ENOSPC
TestConcurrentWriters     // Stress test
TestMemoryPressure        // Large values
```

### 12.3 Benchmark Suite

```go
BenchmarkGet              // Single-threaded reads
BenchmarkSet              // Single-threaded writes
BenchmarkBatchWrite       // Batch performance
BenchmarkScan             // Iteration speed
BenchmarkParallelReads    // Concurrent read scaling
BenchmarkStartup          // Recovery time
```

**Baseline targets** (on MacBook Pro M1):
```
BenchmarkGet-8              5000000    250 ns/op
BenchmarkSet-8              100000     15000 ns/op (sync=periodic)
BenchmarkBatchWrite-8       500000     3000 ns/op (100 keys/batch)
BenchmarkScan1000-8         10000      120000 ns/op
```

### 12.4 Fuzz Testing

```go
FuzzWALDecode             // Random byte sequences
FuzzSnapshotDecode        // Corrupt snapshots
FuzzKeyValuePairs         // Random inputs
FuzzCrashRecovery         // Random crash points
```

### 12.5 Property-Based Tests

Using `gopter` or similar:
```go
Property: Set(k, v) → Get(k) == v
Property: Delete(k) → Get(k) == ErrNotFound
Property: Batch atomicity
Property: Scan ordering
Property: TTL expiration
```

### 12.6 Continuous Benchmarking

- Run benchmarks on every commit
- Track performance regressions (>5% degradation fails CI)
- Publish benchmark history

---

## 13. Package Structure

```
minikv/
├── cmd/
│   └── minikv-cli/
│       └── main.go              # CLI tool
├── internal/
│   ├── wal/
│   │   ├── wal.go              # WAL writer
│   │   ├── reader.go           # WAL reader
│   │   └── format.go           # Record encoding
│   ├── snapshot/
│   │   ├── snapshot.go         # Snapshot manager
│   │   └── format.go           # Snapshot encoding
│   ├── index/
│   │   ├── hashmap.go          # HashMap index
│   │   └── btree.go            # B-Tree index (optional)
│   ├── manifest/
│   │   └── manifest.go         # MANIFEST manager
│   └── metrics/
│       └── metrics.go          # Prometheus metrics
├── minikv.go                   # Public API
├── batch.go                    # Batch operations
├── iterator.go                 # Scan/iteration
├── options.go                  # Configuration
├── errors.go                   # Error definitions
├── db_test.go                  # Integration tests
└── README.md

examples/
├── basic/
│   └── main.go                 # Basic usage
├── cache/
│   └── main.go                 # Caching example
└── minisearch/
    └── main.go                 # Integration with MiniSearchDB

benchmarks/
└── bench_test.go               # Comprehensive benchmarks

docs/
├── architecture.md
├── file_formats.md
└── migration_guide.md
```

**Implementation Note (2026-01-30):**
The Go module root is the repository root. It is normal (and idiomatic) for core files
like `minikv.go`, `options.go`, and `errors.go` to live at the repo root. The folders
`cmd/`, `examples/`, `benchmarks/`, and `docs/` will be added incrementally as features
stabilize. Avoid nesting a secondary `minikv/` folder inside the repo, since that would
change import paths and complicate module usage.

**Doc Note (2026-01-30):**
Compaction and MANIFEST update flows are documented in `docs/architecture.md` and
`docs/file_formats.md`. Refer there for current on-disk formats and recovery steps.

---

## 14. Implementation Roadmap

### Phase 1: Core Foundation (Weeks 1-2)

**Deliverables**:
- [ ] Package structure and build system
- [ ] WAL format and encoding
- [ ] Basic in-memory index (HashMap)
- [ ] Open/Close with file locking
- [ ] Get/Set/Delete operations
- [ ] Unit tests for core operations

**Success Criteria**:
- All basic operations work
- Data persists across restarts
- 80% test coverage

### Phase 2: Durability & Recovery (Weeks 3-4)

**Deliverables**:
- [ ] WAL replay on startup
- [ ] CRC checksums for corruption detection
- [ ] Crash recovery logic
- [ ] Snapshot format implementation
- [ ] MANIFEST file management
- [ ] Compaction trigger logic

**Success Criteria**:
- Zero data loss in crash tests
- Clean recovery from corrupt WAL
- Compaction produces valid snapshots

### Phase 3: Advanced Features (Weeks 5-6)

**Deliverables**:
- [ ] TTL support with lazy deletion
- [ ] Background TTL cleaner
- [ ] Batch operations
- [ ] Iterator/Scan implementation
- [ ] Atomic operations (CAS, GetAndSet)
- [ ] Context support for cancellation

**Success Criteria**:
- TTL expires keys correctly
- Batches are atomic
- Scans return sorted, consistent results

### Phase 4: Performance & Observability (Weeks 7-8)

**Deliverables**:
- [ ] Prometheus metrics
- [ ] Structured logging
- [ ] CLI tool
- [ ] Comprehensive benchmarks
- [ ] Performance tuning
- [ ] Memory profiling

**Success Criteria**:
- Meet all performance targets
- <100 bytes overhead per key
- Benchmarks in CI

### Phase 5: Testing & Documentation (Weeks 9-10)

**Deliverables**:
- [ ] Fuzz tests
- [ ] Property-based tests
- [ ] Crash consistency tests
- [ ] Example applications
- [ ] API documentation
- [ ] Architecture guide

**Success Criteria**:
- 85% test coverage
- All examples run successfully
- Documentation complete

### Phase 6: Integration & Polish (Weeks 11-12)

**Deliverables**:
- [ ] MiniSearchDB integration
- [ ] FYI Notes integration
- [ ] Migration tools
- [ ] Performance regression tests
- [ ] v1.0 release preparation

**Success Criteria**:
- Works in production use cases
- No critical bugs
- Ready for v1.0 tag

---

## 15. Future Enhancements

### v1.1: Advanced Indexing
- Secondary indexes
- Range queries with skip lists
- Prefix compression in snapshots

### v1.2: Transactions
- Multi-key transactions (ACID)
- Optimistic concurrency control
- MVCC for snapshot isolation

### v1.3: Network Protocol
- Redis RESP2/RESP3 compatibility
- TCP server mode
- Client library

### v1.4: Replication
- Single-leader replication
- WAL shipping
- Read replicas

### v1.5: Performance
- Bloom filters for negative lookups
- LZ4 compression in snapshots
- mmap for read-heavy workloads

### v2.0: Advanced Features
- Column families (like RocksDB)
- Backup/restore utilities
- Encryption at rest
- Point-in-time recovery

---

## Appendix A: Comparison with Alternatives

| Feature | MiniKV | BoltDB | BadgerDB | Redis |
|---------|--------|--------|----------|-------|
| Language | Go | Go | Go | C |
| Architecture | Log-structured | B+Tree (LMDB) | LSM Tree | In-memory |
| Durability | WAL + Snapshots | Copy-on-write | WAL + LSM | AOF/RDB |
| Concurrency | Single writer | Single writer | Multi-writer | Single thread |
| Transactions | Single-key (v1) | Multi-key | Multi-key | Multi-key |
| Network | No (v1) | No | No | Yes |
| TTL | Yes | No | Yes | Yes |
| Complexity | ~3K LOC | ~10K LOC | ~30K LOC | ~100K LOC |
| Learning Curve | Low | Medium | High | Medium |

**MiniKV Differentiators**:
- Simplest implementation for learning
- Designed for embedding, not networking
- Tight integration with MiniSearchDB ecosystem
- Explicit trade-offs documented

---

## Appendix B: Glossary

- **WAL**: Write-Ahead Log, append-only transaction log
- **Snapshot**: Point-in-time full database backup
- **Compaction**: Process of consolidating WAL into snapshot
- **Tombstone**: Marker indicating deleted key
- **LSM**: Log-Structured Merge-tree
- **MVCC**: Multi-Version Concurrency Control
- **CAS**: Compare-And-Swap atomic operation
- **fsync**: Force flush to physical disk
- **Varint**: Variable-length integer encoding

---

## Appendix C: Open Questions

1. **Index Choice**: HashMap vs B-Tree for default?
   - Recommendation: Start with HashMap, add B-Tree in v1.1

2. **Compression**: Should snapshots be compressed?
   - Recommendation: No in v1.0, add LZ4 in v1.5

3. **Memory Limits**: How to handle when index exceeds MaxMemoryBytes?
   - Recommendation: Return error on Open, document limitation

4. **File Format Versioning**: How to handle backwards compatibility?
   - Recommendation: Version in header, explicit migration tools

5. **Concurrency Model**: Allow multi-writer in v1.0?
   - Recommendation: No, keep single-writer for simplicity

---

## Appendix D: Success Metrics Dashboard

```
┌─────────────────────────────────────────────────────────┐
│ MiniKV v1.0 Release Checklist                          │
├─────────────────────────────────────────────────────────┤
│ ☐ All core APIs implemented                            │
│ ☐ WAL replay works correctly                           │
│ ☐ Compaction produces valid snapshots                  │
│ ☐ TTL expiration verified                              │
│ ☐ Crash recovery passes 1000 iterations                │
│ ☐ Test coverage ≥85%                                   │
│ ☐ Write throughput ≥50K ops/sec (batched)              │
│ ☐ Read throughput ≥200K ops/sec                        │
│ ☐ Memory overhead <100 bytes/key                       │
│ ☐ Startup time <1s for 1M keys                         │
│ ☐ CLI tool functional                                  │
│ ☐ Examples run successfully                            │
│ ☐ Documentation complete                               │
│ ☐ MiniSearchDB integration verified                    │
│ ☐ Zero critical bugs                                   │
└─────────────────────────────────────────────────────────┘
```

---

**Document Version**: 1.0
**Last Updated**: 2026-01-30
**Next Review**: After Phase 3 completion

---

## Summary

This PRD provides a complete blueprint for implementing MiniKV in Go. Key highlights:

1. **Clear Scope**: Focused on embedded use, not distributed systems
2. **Concrete API**: Go-idiomatic with context support
3. **Proven Architecture**: Log-structured storage with WAL + snapshots
4. **Performance Goals**: Quantified targets for throughput and latency
5. **Testing Strategy**: Comprehensive coverage including crash consistency
6. **Phased Delivery**: 12-week roadmap with clear milestones

The implementation should prioritize simplicity and correctness over premature optimization, maintaining the "boring on purpose" philosophy from the original spec.
