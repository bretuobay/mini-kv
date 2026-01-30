# Requirements Document: MiniKV Embedded Database

## Introduction

MiniKV is an embedded, single-node key-value database written in pure Go. It provides durable storage with predictable performance for applications that need local persistence without the complexity of distributed databases. The system uses a log-structured storage architecture with Write-Ahead Logging (WAL) and periodic snapshots to ensure crash-safe durability while maintaining high performance for both reads and writes.

## Glossary

- **DB**: The main database handle that manages all operations and lifecycle
- **WAL**: Write-Ahead Log, an append-only transaction log that ensures durability
- **Snapshot**: A point-in-time full database backup used for fast recovery
- **Compaction**: The process of consolidating WAL entries into a snapshot and truncating old WAL files
- **Index**: The in-memory data structure (HashMap or B-Tree) that maps keys to their values and metadata
- **Record**: A single entry in the WAL or snapshot containing a key, value, and metadata
- **TTL**: Time-To-Live, the duration after which a key expires and behaves as deleted
- **Batch**: A collection of write operations that are committed atomically
- **Iterator**: A cursor for traversing keys in lexicographic order
- **MANIFEST**: A file tracking the current state of WAL segments and snapshots
- **Tombstone**: A marker indicating a deleted key in the storage layer
- **Sync_Mode**: Configuration controlling when data is flushed to disk (always, periodic, or manual)

## Requirements

### Requirement 1: Database Lifecycle Management

**User Story:** As a developer, I want to open, close, and manage database instances, so that I can safely persist data with proper resource management.

#### Acceptance Criteria

1. WHEN a developer calls Open with a valid directory path, THE DB SHALL create or open the database at that location
2. WHEN opening a database, THE DB SHALL acquire an exclusive file lock to prevent concurrent access
3. WHEN opening an existing database, THE DB SHALL load the latest snapshot and replay WAL segments to rebuild the in-memory index
4. WHEN closing a database, THE DB SHALL flush pending WAL writes, stop background workers, and release the file lock
5. WHEN a database is already open in another process, THE DB SHALL return an error indicating the lock is held
6. WHEN opening a database in read-only mode, THE DB SHALL prevent all write operations and return ErrReadOnly
7. THE DB SHALL complete startup in less than 1 second for databases containing 1 million keys

### Requirement 2: Core Key-Value Operations

**User Story:** As a developer, I want to perform basic CRUD operations on key-value pairs, so that I can store and retrieve data.

#### Acceptance Criteria

1. WHEN a developer calls Get with an existing key, THE DB SHALL return the associated value
2. WHEN a developer calls Get with a non-existent key, THE DB SHALL return ErrNotFound
3. WHEN a developer calls Set with a key and value, THE DB SHALL store the pair and make it immediately available for reads
4. WHEN a developer calls Set with an existing key, THE DB SHALL overwrite the previous value
5. WHEN a developer calls Delete with a key, THE DB SHALL remove the key and make subsequent Get calls return ErrNotFound
6. WHEN a developer calls Delete with a non-existent key, THE DB SHALL succeed without error
7. WHEN a developer calls Exists with a key, THE DB SHALL return true if the key exists and is not expired, false otherwise
8. THE DB SHALL enforce a maximum key size of 1024 bytes and return ErrKeyTooLarge for larger keys
9. THE DB SHALL enforce a maximum value size of 10 MB and return ErrValueTooLarge for larger values

### Requirement 3: Write-Ahead Logging and Durability

**User Story:** As a developer, I want all acknowledged writes to be durable, so that no data is lost in the event of a crash.

#### Acceptance Criteria

1. WHEN a write operation completes successfully, THE DB SHALL have appended the operation to the WAL
2. WHEN Sync_Mode is SyncAlways, THE DB SHALL call fsync after every write operation
3. WHEN Sync_Mode is SyncPeriodic, THE DB SHALL call fsync every 1 second via a background worker
4. WHEN Sync_Mode is SyncManual, THE DB SHALL only call fsync when the developer explicitly calls Sync
5. WHEN appending to the WAL, THE DB SHALL include a CRC32 checksum for each record
6. WHEN the WAL size exceeds MaxWALSize, THE DB SHALL trigger compaction
7. THE DB SHALL encode WAL records with record length, type, timestamp, key, value, expiration, and checksum

### Requirement 4: Crash Recovery

**User Story:** As a developer, I want the database to recover correctly after a crash, so that all acknowledged writes are preserved.

#### Acceptance Criteria

1. WHEN opening a database after a crash, THE DB SHALL load the latest valid snapshot
2. WHEN replaying WAL segments, THE DB SHALL verify the CRC32 checksum of each record
3. IF a WAL record has an invalid checksum, THEN THE DB SHALL stop replay at that point and truncate the WAL
4. WHEN replaying WAL segments, THE DB SHALL apply operations in the order they were written
5. WHEN replaying is complete, THE DB SHALL rebuild the in-memory index with all non-expired keys
6. THE DB SHALL ensure that replaying the same WAL segment multiple times produces the same final state (idempotent replay)
7. WHEN a snapshot file is corrupt, THE DB SHALL fall back to an older snapshot and replay all subsequent WAL segments

