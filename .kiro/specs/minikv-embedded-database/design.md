# Design Document: MiniKV Embedded Database

## Overview

MiniKV is an embedded key-value database implemented in Go that provides durable, crash-safe storage using a log-structured architecture. The system combines Write-Ahead Logging (WAL) for durability with periodic snapshots for fast recovery, while maintaining an in-memory index for high-performance reads.

### Key Design Decisions

1. **Log-Structured Storage**: All writes are appended to a WAL, avoiding expensive random I/O operations
2. **In-Memory Index**: Keys and metadata are kept in memory for O(1) read access
3. **Single-Writer Model**: Simplifies concurrency control and ensures consistency
4. **Lazy TTL Deletion**: Expired keys are cleaned up on access or by background workers
5. **Snapshot-Based Compaction**: Periodic snapshots enable WAL truncation and fast startup

### Design Goals

- **Simplicity**: Clean, understandable implementation under 3000 lines of code
- **Correctness**: Zero data loss for acknowledged writes
- **Performance**: 200K+ reads/sec, 50K+ writes/sec (batched)
- **Embeddability**: Works like `sql.DB` with a simple import
- **Observability**: Built-in metrics and debugging tools

## Architecture

### System Components

```
┌─────────────────────────────────────────┐
│        Application Code                 │
└────────────────┬────────────────────────┘
                 │
┌────────────────▼────────────────────────┐
│       MiniKV Public API                 │
│  (Get, Set, Delete, Batch, Scan)        │
└────────────────┬────────────────────────┘
                 │
┌────────────────▼────────────────────────┐
│      In-Memory Index (HashMap)          │
│   map[string]*Entry                     │
│   - Value: []byte                       │
│   - ExpiresAt: int64                    │
│   - CreatedAt: int64                    │
└─────┬──────────────────────────────┬────┘
      │                              │
┌─────▼──────────┐          ┌────────▼─────┐
│  WAL Manager   │          │  Snapshot    │
│  (Append-only) │          │  Manager     │
│  - Writer      │          │  - Reader    │
│  - Syncer      │          │  - Writer    │
└─────┬──────────┘          └────────┬─────┘
      │                              │
┌─────▼──────────────────────────────▼─────┐
│      File System Layer                    │
│  wal/*.log, snapshots/*.snap, MANIFEST    │
└───────────────────────────────────────────┘
```

### Directory Structure

```
/path/to/db/
├── MANIFEST               # Current WAL/snapshot state
├── LOCK                   # Process-level file lock
├── OPTIONS                # DB configuration
├── wal/
│   ├── 000001.log        # WAL segments
│   ├── 000002.log
│   └── 000003.log
└── snapshots/
    ├── 000005.snap       # Periodic snapshots
    └── 000010.snap
```

### Data Flow

#### Write Path
```
Set(key, value)
    ↓
[Validate key/value size]
    ↓
[Acquire write lock]
    ↓
[Encode WAL record]
    ↓
[Append to WAL file]
    ↓
[fsync if SyncAlways]
    ↓
[Update in-memory index]
    ↓
[Release write lock]
    ↓
[Return success]
```

#### Read Path
```
Get(key)
    ↓
[Acquire read lock]
    ↓
[Lookup in index]
    ↓
[Check if expired]
    ↓
[Release read lock]
    ↓
[Return value or ErrNotFound]
```

#### Startup Path
```
Open(path)
    ↓
[Create/open directory]
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

## Components and Interfaces

### Core Types

```go
// DB is the main database handle
type DB struct {
    mu            sync.RWMutex
    path          string
    opts          Options
    index         *MemIndex
    wal           *WALManager
    snapshot      *SnapshotManager
    manifest      *Manifest
    closed        bool
    lockFile      *os.File
    
    // Background workers
    syncTicker    *time.Ticker
    cleanTicker   *time.Ticker
    stopCh        chan struct{}
    wg            sync.WaitGroup
    
    // Metrics
    stats         *Stats
}

// MemIndex is the in-memory key-value index
type MemIndex struct {
    mu      sync.RWMutex
    data    map[string]*Entry
    size    int64  // total memory usage in bytes
}

// Entry represents a key-value pair with metadata
type Entry struct {
    Value     []byte
    ExpiresAt int64  // Unix nanoseconds, -1 = never
    CreatedAt int64  // Unix nanoseconds
}

