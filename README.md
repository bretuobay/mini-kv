# MiniKV

MiniKV is a pure-Go, embedded key-value database with WAL durability, snapshots, TTL, and simple atomic operations.

## Install

```bash
go get github.com/bretuobay/mini-kv
```

## Usage

```go
package main

import (
    "log"

    "github.com/bretuobay/mini-kv"
)

func main() {
    db, err := minikv.Open(minikv.DefaultOptions("./data"))
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()

    _ = db.Set([]byte("hello"), []byte("world"))
    value, _ := db.Get([]byte("hello"))
    log.Printf("%s", value)
}
```

## Configuration

```go
opts := minikv.DefaultOptions("./data")
opts.SyncMode = minikv.SyncPeriodic // SyncAlways | SyncManual
opts.ReadOnly = false
```

Defaults:
- `MaxKeySize`: 1024 bytes
- `MaxValueSize`: 10 MB
- `MaxBatchSize`: 100 MB
- `MaxWALSize`: 256 MB
- `SyncMode`: `SyncPeriodic`

## Errors

Common errors:
- `ErrNotFound`
- `ErrKeyTooLarge`, `ErrValueTooLarge`
- `ErrReadOnly`, `ErrClosed`, `ErrLocked`
- `ErrInvalidValue`

## API Highlights

- CRUD: `Get`, `GetInto`, `Set`, `Delete`, `Exists`
- TTL: `SetWithTTL`, `TTL`, `Expire`, `Persist`
- Iteration: `Scan`, `ScanRange`, `Keys`, `Count`
- Atomic: `SetNX`, `Incr`, `Decr`, `IncrBy`, `CompareAndSwap`, `GetAndSet`
- Batch: `NewBatch()` + `Batch.Write()`
- Observability: `Stats`, `DumpKeys`

## Benchmarks

```
go test ./benchmarks -bench . -run ^$
```

## Documentation

- `docs/architecture.md` — system design and background workers
- `docs/file_formats.md` — WAL/snapshot/MANIFEST formats
- `docs/migration_guide.md` — version notes

## Repository

Source: `https://github.com/bretuobay/mini-kv`

## License

MIT. See `LICENSE`.

## Contributing

See `CONTRIBUTING.md`.
