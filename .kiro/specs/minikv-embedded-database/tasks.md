# Implementation Plan: MiniKV Embedded Database

## Overview

This implementation plan breaks down the MiniKV embedded database into discrete, incremental tasks. The approach follows a bottom-up strategy: building core storage primitives first, then adding durability features, and finally implementing advanced operations. Each task builds on previous work, with property-based tests integrated throughout to catch errors early.

## Tasks

- [x] 1. Project Setup and Core Types
  - Create Go module structure with `go.mod`
  - Define core types: `DB`, `Options`, `Entry`, `MemIndex`
  - Define error constants: `ErrNotFound`, `ErrKeyTooLarge`, `ErrValueTooLarge`, etc.
  - Define size limits: `MaxKeySize`, `MaxValueSize`, `MaxBatchSize`
  - Set up testing framework with `gopter` for property-based testing
  - _Requirements: 2.8, 2.9, 12.1, 12.2_

- [ ] 2. In-Memory Index Implementation
  - [x] 2.1 Implement `MemIndex` with HashMap backing
    - Implement `Set(key, value, expiresAt)` method
    - Implement `Get(key)` method with expiration checking
    - Implement `Delete(key)` method
    - Implement `Exists(key)` method
    - Implement `Count()` method for non-expired keys
    - Add memory size tracking
    - _Requirements: 2.1, 2.2, 2.3, 2.4, 2.5, 2.6, 2.7, 8.7_
  
  - [x]* 2.2 Write property test for Set-Get round trip
    - **Property 1: Set-Get Round Trip**
    - **Validates: Requirements 2.1, 2.3, 2.4**
  
  - [x]* 2.3 Write property test for Get non-existent key
    - **Property 2: Get Non-Existent Key**
    - **Validates: Requirements 2.2**
  
  - [x]* 2.4 Write property test for Delete
    - **Property 3: Delete Makes Key Non-Existent**
    - **Property 4: Delete Idempotence**
    - **Validates: Requirements 2.5, 2.6**
  
  - [x]* 2.5 Write property test for Exists
    - **Property 5: Exists Reflects Key State**
    - **Validates: Requirements 2.7**

- [ ] 3. WAL Format and Encoding
  - [x] 3.1 Implement WAL record encoding
    - Define `WALRecord` struct with Type, Timestamp, Key, Value, ExpiresAt
    - Implement `EncodeWALRecord(record)` with varint encoding for lengths
    - Implement `DecodeWALRecord(data)` with checksum verification
    - Use little-endian byte order for all multi-byte integers
    - Calculate CRC32 checksum using IEEE polynomial
    - _Requirements: 3.1, 3.5, 3.7, 14.1_
  
  - [x]* 3.2 Write property test for WAL encoding round trip
    - **Property 13: WAL Encoding Round Trip**
    - **Validates: Requirements 3.1, 3.5, 3.7, 14.1**

- [ ] 4. WAL Manager Implementation
  - [x] 4.1 Implement WAL file management
    - Create `WALManager` struct with file handle, sequence number, size tracking
    - Implement `Open(dir)` to create or open WAL directory
    - Implement `AppendRecord(record)` to write encoded records
    - Implement `Sync()` to fsync the WAL file
    - Implement `Close()` to close WAL file
    - Handle WAL segment rotation when size exceeds threshold
    - _Requirements: 3.1, 3.2, 3.4, 3.6_
  
  - [x] 4.2 Implement WAL reader for recovery
    - Implement `ReadWAL(path)` to read and decode WAL records
    - Verify CRC32 checksum for each record
    - Stop reading at first corrupt record
    - Return all valid records in order
    - _Requirements: 4.2, 4.3, 4.4_
  
  - [x]* 4.3 Write property test for WAL replay idempotence
    - **Property 16: WAL Replay Idempotence**
    - **Validates: Requirements 4.6**
  
  - [x]* 4.4 Write unit test for checksum corruption detection
    - **Property 15: Checksum Detects Corruption**
    - **Validates: Requirements 4.2, 4.3**

