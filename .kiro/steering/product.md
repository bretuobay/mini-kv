# Product Steering: MiniKV

## Purpose
MiniKV is an embedded, single-node key-value database written in Go for local persistence with predictable performance and clear, educational internals.

## Target Users
- Go developers needing local storage without a server
- Students learning database internals
- Internal projects like MiniSearchDB and FYI Notes

## Core Capabilities
- Basic CRUD: Get/Set/Delete/Exists
- Durability: WAL-based writes with configurable sync
- Recovery: snapshots + WAL replay
- TTL: expirations with lazy cleanup and background sweeper
- Iteration: prefix/range scans with snapshot consistency
- Atomic ops: SetNX, Incr/Decr, IncrBy, CompareAndSwap
- Batch writes with atomic commit

## Non-Goals
- Distributed or replicated operation
- Complex data structures (lists/sets/etc.)
- Built-in auth, encryption, or network protocol

## Success Targets (v1.0)
- 50K+ writes/sec (batched), 200K+ reads/sec
- <1s startup for ~1M keys
- Zero data loss after acknowledged writes
