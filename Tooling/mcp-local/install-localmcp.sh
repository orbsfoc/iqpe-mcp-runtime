#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
BIN_DIR="${HOME}/bin"
BIN_PATH="${BIN_DIR}/iqpe-localmcp"

mkdir -p "$BIN_DIR"

cd "$ROOT_DIR/Tooling/docflow"
go build -o "$BIN_PATH" ./cmd/localmcp

chmod +x "$BIN_PATH"

for mode in repo-read docflow-actions docs-graph policy; do
	if ! "$BIN_PATH" --server "$mode" --self-test >/dev/null 2>&1; then
		echo "self-test failed for mode: $mode" >&2
		exit 1
	fi
done

if [[ "$(uname -s)" == "Darwin" ]]; then
	if command -v xattr >/dev/null 2>&1; then
		xattr -d com.apple.quarantine "$BIN_PATH" >/dev/null 2>&1 || true
	fi
	if command -v codesign >/dev/null 2>&1; then
		codesign --force --sign - "$BIN_PATH" >/dev/null 2>&1 || true
	fi
fi

echo "installed: $BIN_PATH"
echo "next (mac): codesign --force --sign - \"$BIN_PATH\""