- [ ] 5. Snapshot Format and Manager
  - [x] 5.1 Implement snapshot encoding
    - Define snapshot header with magic number "MINIKVSN", version, timestamp, count
    - Implement `EncodeSnapshot(entries)` to write sorted key-value pairs
    - Implement `DecodeSnapshot(path)` to read snapshot file
    - Calculate footer CRC32 for integrity verification
    - _Requirements: 5.1, 5.2, 14.2_
  
  - [x] 5.2 Implement `SnapshotManager`
    - Implement `CreateSnapshot(index, path)` to write snapshot from in-memory index
    - Exclude expired keys during snapshot creation
    - Write keys in lexicographic order
    - Implement `LoadSnapshot(path)` to read snapshot into memory
    - _Requirements: 5.1, 6.11_
  
  - [x]* 5.3 Write property test for snapshot round trip
    - **Property 18: Snapshot Format Round Trip**
    - **Validates: Requirements 5.2, 14.2**
  
  - [x]* 5.4 Write property test for snapshot contains all non-expired keys
    - **Property 17: Snapshot Contains All Non-Expired Keys**
    - **Validates: Requirements 5.1, 6.11**

- [ ] 6. MANIFEST File Management
  - [x] 6.1 Implement MANIFEST format
    - Define `Manifest` struct with WAL sequence, snapshot sequence, segment lists
    - Implement `WriteManifest(path, manifest)` using atomic write-then-rename
    - Implement `ReadManifest(path)` to load MANIFEST
    - Use YAML-like text format for human readability
    - _Requirements: 5.3, 14.4_
  
  - [x]* 6.2 Write property test for MANIFEST tracking
    - **Property 51: MANIFEST Tracks State**
    - **Validates: Requirements 14.4**

- [ ] 7. Database Lifecycle Operations
  - [x] 7.1 Implement database Open
    - Implement `Open(opts Options)` function
    - Create directory structure if it doesn't exist
    - Acquire exclusive file lock using `flock`
    - Read MANIFEST to determine recovery strategy
    - Load latest snapshot if it exists
    - Replay WAL segments in order
    - Build in-memory index from snapshot and WAL
    - _Requirements: 1.1, 1.2, 1.3, 4.1, 4.4, 4.5_
  
  - [x] 7.2 Implement database Close
    - Flush pending WAL writes
    - Stop background workers (sync, TTL cleanup)
    - Release file lock
    - Close all file handles
    - _Requirements: 1.4_
  
  - [x]* 7.3 Write property test for Open-Close-Reopen
    - **Property 10: Open-Close-Reopen Preserves Data**
    - **Validates: Requirements 1.3, 4.1, 4.4, 4.5**
  
  - [x]* 7.4 Write property test for exclusive file lock
    - **Property 9: Exclusive File Lock**
    - **Validates: Requirements 1.2, 1.5**
  
  - [x]* 7.5 Write property test for Close releases lock
    - **Property 11: Close Releases Lock**
    - **Validates: Requirements 1.4**

- [ ] 8. Core CRUD Operations
  - [x] 8.1 Implement Get operation
    - Implement `Get(key)` method
    - Acquire read lock on index
    - Check if key exists and is not expired
    - Return value or ErrNotFound
    - Validate key size
    - _Requirements: 2.1, 2.2, 2.8_
  
  - [x] 8.2 Implement Set operation
    - Implement `Set(key, value)` method
    - Validate key and value sizes
    - Acquire write lock
    - Encode and append WAL record
    - Sync if SyncMode is SyncAlways
    - Update in-memory index
    - _Requirements: 2.3, 2.4, 2.8, 2.9, 3.1, 3.2_
  
  - [x] 8.3 Implement Delete operation
    - Implement `Delete(key)` method
    - Acquire write lock
    - Append DELETE record to WAL
    - Remove from in-memory index
    - Succeed even if key doesn't exist
    - _Requirements: 2.5, 2.6_
  
  - [x] 8.4 Implement Exists operation
    - Implement `Exists(key)` method
    - Check index and verify not expired
    - _Requirements: 2.7_
  
  - [x]* 8.5 Write property tests for CRUD operations
    - **Property 1: Set-Get Round Trip**
    - **Property 2: Get Non-Existent Key**
    - **Property 3: Delete Makes Key Non-Existent**
    - **Property 4: Delete Idempotence**
    - **Property 5: Exists Reflects Key State**
    - **Property 6: Key Size Validation**
    - **Property 7: Value Size Validation**
    - **Validates: Requirements 2.1-2.9**

