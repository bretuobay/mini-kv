# Contributing

Thanks for helping improve MiniKV!

## Development Setup
- Go 1.21+
- Run tests: `go test ./...`
- Benchmarks: `go test ./benchmarks -bench . -run ^$`

## Guidelines
- Keep changes small and focused.
- Follow the existing package structure and file formats.
- Add tests for new behavior (unit, property, or integration).
- Avoid new dependencies for core storage logic unless approved.

## Submitting Changes
1. Create a focused branch.
2. Add/adjust tests.
3. Ensure `go test ./...` is green.
4. Open a PR with a clear description and rationale.

## Reporting Issues
- Include reproduction steps and expected behavior.
- Attach logs or failing tests when possible.

