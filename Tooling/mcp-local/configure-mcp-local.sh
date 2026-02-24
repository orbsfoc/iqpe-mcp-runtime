#!/usr/bin/env bash
set -euo pipefail

TARGET_ROOT="${1:-$PWD}"
TARGET_ROOT="$(cd "$TARGET_ROOT" && pwd)"
TRANSPORT="${2:-stdio}"

if [[ "$TRANSPORT" != "stdio" && "$TRANSPORT" != "http" ]]; then
  echo "ERROR: invalid transport '$TRANSPORT' (expected stdio|http)" >&2
  echo "Usage: bash Tooling/mcp-local/configure-mcp-local.sh [target_root] [stdio|http]" >&2
  exit 1
fi

resolve_binary() {
  if command -v iqpe-localmcp >/dev/null 2>&1; then
    command -v iqpe-localmcp
    return 0
  fi

  local candidates=(
    "$HOME/bin/iqpe-localmcp"
    "$HOME/.local/bin/iqpe-localmcp"
  )

  local c
  for c in "${candidates[@]}"; do
    if [[ -x "$c" ]]; then
      echo "$c"
      return 0
    fi
  done

  return 1
}

BIN_PATH="$(resolve_binary || true)"
if [[ "$TRANSPORT" == "stdio" && -z "$BIN_PATH" ]]; then
  echo "ERROR: iqpe-localmcp not found on PATH or common local bin dirs." >&2
  echo "Install first: bash Tooling/mcp-local/install-localmcp.sh" >&2
  exit 1
fi

mkdir -p "$TARGET_ROOT/.vscode"

if [[ "$TRANSPORT" == "stdio" ]]; then
  cat > "$TARGET_ROOT/.vscode/mcp.json" <<EOF
{
  "servers": {
    "repo-read-local": {
      "transport": "stdio",
      "command": "${BIN_PATH}",
      "args": ["--server", "repo-read", "--workspace", "${TARGET_ROOT}"]
    },
    "docflow-actions-local": {
      "transport": "stdio",
      "command": "${BIN_PATH}",
      "args": ["--server", "docflow-actions", "--workspace", "${TARGET_ROOT}"]
    },
    "docs-graph-local": {
      "transport": "stdio",
      "command": "${BIN_PATH}",
      "args": ["--server", "docs-graph", "--workspace", "${TARGET_ROOT}"]
    },
    "policy-local": {
      "transport": "stdio",
      "command": "${BIN_PATH}",
      "args": ["--server", "policy", "--workspace", "${TARGET_ROOT}"]
    }
  }
}
EOF

  echo "Wrote MCP config: $TARGET_ROOT/.vscode/mcp.json"
  echo "Using binary: $BIN_PATH"
else
  cat > "$TARGET_ROOT/.vscode/mcp.json" <<EOF
{
  "servers": {
    "repo-read-local": {
      "transport": "http",
      "url": "http://127.0.0.1:18080"
    },
    "docflow-actions-local": {
      "transport": "http",
      "url": "http://127.0.0.1:18081"
    },
    "docs-graph-local": {
      "transport": "http",
      "url": "http://127.0.0.1:18082"
    },
    "policy-local": {
      "transport": "http",
      "url": "http://127.0.0.1:18083"
    }
  }
}
EOF

  echo "Wrote MCP config: $TARGET_ROOT/.vscode/mcp.json"
  echo "Configured HTTP endpoints on localhost ports 18080-18083"
fi