- [ ] 9. Checkpoint - Ensure all tests pass
  - Ensure all tests pass, ask the user if questions arise.

- [ ] 10. TTL Support
  - [x] 10.1 Implement TTL operations
    - Implement `SetWithTTL(key, value, ttl)` method
    - Calculate expiration timestamp from TTL duration
    - Store expiration in WAL record and index
    - Implement `TTL(key)` to return remaining duration
    - Implement `Expire(key, ttl)` to set expiration on existing key
    - Implement `Persist(key)` to remove expiration
    - _Requirements: 6.1, 6.3, 6.4, 6.5, 6.6, 6.7, 6.8, 6.9_
  
  - [ ] 10.2 Implement lazy TTL deletion
    - Check expiration in `Get` and delete if expired
    - Check expiration in `Exists`
    - Check expiration during iteration
    - _Requirements: 6.2, 8.5_
  
  - [ ] 10.3 Implement background TTL cleaner
    - Create background goroutine that runs every 1 second
    - Scan subset of keys for expired entries
    - Delete expired keys
    - Limit cleanup time to 10ms per iteration
    - _Requirements: 6.10_
  
  - [ ]* 10.4 Write property test for TTL expiration
    - **Property 22: TTL Expiration**
    - **Property 23: TTL Query Returns Remaining Duration**
    - **Validates: Requirements 6.1, 6.2, 6.3**
  
  - [ ]* 10.5 Write property test for Expire and Persist
    - **Property 24: Expire Sets Expiration**
    - **Property 25: Persist Removes Expiration**
    - **Validates: Requirements 6.6, 6.8**

- [ ] 11. Batch Operations
  - [ ] 11.1 Implement Batch interface
    - Create `batchImpl` struct with operation buffer
    - Implement `Set(key, value)` to buffer operation
    - Implement `SetWithTTL(key, value, ttl)` to buffer operation
    - Implement `Delete(key)` to buffer operation
    - Track total batch size
    - Implement `Size()` to return current batch size
    - Implement `Discard()` to abandon operations
    - _Requirements: 7.1, 7.5, 7.6, 7.7_
  
  - [ ] 11.2 Implement atomic batch Write
    - Implement `Write()` method
    - Validate batch size doesn't exceed MaxBatchSize
    - Acquire write lock
    - Write BATCH_START record to WAL
    - Write all operations to WAL
    - Write BATCH_END record to WAL
    - Sync if needed
    - Apply all operations to index atomically
    - Rollback on any error
    - _Requirements: 7.2, 7.3, 7.4, 7.6_
  
  - [ ]* 11.3 Write property test for batch atomicity
    - **Property 27: Batch Write Is Atomic**
    - **Validates: Requirements 7.2, 7.3, 7.4**
  
  - [ ]* 11.4 Write property test for batch buffering
    - **Property 26: Batch Operations Are Buffered**
    - **Property 28: Batch Discard Abandons Operations**
    - **Validates: Requirements 7.1, 7.5**
  
  - [ ]* 11.5 Write property test for batch size validation
    - **Property 29: Batch Size Validation**
    - **Validates: Requirements 7.6**