// WALManager handles write-ahead logging
type WALManager struct {
    mu            sync.Mutex
    dir           string
    currentFile   *os.File
    currentSeq    uint64
    currentSize   int64
    syncMode      SyncMode
}

// SnapshotManager handles snapshot creation and loading
type SnapshotManager struct {
    mu  sync.Mutex
    dir string
}

// Manifest tracks WAL and snapshot state
type Manifest struct {
    mu                  sync.Mutex
    path                string
    currentWALSeq       uint64
    lastSnapshotSeq     uint64
    walSegments         []WALSegment
    snapshots           []SnapshotInfo
}
```

### Public API

```go
// Database lifecycle
func Open(opts Options) (*DB, error)
func (db *DB) Close() error
func (db *DB) Sync() error
func (db *DB) Compact() error

// Core operations
func (db *DB) Get(key []byte) ([]byte, error)
func (db *DB) GetWithContext(ctx context.Context, key []byte) ([]byte, error)
func (db *DB) Set(key, value []byte) error
func (db *DB) SetWithContext(ctx context.Context, key, value []byte) error
func (db *DB) Delete(key []byte) error
func (db *DB) Exists(key []byte) bool

// Extended operations
func (db *DB) SetNX(key, value []byte) (bool, error)
func (db *DB) Incr(key []byte) (int64, error)
func (db *DB) Decr(key []byte) (int64, error)
func (db *DB) IncrBy(key []byte, delta int64) (int64, error)
func (db *DB) CompareAndSwap(key, old, new []byte) (bool, error)
func (db *DB) GetAndSet(key, value []byte) ([]byte, error)

// TTL operations
func (db *DB) SetWithTTL(key, value []byte, ttl time.Duration) error
func (db *DB) TTL(key []byte) (time.Duration, error)
func (db *DB) Expire(key []byte, ttl time.Duration) (bool, error)
func (db *DB) Persist(key []byte) (bool, error)

// Iteration
func (db *DB) Scan(prefix []byte, limit int) Iterator
func (db *DB) ScanRange(start, end []byte, limit int) Iterator
func (db *DB) Keys(pattern string) ([][]byte, error)
func (db *DB) Count() int64

// Batch operations
func (db *DB) NewBatch() Batch

// Observability
func (db *DB) Stats() Stats
func (db *DB) DumpKeys(w io.Writer) error
```

### Internal Interfaces

```go
// WAL record types
const (
    RecordTypeSet         byte = 1
    RecordTypeDelete      byte = 2
    RecordTypeExpire      byte = 3
    RecordTypeBatchStart  byte = 4
    RecordTypeBatchEnd    byte = 5
)

// WALRecord represents a single WAL entry
type WALRecord struct {
    Type      byte
    Timestamp int64
    Key       []byte
    Value     []byte
    ExpiresAt int64
}

// Iterator for scanning keys
type Iterator interface {
    Next() bool
    Key() []byte
    Value() []byte
    Error() error
    Close() error
}

// Batch for atomic writes
type Batch interface {
    Set(key, value []byte) error
    SetWithTTL(key, value []byte, ttl time.Duration) error
    Delete(key []byte)
    Write() error
    Discard()
    Size() int
}
```

## Data Models

### WAL Record Format

Each WAL record is encoded as follows:

```
┌─────────────────────────────────────────────────────┐
│ Record Length (4 bytes, little-endian uint32)       │
├─────────────────────────────────────────────────────┤
│ Record Type (1 byte)                                │
│   1 = SET, 2 = DELETE, 3 = EXPIRE                  │
│   4 = BATCH_START, 5 = BATCH_END                   │
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

**Encoding Details:**
- All multi-byte integers use little-endian byte order
- Varint encoding uses Go's `binary.PutUvarint` for space efficiency
- CRC32 checksum covers all fields except the checksum itself
- Record length includes all fields except the length field itself

### Snapshot Format

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
│         (repeated for each record)        │
├──────────────────────────────────────────┤
│ Footer CRC32 (4 bytes)                   │
└──────────────────────────────────────────┘
```

**Snapshot Properties:**
- Keys are stored in lexicographic order for efficient scanning
- Expired keys are excluded during snapshot creation
- Footer CRC32 covers the entire snapshot for integrity verification
- Format version allows for future format evolution

### MANIFEST Format

The MANIFEST file uses a simple text-based format (YAML-like):

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

**MANIFEST Updates:**
- Written atomically using write-then-rename pattern
- Updated after each compaction
- Updated when new WAL segments are created
- Loaded during database startup to determine recovery strategy

### In-Memory Index Structure

```go
type MemIndex struct {
    mu      sync.RWMutex
    data    map[string]*Entry
    size    int64
}

