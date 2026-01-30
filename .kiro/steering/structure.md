# Structure Steering: MiniKV

## Preferred Layout
```
minikv/
  cmd/
    minikv-cli/
      main.go
  internal/
    wal/
    snapshot/
    index/
    manifest/
    metrics/
  minikv.go
  batch.go
  iterator.go
  options.go
  errors.go
  db_test.go
examples/
benchmarks/
docs/
```

## Placement Rules
- Public API lives at repo root (e.g., `minikv.go`, `options.go`).
- Storage primitives belong in `internal/`.
- Keep file formats and serialization in `internal/wal` and `internal/snapshot`.
- Tests should sit alongside the code they validate.
