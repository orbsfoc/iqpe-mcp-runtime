# Local MCP Instance Scaffold

This folder defines the local MCP hosting scaffold for VS Code.

## Current mode
- Transport: `stdio` (local process)
- VS Code config: `.vscode/mcp.json`
- Copy-ready template: `Tooling/mcp-local/mcp.example.json`
- Local server binary: `iqpe-localmcp` (must be on user `PATH`)
- Build/install script: `Tooling/mcp-local/install-localmcp.sh`
- Use direct binary command in MCP config (avoid `bash -lc` wrappers for stdio servers).

## Install and sign (mac-friendly)
- `bash Tooling/mcp-local/install-localmcp.sh`
- `codesign --force --sign - "$(command -v iqpe-localmcp)"`

## Quick restore
- `bash Tooling/mcp-local/configure-mcp-local.sh <target_repo_root>`
- Reload VS Code window after copying.

## If initialize hangs
- Confirm binary is on PATH: `command -v iqpe-localmcp`
- Confirm mode self-test: `iqpe-localmcp --server repo-read --self-test`
- Regenerate MCP config with resolved binary path: `bash Tooling/mcp-local/configure-mcp-local.sh <target_repo_root>`
- Confirm initialize response manually (any mode) before VS Code restart.
- Reload VS Code window to restart MCP processes after config changes.

## Planned mode
- Transport: `http`
- Keep method/action contracts stable while changing transport and auth model.
- Convert each server to hosted HTTP entrypoint while preserving tool names:
	- `list_dir`, `read_file`, `grep_search`
	- `list_actions`, `run_action`, `run_script`
	- `queryImpacts`, `getLatestApproved`
	- `validateOwnership`, `checkNonCloningRule`

## Required source data roots to expose
- `Docs/ConcretePOCProduct/`
- `Docs/RefactoredProductDocs/00-governance/`
- `Docs/RefactoredProductDocs/artifacts/`
- `Docs/RefactoredTechnicalDocs/00-architecture/`
- `Docs/RefactoredTechnicalDocs/01-implementation/`
- `Docs/RefactoredTechnicalDocs/02-operations/`
- `Tooling/docflow/`
- `Tooling/agent-tools/`
- `POC/services/`
- `POC/frontend/`

## Notes
- Keep server permissions read-only by default; only action server should execute allowed agent tools/actions.

## Self-test
- `iqpe-localmcp --server repo-read --self-test`
- `iqpe-localmcp --server docflow-actions --self-test`
- `iqpe-localmcp --server docs-graph --self-test`
- `iqpe-localmcp --server policy --self-test`