type Entry struct {
    Value     []byte
    ExpiresAt int64  // -1 means no expiration
    CreatedAt int64
}
```

**Index Properties:**
- HashMap provides O(1) average-case lookup
- Keys are stored as strings (converted from []byte)
- Values are stored as byte slices to avoid copying
- Size tracking enables memory limit enforcement
- RWMutex allows concurrent reads

**Memory Overhead Calculation:**
```
Per-key overhead = 
    len(key) +           // key bytes
    len(value) +         // value bytes
    16 +                 // ExpiresAt + CreatedAt
    ~48                  // map overhead + pointer + struct padding
≈ len(key) + len(value) + 64 bytes
```

## Correctness Properties

*A property is a characteristic or behavior that should hold true across all valid executions of a system—essentially, a formal statement about what the system should do. Properties serve as the bridge between human-readable specifications and machine-verifiable correctness guarantees.*


### Property Reflection

After analyzing all acceptance criteria, I identified the following redundancies:

**Redundant Properties:**
- 5.6 (automatic compaction trigger) is identical to 3.6
- 12.1 (key size validation) is identical to 2.8
- 12.2 (value size validation) is identical to 2.9
- 12.3 (read-only error) is identical to 1.6
- 12.5 (batch size validation) is identical to 7.6
- 12.6 (corrupt WAL error) is identical to 4.2
- 14.3 (snapshot record format) is covered by 14.2 round-trip property
- 14.5 (byte order) is covered by encoding round-trip properties
- 14.6 (varint encoding) is covered by encoding round-trip properties
- 15.6 (startup time) is identical to 1.7

**Combined Properties:**
- 2.1, 2.3, 2.4 can be combined into a single "Set-Get round-trip" property
- 3.1, 3.5, 3.7 can be combined into a single "WAL encoding round-trip" property
- 4.1, 4.4, 4.5 can be combined into a single "crash recovery preserves data" property
- 6.1, 6.2, 6.3 can be combined into a single "TTL expiration" property
- 9.3, 9.6, 9.7 can be combined into a single "integer operations" property
- 14.1, 14.2 can be combined into a single "storage format round-trip" property

After reflection, we have approximately 50 unique, non-redundant properties to implement.

### Correctness Properties

Property 1: Set-Get Round Trip
*For any* valid key and value, after calling Set(key, value), calling Get(key) should return the same value.
**Validates: Requirements 2.1, 2.3, 2.4**

Property 2: Get Non-Existent Key
*For any* key that has never been set, calling Get(key) should return ErrNotFound.
**Validates: Requirements 2.2**

Property 3: Delete Makes Key Non-Existent
*For any* key, after calling Set(key, value) then Delete(key), calling Get(key) should return ErrNotFound.
**Validates: Requirements 2.5**

Property 4: Delete Idempotence
*For any* key, calling Delete(key) multiple times should succeed without error, even if the key doesn't exist.
**Validates: Requirements 2.6**

Property 5: Exists Reflects Key State
*For any* key, Exists(key) should return true if and only if Get(key) would not return ErrNotFound.
**Validates: Requirements 2.7**

Property 6: Key Size Validation
*For any* key with length > 1024 bytes, calling Set(key, value) should return ErrKeyTooLarge.
**Validates: Requirements 2.8**

Property 7: Value Size Validation
*For any* value with length > 10 MB, calling Set(key, value) should return ErrValueTooLarge.
**Validates: Requirements 2.9**

Property 8: Database Open Creates Directory
*For any* valid directory path, calling Open(path) should create the directory structure if it doesn't exist.
**Validates: Requirements 1.1**

Property 9: Exclusive File Lock
*For any* database path, if one process has the database open, attempting to open it from another process should return an error.
**Validates: Requirements 1.2, 1.5**

Property 10: Open-Close-Reopen Preserves Data
*For any* set of key-value pairs, after writing them, closing the database, and reopening it, all keys should still be present with the same values.
**Validates: Requirements 1.3, 4.1, 4.4, 4.5**

Property 11: Close Releases Lock
*For any* database, after calling Close(), another process should be able to successfully open the database.
**Validates: Requirements 1.4**

Property 12: Read-Only Mode Prevents Writes
*For any* database opened in read-only mode, all write operations (Set, Delete, SetWithTTL) should return ErrReadOnly.
**Validates: Requirements 1.6**

Property 13: WAL Encoding Round Trip
*For any* valid WAL record, encoding it then decoding it should produce an equivalent record.
**Validates: Requirements 3.1, 3.5, 3.7, 14.1**

Property 14: Compaction Triggers on WAL Size
*For any* database, when the WAL size exceeds MaxWALSize, compaction should be triggered automatically.
**Validates: Requirements 3.6**

Property 15: Checksum Detects Corruption
*For any* WAL record with a corrupted checksum, attempting to replay it during recovery should detect the corruption and stop replay.
**Validates: Requirements 4.2, 4.3**

Property 16: WAL Replay Idempotence
*For any* WAL segment, replaying it multiple times should produce the same final database state.
**Validates: Requirements 4.6**

Property 17: Snapshot Contains All Non-Expired Keys
*For any* database state, after creating a snapshot, the snapshot should contain all non-expired keys in sorted order.
**Validates: Requirements 5.1, 6.11**

Property 18: Snapshot Format Round Trip
*For any* valid snapshot, encoding it then decoding it should produce an equivalent set of key-value pairs.
**Validates: Requirements 5.2, 14.2**

Property 19: Compaction Updates MANIFEST
*For any* database, after compaction completes, the MANIFEST should reflect the new snapshot and WAL state.
**Validates: Requirements 5.3**

Property 20: Compaction Deletes Old WAL
*For any* database, after compaction completes, WAL segments older than the snapshot should be deleted.
**Validates: Requirements 5.4**

Property 21: Compaction Allows Concurrent Operations
*For any* database, while compaction is running, read and write operations should continue to work correctly.
**Validates: Requirements 5.5**

Property 22: TTL Expiration
*For any* key set with TTL, after the TTL duration has elapsed, Get(key) should return ErrNotFound.
**Validates: Requirements 6.1, 6.2**

Property 23: TTL Query Returns Remaining Duration
*For any* key set with TTL, calling TTL(key) should return a duration approximately equal to the remaining time until expiration.
**Validates: Requirements 6.3**

Property 24: Expire Sets Expiration
*For any* existing key, after calling Expire(key, ttl), the key should expire after the specified duration.
**Validates: Requirements 6.6**

Property 25: Persist Removes Expiration
*For any* key with expiration, after calling Persist(key), the key should not expire.
**Validates: Requirements 6.8**

Property 26: Batch Operations Are Buffered
*For any* batch with operations added, the operations should not be visible in the database until Write() is called.
**Validates: Requirements 7.1**

Property 27: Batch Write Is Atomic
*For any* batch with multiple operations, after calling Write(), either all operations should be applied or none should be applied.
**Validates: Requirements 7.2, 7.3, 7.4**

Property 28: Batch Discard Abandons Operations
*For any* batch with operations added, after calling Discard(), none of the operations should be applied to the database.
**Validates: Requirements 7.5**

Property 29: Batch Size Validation
*For any* batch, if the total size of operations exceeds 100 MB, Write() should return ErrBatchTooBig.
**Validates: Requirements 7.6**

Property 30: Scan Prefix Filtering
*For any* prefix, calling Scan(prefix, limit) should return only keys that start with that prefix.
**Validates: Requirements 8.1**

Property 31: ScanRange Filtering
*For any* start and end keys, calling ScanRange(start, end, limit) should return only keys in the range [start, end).
**Validates: Requirements 8.2**

Property 32: Iterator Returns Sorted Keys
*For any* set of keys, iterating over them should return keys in lexicographic order.
**Validates: Requirements 8.3**

Property 33: Iterator Snapshot Isolation
*For any* iterator, keys added after the iterator is created should not be visible during iteration.
**Validates: Requirements 8.4, 10.4**

Property 34: Iterator Skips Expired Keys
*For any* iterator, keys that have expired should not be returned during iteration.
**Validates: Requirements 8.5**

Property 35: Keys Pattern Matching
*For any* glob pattern, calling Keys(pattern) should return only keys matching the pattern.
**Validates: Requirements 8.6**

Property 36: Count Returns Non-Expired Key Count
*For any* database state, Count() should return the number of keys that are not expired.
**Validates: Requirements 8.7**

Property 37: Iterator Limit
*For any* iterator created with a limit, the iterator should return at most that many keys.
**Validates: Requirements 8.8**

Property 38: SetNX Conditional Set
*For any* non-existent key, SetNX(key, value) should set the value and return true. For any existing key, SetNX should not modify the value and return false.
**Validates: Requirements 9.1, 9.2**

Property 39: Integer Operations
*For any* key containing an integer value, Incr, Decr, and IncrBy should correctly modify the integer value and return the new value.
**Validates: Requirements 9.3, 9.6, 9.7**

Property 40: Incr Non-Integer Error
*For any* key containing a non-integer value, calling Incr(key) should return ErrInvalidValue.
**Validates: Requirements 9.5**

Property 41: CompareAndSwap Success
*For any* key with value V1, calling CompareAndSwap(key, V1, V2) should set the value to V2 and return true.
**Validates: Requirements 9.8**

Property 42: CompareAndSwap Failure
*For any* key with value V1, calling CompareAndSwap(key, V2, V3) where V2 != V1 should not modify the value and return false.
**Validates: Requirements 9.9**

Property 43: GetAndSet Atomic Swap
*For any* key with value V1, calling GetAndSet(key, V2) should return V1 and set the value to V2.
**Validates: Requirements 9.10**

Property 44: Concurrent Reads Are Safe
*For any* database, multiple goroutines calling Get concurrently should not cause data races or incorrect results.
**Validates: Requirements 10.1**

Property 45: Concurrent Writes Are Serialized
*For any* database, multiple goroutines calling Set concurrently should result in all writes being applied correctly.
**Validates: Requirements 10.2**

Property 46: Read-Your-Writes
*For any* goroutine, after calling Set(key, value), an immediate Get(key) in the same goroutine should return the same value.
**Validates: Requirements 10.3**

Property 47: Context Cancellation
*For any* operation with a cancelled context, the operation should return context.Canceled or context.DeadlineExceeded.
**Validates: Requirements 11.1, 11.2, 11.3**

Property 48: Closed Database Returns Error
*For any* database, after calling Close(), all operations should return ErrClosed.
**Validates: Requirements 12.4**

Property 49: Stats Reflects Database State
*For any* database state, calling Stats() should return accurate counts for keys, WAL size, snapshots, and operations.
**Validates: Requirements 13.1, 13.2**

Property 50: DumpKeys Writes All Keys
*For any* database, calling DumpKeys(w) should write all non-expired keys to the writer.
**Validates: Requirements 13.5**

Property 51: MANIFEST Tracks State
*For any* database, the MANIFEST file should accurately track the current WAL sequence, last snapshot sequence, and active segments.
**Validates: Requirements 14.4**

## Error Handling

### Error Categories

**User Errors** (client mistakes, don't corrupt database):
- `ErrNotFound`: Key doesn't exist or is expired
- `ErrKeyTooLarge`: Key exceeds 1024 bytes
- `ErrValueTooLarge`: Value exceeds 10 MB
- `ErrReadOnly`: Write attempted on read-only database
- `ErrInvalidValue`: Value is not in expected format (e.g., non-integer for Incr)
- `ErrBatchTooBig`: Batch size exceeds 100 MB

**System Errors** (require recovery or intervention):
- `ErrCorruptWAL`: WAL checksum mismatch during recovery
- `ErrClosed`: Operation attempted on closed database
- I/O errors: Disk full, permission denied, etc.

**Panic Conditions** (programming bugs, should never happen):
- Index corruption
- Deadlock
- Nil pointer dereference

### Error Handling Strategy

**User Errors:**
- Return immediately with descriptive error
- Do not modify database state
- Log at DEBUG level

**System Errors:**
- Attempt graceful degradation where possible
- For corrupt WAL: stop replay, truncate at corruption point
- For disk full: stop accepting writes, allow reads
- Log at ERROR level with context

**Recovery Procedures:**

1. **Corrupt WAL Record:**
   - Stop replay at corrupt record
   - Truncate WAL at corruption point
   - Log warning with record position
   - Continue with partial recovery

2. **Corrupt Snapshot:**
   - Delete corrupt snapshot
   - Fall back to older snapshot
   - Replay all subsequent WAL segments
   - Log warning

3. **Disk Full:**
   - Stop accepting writes
   - Return appropriate I/O error
   - Allow reads to continue
   - Wait for space or manual intervention

4. **Lock Held:**
   - Return error immediately
   - Suggest checking for other processes
   - Do not attempt to break lock

### Crash Consistency Guarantees

**After Crash:**
- All acknowledged writes (where Set/Delete returned success) are durable
- No torn writes (checksums detect partial records)
- Database can be opened and recovered
- Partial writes at end of WAL are truncated

**Not Guaranteed:**
- Writes that returned an error may or may not be durable
- Writes in progress during crash are lost
- Exact timing of background operations (TTL cleanup, compaction)

## Testing Strategy

### Dual Testing Approach

MiniKV uses both unit tests and property-based tests for comprehensive coverage:

**Unit Tests:**
- Specific examples demonstrating correct behavior
- Edge cases (empty keys, maximum sizes, boundary conditions)
- Error conditions (invalid inputs, closed database, read-only mode)
- Integration points between components
- Regression tests for discovered bugs

**Property-Based Tests:**
- Universal properties that hold for all inputs
- Comprehensive input coverage through randomization
- Minimum 100 iterations per property test
- Each property test references its design document property
- Tag format: `// Feature: minikv-embedded-database, Property N: [property text]`

