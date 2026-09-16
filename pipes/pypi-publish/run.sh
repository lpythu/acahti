#!/usr/bin/env bash
set -euo pipefail
# shellcheck source=../_lib.sh
source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/_lib.sh"

require_input REGISTRY
cd "$ROOT"
user="${ACAHTI_USER:-}"
token="${ACAHTI_TOKEN:-}"
if [[ -z "$token" || -z "$user" ]]; then
	echo "error: job Acahti identity is required" >&2
	exit 1
fi
command -v uv >/dev/null || {
	echo "error: uv required on the Runner" >&2
	exit 1
}
rm -rf dist
uv build
echo "==> uv publish $(input REGISTRY)"
UV_PUBLISH_USERNAME="$user" UV_PUBLISH_PASSWORD="$token" uv publish --publish-url "$(input REGISTRY)" dist/*
echo "OK pypi-publish $(input REGISTRY)"