- [ ] 12. Iteration and Scanning
  - [ ] 12.1 Implement Iterator interface
    - Create `iteratorImpl` struct with snapshot of keys
    - Implement `Next()` to advance to next key
    - Implement `Key()` to return current key
    - Implement `Value()` to return current value
    - Implement `Error()` to return any error
    - Implement `Close()` to release resources
    - _Requirements: 8.1, 8.2, 8.3, 8.4_
  
  - [ ] 12.2 Implement Scan operations
    - Implement `Scan(prefix, limit)` to create prefix iterator
    - Implement `ScanRange(start, end, limit)` to create range iterator
    - Create snapshot of index at iterator creation time
    - Filter keys by prefix or range
    - Sort keys lexicographically
    - Skip expired keys
    - Apply limit if specified
    - _Requirements: 8.1, 8.2, 8.3, 8.4, 8.5, 8.8_
  
  - [ ] 12.3 Implement Keys and Count
    - Implement `Keys(pattern)` using glob pattern matching
    - Implement `Count()` to return non-expired key count
    - _Requirements: 8.6, 8.7_
  
  - [ ]* 12.4 Write property test for scan prefix filtering
    - **Property 30: Scan Prefix Filtering**
    - **Validates: Requirements 8.1**
  
  - [ ]* 12.5 Write property test for scan range filtering
    - **Property 31: ScanRange Filtering**
    - **Validates: Requirements 8.2**
  
  - [ ]* 12.6 Write property test for iterator ordering
    - **Property 32: Iterator Returns Sorted Keys**
    - **Validates: Requirements 8.3**
  
  - [ ]* 12.7 Write property test for iterator snapshot isolation
    - **Property 33: Iterator Snapshot Isolation**
    - **Validates: Requirements 8.4, 10.4**
  
  - [ ]* 12.8 Write property test for iterator TTL filtering
    - **Property 34: Iterator Skips Expired Keys**
    - **Validates: Requirements 8.5**
  
  - [ ]* 12.9 Write property test for Keys pattern matching
    - **Property 35: Keys Pattern Matching**
    - **Validates: Requirements 8.6**
  
  - [ ]* 12.10 Write property test for Count
    - **Property 36: Count Returns Non-Expired Key Count**
    - **Validates: Requirements 8.7**
  
  - [ ]* 12.11 Write property test for iterator limit
    - **Property 37: Iterator Limit**
    - **Validates: Requirements 8.8**

- [ ] 13. Checkpoint - Ensure all tests pass
  - Ensure all tests pass, ask the user if questions arise.

- [ ] 14. Extended Atomic Operations
  - [ ] 14.1 Implement SetNX
    - Implement `SetNX(key, value)` method
    - Check if key exists in index
    - If not exists, call Set and return true
    - If exists, return false without modifying
    - _Requirements: 9.1, 9.2_
  
  - [ ] 14.2 Implement integer operations
    - Implement `Incr(key)` to increment by 1
    - Implement `Decr(key)` to decrement by 1
    - Implement `IncrBy(key, delta)` to increment by delta
    - Parse value as int64, return ErrInvalidValue if not integer
    - Create key with value 1 if doesn't exist (for Incr)
    - _Requirements: 9.3, 9.4, 9.5, 9.6, 9.7_
  
  - [ ] 14.3 Implement CompareAndSwap
    - Implement `CompareAndSwap(key, old, new)` method
    - Acquire write lock
    - Get current value
    - Compare with old value
    - If match, set new value and return true
    - If no match, return false
    - _Requirements: 9.8, 9.9_
  
  - [ ] 14.4 Implement GetAndSet
    - Implement `GetAndSet(key, value)` method
    - Acquire write lock
    - Get current value
    - Set new value
    - Return old value
    - _Requirements: 9.10_
  
  - [ ]* 14.5 Write property test for SetNX
    - **Property 38: SetNX Conditional Set**
    - **Validates: Requirements 9.1, 9.2**
  
  - [ ]* 14.6 Write property test for integer operations
    - **Property 39: Integer Operations**
    - **Property 40: Incr Non-Integer Error**
    - **Validates: Requirements 9.3, 9.5, 9.6, 9.7**
  
  - [ ]* 14.7 Write property test for CompareAndSwap
    - **Property 41: CompareAndSwap Success**
    - **Property 42: CompareAndSwap Failure**
    - **Validates: Requirements 9.8, 9.9**
  
  - [ ]* 14.8 Write property test for GetAndSet
    - **Property 43: GetAndSet Atomic Swap**
    - **Validates: Requirements 9.10**

