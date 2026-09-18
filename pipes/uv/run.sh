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
export PYTHONUNBUFFERED=1
export PYTHONFAULTHANDLER=1
# Do not inherit UV_INDEX_URL (Aliyun simple lags; e2e needs current PyPI argospy).
unset UV_INDEX_URL
echo "==> uv $(uv --version 2>/dev/null || true) project=${project}"
while IFS= read -r line; do
	[[ -z "$line" ]] && continue
	echo "==> run ${line}"
	uv run --default-index https://pypi.org/simple --project "$project" -- bash -c "$line"
done < <(each_item "$(input RUN)")
echo "OK uv"
