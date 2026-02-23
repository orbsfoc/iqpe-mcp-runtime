# CI Hosting Portability (MCP Runtime Repo)

## Current host

GitHub Actions is used for early testing.

## Command contract (host-neutral)

Every CI host must preserve these checks:

1. Required baseline docs exist (`README.md`, `CHANGELOG.md`, `OWNERS.md`, `docs/README.md`).
2. Extracted runtime content exists (`Tooling/docflow/go.mod`, `Tooling/mcp-local/README.md`).
3. Runtime Go tests execute: `cd Tooling/docflow && go test ./...`.

## Migration rule

When moving away from GitHub, port checks with identical command semantics first, then optimize.
