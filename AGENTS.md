# Agent Instructions (Codex)

## Project
MiniKV is a pure-Go, embedded key-value database focused on correctness, durability, and educational clarity.

## Sources of Truth
- `PRD_GOLANG.md` (product goals, package structure, performance targets)
- `SPECIFICATION.md` (language-agnostic API and behavior)
- `.kiro/specs/minikv-embedded-database/*` (requirements, design, tasks)

## Technical Constraints
- Go 1.21+; standard library only unless explicitly approved.
- No CGO, no external services, no networked mode.
- Single-node, single-writer, multiple readers with snapshot-consistent reads.
- Durable storage via WAL + snapshots; crash-safe recovery.
- TTL support with lazy deletion + periodic cleanup.

## Implementation Guidance
- Follow the package structure in `PRD_GOLANG.md` (Section 13).
- Keep APIs small and predictable; prefer clear errors over panics.
- Preserve deterministic file formats (WAL, snapshot, MANIFEST).
- Use CRC32 for record integrity as specified.

## Testing
- Add unit tests for correctness and edge cases.
- Default test command: `go test ./...`

## When Unsure
- Ask for clarification rather than guessing on file formats or API shapes.
