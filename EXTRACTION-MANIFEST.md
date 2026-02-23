# Extraction Manifest

## Scope

This package materializes the `iqpe-mcp-runtime` extraction target.

## Source mapping

- Source: `Tooling/docflow`
  - Target: `Tooling/docflow`
- Source: `Tooling/mcp-local`
  - Target: `Tooling/mcp-local`
- Source: `Tooling/contracts`
  - Target: `Tooling/contracts`

## Integrity rules

- Preserve Go module/test behavior for `Tooling/docflow`.
- Preserve MCP runtime contract paths and naming.
- Record runtime changes in `CHANGELOG.md` with test impact.
