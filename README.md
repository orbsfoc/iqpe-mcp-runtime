# iqpe-mcp-runtime

Go-based MCP runtime repository for server modes, action execution, and runtime contracts.

## Extracted package contents

- `Tooling/docflow` (Go CLI and runner logic)
- `Tooling/mcp-local` (local runtime setup helpers)
- `Tooling/contracts` (runtime contract assets)
- `docs/README.md` (runtime docs index)

## Owns

- MCP server binaries and runtime logic
- Action runner interfaces
- Runtime self-tests and diagnostics

## Language policy

- Tooling/runtime code is Go-first and Go-only by default.
- Non-Go runtime additions require explicit approved exception.

## Required standards

- Maintain CHANGELOG entries with runtime/test impact.
- Keep runtime contracts documented and versioned.

## CI and hosting

- GitHub Actions is the active CI host for early testing.
- Runtime checks should remain command-compatible across CI providers.
- Follow `CI-HOSTING-PORTABILITY.md` in this repo after extraction.

## Extraction provenance

- See `EXTRACTION-MANIFEST.md` for source mapping from monorepo paths.
