#!/usr/bin/env bash
set -euo pipefail
# shellcheck source=../_lib.sh
source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/_lib.sh"

require_input RUN
cd "$ROOT"
project="$(input PROJECT)"
project="${project:-.}"
if [[ ! -f "${project}/pyproject.toml" ]]; then
	echo "error: ${project}/pyproject.toml missing" >&2
	exit 1
fi
command -v uv >/dev/null || pip install --no-cache-dir uv
venv="$(mktemp -d)/venv"
export UV_PROJECT_ENVIRONMENT="$venv"
echo "==> uv $(uv --version 2>/dev/null || true) project=${project}"
while IFS= read -r line; do
	[[ -z "$line" ]] && continue
	uv run --project "$project" -- bash -c "$line"
done < <(each_item "$(input RUN)")
echo "OK uv"
