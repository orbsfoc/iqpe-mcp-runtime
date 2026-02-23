#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
BIN_DIR="${HOME}/bin"
BIN_PATH="${BIN_DIR}/iqpe-localmcp"

mkdir -p "$BIN_DIR"

cd "$ROOT_DIR/Tooling/docflow"
go build -o "$BIN_PATH" ./cmd/localmcp

chmod +x "$BIN_PATH"
echo "installed: $BIN_PATH"
echo "next (mac): codesign --force --sign - \"$BIN_PATH\""