- [ ] 15. Compaction Implementation
  - [ ] 15.1 Implement compaction trigger logic
    - Monitor WAL size after each write
    - Trigger compaction when size exceeds MaxWALSize
    - Implement `Compact()` method for manual compaction
    - _Requirements: 3.6, 5.6, 5.7_
  
  - [ ] 15.2 Implement compaction process
    - Create new snapshot with incremented sequence number
    - Write all non-expired keys to snapshot in sorted order
    - Fsync snapshot file
    - Update MANIFEST atomically
    - Delete WAL segments older than snapshot
    - Fsync MANIFEST
    - Run compaction in background goroutine
    - _Requirements: 5.1, 5.2, 5.3, 5.4, 5.5_
  
  - [ ]* 15.3 Write property test for compaction triggers
    - **Property 14: Compaction Triggers on WAL Size**
    - **Validates: Requirements 3.6**
  
  - [ ]* 15.4 Write property test for compaction updates MANIFEST
    - **Property 19: Compaction Updates MANIFEST**
    - **Validates: Requirements 5.3**
  
  - [ ]* 15.5 Write property test for compaction deletes old WAL
    - **Property 20: Compaction Deletes Old WAL**
    - **Validates: Requirements 5.4**
  
  - [ ]* 15.6 Write property test for compaction concurrency
    - **Property 21: Compaction Allows Concurrent Operations**
    - **Validates: Requirements 5.5**

- [ ] 16. Concurrency and Thread Safety
  - [ ] 16.1 Implement locking strategy
    - Use `sync.RWMutex` for index access
    - Use `sync.Mutex` for WAL writes
    - Ensure consistent lock hierarchy to prevent deadlocks
    - _Requirements: 10.1, 10.2, 10.5, 10.6_
  
  - [ ] 16.2 Implement read-your-writes guarantee
    - Ensure writes are visible immediately in same goroutine
    - Use memory barriers appropriately
    - _Requirements: 10.3_
  
  - [ ]* 16.3 Write property test for concurrent reads
    - **Property 44: Concurrent Reads Are Safe**
    - **Validates: Requirements 10.1**
  
  - [ ]* 16.4 Write property test for concurrent writes
    - **Property 45: Concurrent Writes Are Serialized**
    - **Validates: Requirements 10.2**
  
  - [ ]* 16.5 Write property test for read-your-writes
    - **Property 46: Read-Your-Writes**
    - **Validates: Requirements 10.3**

- [ ] 17. Context Support
  - [ ] 17.1 Implement context-aware operations
    - Implement `GetWithContext(ctx, key)` method
    - Implement `SetWithContext(ctx, key, value)` method
    - Check context cancellation before acquiring locks
    - Check context cancellation during long operations
    - Return context.Canceled or context.DeadlineExceeded appropriately
    - _Requirements: 11.1, 11.2, 11.3, 11.4_
  
  - [ ]* 17.2 Write property test for context cancellation
    - **Property 47: Context Cancellation**
    - **Validates: Requirements 11.1, 11.2, 11.3**

- [ ] 18. Read-Only Mode and Error Handling
  - [ ] 18.1 Implement read-only mode
    - Add `ReadOnly` flag to Options
    - Check flag before all write operations
    - Return ErrReadOnly for write attempts
    - _Requirements: 1.6, 12.3_
  
  - [ ] 18.2 Implement closed state handling
    - Add `closed` flag to DB struct
    - Check flag at start of all operations
    - Return ErrClosed if database is closed
    - _Requirements: 12.4_
  
  - [ ]* 18.3 Write property test for read-only mode
    - **Property 12: Read-Only Mode Prevents Writes**
    - **Validates: Requirements 1.6**
  
  - [ ]* 18.4 Write property test for closed database
    - **Property 48: Closed Database Returns Error**
    - **Validates: Requirements 12.4**

- [ ] 19. Observability and Metrics
  - [ ] 19.1 Implement Stats collection
    - Create `Stats` struct with counters and gauges
    - Track key count, WAL size, snapshot count, memory usage
    - Track operation counters (reads, writes, deletes, scans)
    - Track latency histograms (P50, P95, P99)
    - Implement `Stats()` method to return current stats
    - _Requirements: 13.1, 13.2, 13.3_
  
  - [ ] 19.2 Implement DumpKeys for debugging
    - Implement `DumpKeys(w io.Writer)` method
    - Write all non-expired keys to writer
    - Include key, value length, expiration info
    - _Requirements: 13.5_
  
  - [ ]* 19.3 Write property test for Stats
    - **Property 49: Stats Reflects Database State**
    - **Validates: Requirements 13.1, 13.2**
  
  - [ ]* 19.4 Write property test for DumpKeys
    - **Property 50: DumpKeys Writes All Keys**
    - **Validates: Requirements 13.5**

