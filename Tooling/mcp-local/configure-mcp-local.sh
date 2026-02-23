#!/usr/bin/env bash
set -euo pipefail

TARGET_ROOT="${1:-$PWD}"
TARGET_ROOT="$(cd "$TARGET_ROOT" && pwd)"

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
if [[ -z "$BIN_PATH" ]]; then
  echo "ERROR: iqpe-localmcp not found on PATH or common local bin dirs." >&2
  echo "Install first: bash Tooling/mcp-local/install-localmcp.sh" >&2
  exit 1
fi

mkdir -p "$TARGET_ROOT/.vscode"

cat > "$TARGET_ROOT/.vscode/mcp.json" <<EOF
{
  "servers": {
    "repo-read-local": {
      "transport": "stdio",
      "command": "${BIN_PATH}",
      "args": ["--server", "repo-read"]
    },
    "docflow-actions-local": {
      "transport": "stdio",
      "command": "${BIN_PATH}",
      "args": ["--server", "docflow-actions"]
    },
    "docs-graph-local": {
      "transport": "stdio",
      "command": "${BIN_PATH}",
      "args": ["--server", "docs-graph"]
    },
    "policy-local": {
      "transport": "stdio",
      "command": "${BIN_PATH}",
      "args": ["--server", "policy"]
    }
  }
}
EOF

echo "Wrote MCP config: $TARGET_ROOT/.vscode/mcp.json"
echo "Using binary: $BIN_PATH"