### Property-Based Testing Configuration

**Library:** Use `gopter` (https://github.com/leanovate/gopter) for property-based testing in Go

**Configuration:**
```go
parameters := gopter.DefaultTestParameters()
parameters.MinSuccessfulTests = 100  // Minimum iterations
parameters.MaxSize = 1000            // Maximum generated value size
```

**Example Property Test:**
```go
// Feature: minikv-embedded-database, Property 1: Set-Get Round Trip
func TestProperty_SetGetRoundTrip(t *testing.T) {
    properties := gopter.NewProperties(gopter.DefaultTestParameters())
    properties.MinSuccessfulTests = 100
    
    properties.Property("Set then Get returns same value", prop.ForAll(
        func(key, value []byte) bool {
            db := setupTestDB(t)
            defer db.Close()
            
            err := db.Set(key, value)
            if err != nil {
                return false
            }
            
            got, err := db.Get(key)
            if err != nil {
                return false
            }
            
            return bytes.Equal(got, value)
        },
        gen.SliceOfN(10, gen.UInt8()),  // Generate random keys
        gen.SliceOfN(100, gen.UInt8()), // Generate random values
    ))
    
    properties.TestingRun(t)
}
```

### Test Coverage Goals

- **Core operations:** 100% coverage
- **WAL encoding/decoding:** 100% coverage
- **Snapshot format:** 100% coverage
- **Error paths:** 90% coverage
- **Overall:** 85% coverage minimum

### Testing Phases

**Phase 1: Unit Tests**
- Test each component in isolation
- Mock dependencies where appropriate
- Focus on correctness of individual functions

**Phase 2: Property Tests**
- Implement property tests for all 51 properties
- Run with race detector enabled
- Verify properties hold across random inputs

**Phase 3: Integration Tests**
- Test full lifecycle (open → write → close → reopen)
- Test crash recovery scenarios
- Test concurrent operations
- Test large datasets (1M keys)

**Phase 4: Stress Tests**
- Long-running tests (hours)
- High concurrency (100+ goroutines)
- Memory pressure tests
- Disk full scenarios

**Phase 5: Benchmarks**
- Measure throughput (ops/sec)
- Measure latency (P50, P95, P99)
- Measure memory usage
- Measure startup time
- Compare against targets

### Crash Consistency Testing

**Approach:**
1. Write data to database
2. Simulate crash (kill process, don't call Close)
3. Reopen database
4. Verify all acknowledged writes are present
5. Repeat with different crash points

**Tools:**
- Use `syscall.Kill` to simulate crashes
- Use `fsync` barriers to control durability
- Use fault injection to simulate I/O errors

### Continuous Integration

- Run all tests on every commit
- Run property tests with 1000 iterations nightly
- Run stress tests weekly
- Track performance regressions (>5% degradation fails CI)
- Publish benchmark history