### Requirement 5: Snapshot and Compaction

**User Story:** As a developer, I want the database to periodically compact data, so that startup time remains fast and disk usage is controlled.

#### Acceptance Criteria

1. WHEN compaction is triggered, THE DB SHALL create a new snapshot file containing all non-expired keys in sorted order
2. WHEN writing a snapshot, THE DB SHALL include a magic number, format version, timestamp, and record count in the header
3. WHEN a snapshot is successfully written, THE DB SHALL update the MANIFEST file atomically
4. WHEN the MANIFEST is updated, THE DB SHALL delete WAL segments older than the snapshot
5. WHEN compaction is running, THE DB SHALL allow reads and writes to continue without blocking
6. THE DB SHALL trigger compaction automatically when WAL size exceeds MaxWALSize
7. WHEN a developer calls Compact manually, THE DB SHALL trigger compaction immediately
8. THE DB SHALL trigger compaction automatically based on SnapshotInterval if configured

### Requirement 6: Time-To-Live (TTL) Support

**User Story:** As a developer, I want to set expiration times on keys, so that temporary data is automatically cleaned up.

#### Acceptance Criteria

1. WHEN a developer calls SetWithTTL, THE DB SHALL store the key with an expiration timestamp
2. WHEN a developer calls Get on an expired key, THE DB SHALL return ErrNotFound and lazily delete the key
3. WHEN a developer calls TTL on a key with expiration, THE DB SHALL return the remaining duration
4. WHEN a developer calls TTL on a key without expiration, THE DB SHALL return -1
5. WHEN a developer calls TTL on a non-existent key, THE DB SHALL return ErrNotFound
6. WHEN a developer calls Expire on an existing key, THE DB SHALL set the expiration and return true
7. WHEN a developer calls Expire on a non-existent key, THE DB SHALL return false
8. WHEN a developer calls Persist on a key with expiration, THE DB SHALL remove the expiration and return true
9. WHEN a developer calls Persist on a non-existent key, THE DB SHALL return false
10. THE DB SHALL run a background worker every 1 second to clean up expired keys
11. WHEN creating a snapshot, THE DB SHALL exclude expired keys from the snapshot

### Requirement 7: Batch Operations

**User Story:** As a developer, I want to perform multiple write operations atomically, so that I can maintain data consistency.

#### Acceptance Criteria

1. WHEN a developer creates a batch and adds operations, THE Batch SHALL store operations in memory without applying them
2. WHEN a developer calls Write on a batch, THE DB SHALL apply all operations atomically
3. IF any operation in a batch fails, THEN THE DB SHALL roll back all operations in the batch
4. WHEN a batch is written, THE DB SHALL write all operations to the WAL in a single atomic write
5. WHEN a developer calls Discard on a batch, THE Batch SHALL abandon all pending operations
6. THE Batch SHALL enforce a maximum total size of 100 MB and return ErrBatchTooBig if exceeded
7. THE Batch SHALL support Set, SetWithTTL, and Delete operations

### Requirement 8: Iteration and Scanning

**User Story:** As a developer, I want to iterate over keys in sorted order, so that I can perform range queries and prefix scans.

#### Acceptance Criteria

1. WHEN a developer calls Scan with a prefix, THE DB SHALL return an iterator over keys matching that prefix
2. WHEN a developer calls ScanRange with start and end keys, THE DB SHALL return an iterator over keys in that range
3. WHEN iterating, THE Iterator SHALL return keys in lexicographic order
4. WHEN iterating, THE Iterator SHALL provide a snapshot-consistent view and exclude keys added after iteration started
5. WHEN iterating, THE Iterator SHALL skip expired keys
6. WHEN a developer calls Keys with a pattern, THE DB SHALL return all keys matching the glob pattern
7. WHEN a developer calls Count, THE DB SHALL return the total number of non-expired keys
8. WHEN an iterator is created with a limit, THE Iterator SHALL return at most that many keys

### Requirement 9: Extended Atomic Operations

**User Story:** As a developer, I want atomic operations like increment and compare-and-swap, so that I can implement counters and lock-free algorithms.

#### Acceptance Criteria

1. WHEN a developer calls SetNX with a non-existent key, THE DB SHALL set the value and return true
2. WHEN a developer calls SetNX with an existing key, THE DB SHALL not modify the value and return false
3. WHEN a developer calls Incr on a key containing an integer value, THE DB SHALL increment it by 1 and return the new value
4. WHEN a developer calls Incr on a non-existent key, THE DB SHALL create the key with value 1
5. WHEN a developer calls Incr on a key with a non-integer value, THE DB SHALL return ErrInvalidValue
6. WHEN a developer calls Decr on a key containing an integer value, THE DB SHALL decrement it by 1 and return the new value
7. WHEN a developer calls IncrBy with a delta, THE DB SHALL increment the value by delta and return the new value
8. WHEN a developer calls CompareAndSwap with matching old value, THE DB SHALL set the new value and return true
9. WHEN a developer calls CompareAndSwap with non-matching old value, THE DB SHALL not modify the value and return false
10. WHEN a developer calls GetAndSet, THE DB SHALL atomically set the new value and return the old value