- [ ] 20. Background Workers
  - [ ] 20.1 Implement periodic sync worker
    - Create goroutine that runs when SyncMode is SyncPeriodic
    - Call fsync every 1 second
    - Stop on database close
    - _Requirements: 3.3_
  
  - [ ] 20.2 Implement TTL cleanup worker
    - Create goroutine that runs every 1 second
    - Scan subset of keys for expired entries
    - Delete expired keys
    - Limit cleanup time to 10ms per iteration
    - Stop on database close
    - _Requirements: 6.10_
  
  - [ ] 20.3 Implement graceful shutdown
    - Stop all background workers on Close
    - Wait for workers to finish using sync.WaitGroup
    - Ensure no goroutine leaks
    - _Requirements: 1.4_

- [ ] 21. Checkpoint - Ensure all tests pass
  - Ensure all tests pass, ask the user if questions arise.

- [ ] 22. Integration Testing
  - [ ]* 22.1 Write integration test for full lifecycle
    - Test open → write → close → reopen cycle
    - Verify all data persists across restarts
    - Test with various data sizes and patterns
  
  - [ ]* 22.2 Write integration test for crash recovery
    - Simulate crash by killing process without Close
    - Reopen database and verify data integrity
    - Test with different crash points
  
  - [ ]* 22.3 Write integration test for concurrent operations
    - Run multiple goroutines performing reads and writes
    - Verify no data races using race detector
    - Verify all operations complete correctly
  
  - [ ]* 22.4 Write integration test for large dataset
    - Write 1 million keys
    - Verify startup time is under 1 second
    - Verify memory usage is reasonable
    - Verify all operations still work correctly

- [ ] 23. Benchmarking and Performance Tuning
  - [ ]* 23.1 Write benchmarks for core operations
    - Benchmark Get operation (target: 200K ops/sec)
    - Benchmark Set operation (target: 50K ops/sec batched)
    - Benchmark Delete operation
    - Benchmark Scan operation
    - Benchmark batch writes
  
  - [ ]* 23.2 Write benchmarks for startup time
    - Benchmark startup with 10K, 100K, 1M keys
    - Verify 1M keys loads in under 1 second
  
  - [ ]* 23.3 Profile memory usage
    - Measure memory overhead per key
    - Verify under 100 bytes per key
    - Identify and fix memory leaks
  
  - [ ]* 23.4 Optimize hot paths
    - Profile CPU usage
    - Optimize encoding/decoding
    - Optimize index lookups
    - Reduce allocations in hot paths

- [ ] 24. Documentation and Examples
  - [ ]* 24.1 Write API documentation
    - Document all public types and methods
    - Include usage examples
    - Document error conditions
    - Document performance characteristics
  
  - [ ]* 24.2 Write architecture documentation
    - Document storage format
    - Document recovery process
    - Document compaction strategy
    - Include diagrams
  
  - [ ]* 24.3 Create example applications
    - Basic usage example
    - Caching example with TTL
    - Batch operations example
    - Integration with MiniSearchDB example

- [ ] 25. Final Checkpoint - Ensure all tests pass
  - Run all unit tests
  - Run all property tests with 1000 iterations
  - Run all integration tests
  - Run all benchmarks and verify performance targets
  - Verify 85% test coverage
  - Run with race detector
  - Fix any remaining issues

## Notes

- Tasks marked with `*` are optional and can be skipped for faster MVP
- Each task references specific requirements for traceability
- Checkpoints ensure incremental validation
- Property tests validate universal correctness properties
- Unit tests validate specific examples and edge cases
- Integration tests validate end-to-end functionality
- Benchmarks validate performance targets
- All property tests should run with minimum 100 iterations
- Use `gopter` library for property-based testing in Go
- Tag each property test with: `// Feature: minikv-embedded-database, Property N: [property text]`