### Requirement 10: Concurrency and Thread Safety

**User Story:** As a developer, I want all database operations to be goroutine-safe, so that I can safely use the database from multiple goroutines.

#### Acceptance Criteria

1. WHEN multiple goroutines call Get concurrently, THE DB SHALL handle all requests without data races
2. WHEN multiple goroutines call Set concurrently, THE DB SHALL serialize writes and ensure each write is atomic
3. WHEN a goroutine writes a key, THE same goroutine SHALL immediately see the new value on subsequent reads
4. WHEN a goroutine creates an iterator, THE Iterator SHALL provide a consistent snapshot even if other goroutines modify keys
5. THE DB SHALL use a single-writer, multiple-readers concurrency model
6. THE DB SHALL prevent deadlocks by enforcing a consistent lock hierarchy

### Requirement 11: Context Support and Cancellation

**User Story:** As a developer, I want to cancel long-running operations using context, so that I can implement timeouts and graceful shutdowns.

#### Acceptance Criteria

1. WHEN a developer calls GetWithContext with a cancelled context, THE DB SHALL return context.Canceled
2. WHEN a developer calls SetWithContext with a cancelled context, THE DB SHALL return context.Canceled
3. WHEN a context deadline is exceeded during an operation, THE DB SHALL return context.DeadlineExceeded
4. THE DB SHALL check context cancellation at appropriate points during long-running operations

### Requirement 12: Error Handling and Validation

**User Story:** As a developer, I want clear error messages for invalid operations, so that I can quickly diagnose and fix issues.

#### Acceptance Criteria

1. WHEN a key exceeds MaxKeySize, THE DB SHALL return ErrKeyTooLarge
2. WHEN a value exceeds MaxValueSize, THE DB SHALL return ErrValueTooLarge
3. WHEN a write is attempted on a read-only database, THE DB SHALL return ErrReadOnly
4. WHEN an operation is attempted on a closed database, THE DB SHALL return ErrClosed
5. WHEN a batch exceeds MaxBatchSize, THE Batch SHALL return ErrBatchTooBig
6. WHEN a WAL record has a corrupt checksum, THE DB SHALL return ErrCorruptWAL during recovery
7. WHEN disk space is exhausted during a write, THE DB SHALL return an appropriate I/O error

### Requirement 13: Observability and Metrics

**User Story:** As a developer, I want to monitor database performance and health, so that I can diagnose issues and optimize usage.

#### Acceptance Criteria

1. WHEN a developer calls Stats, THE DB SHALL return current key count, WAL size, snapshot count, and memory usage
2. WHEN a developer calls Stats, THE DB SHALL return operation counters for reads, writes, deletes, and scans
3. WHEN a developer calls Stats, THE DB SHALL return latency percentiles (P50, P95, P99) for read and write operations
4. WHEN metrics are enabled, THE DB SHALL expose Prometheus-compatible metrics
5. WHEN a developer calls DumpKeys, THE DB SHALL write all keys to the provided writer for debugging

### Requirement 14: Storage Format Specification

**User Story:** As a developer, I want well-defined storage formats, so that I can inspect and debug database files.

#### Acceptance Criteria

1. THE WAL SHALL encode each record with a 4-byte length prefix, 1-byte type, 8-byte timestamp, variable-length key and value, 8-byte expiration, and 4-byte CRC32 checksum
2. THE Snapshot SHALL begin with an 8-byte magic number "MINIKVSN", 4-byte format version, 8-byte timestamp, and 8-byte record count
3. THE Snapshot SHALL encode each record with variable-length key and value, and 8-byte expiration timestamp
4. THE MANIFEST SHALL track the current WAL sequence number, last snapshot sequence number, and lists of active WAL segments and snapshots
5. THE DB SHALL use little-endian byte order for all multi-byte integers
6. THE DB SHALL use varint encoding for key and value lengths to save space

### Requirement 15: Performance Targets

**User Story:** As a developer, I want predictable performance characteristics, so that I can design applications with confidence.

#### Acceptance Criteria

1. THE DB SHALL achieve at least 200,000 read operations per second for in-memory keys
2. THE DB SHALL achieve at least 50,000 write operations per second when using batched writes with SyncPeriodic mode
3. THE DB SHALL achieve P99 read latency under 100 microseconds
4. THE DB SHALL achieve P99 write latency under 5 milliseconds when using SyncAlways mode
5. THE DB SHALL use less than 100 bytes of memory overhead per key
6. THE DB SHALL complete startup in under 1 second for databases with 1 million keys
