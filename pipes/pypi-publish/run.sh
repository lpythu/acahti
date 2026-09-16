#!/usr/bin/env bash
set -euo pipefail
# shellcheck source=../_lib.sh
source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/_lib.sh"

require_input REGISTRY
pkg_path="$(input PATH)"
pkg_path="${pkg_path:-.}"
if [[ ! -f "${ROOT}/${pkg_path}/pyproject.toml" ]]; then
	echo "error: ${pkg_path}/pyproject.toml missing" >&2
	exit 1
fi

user="${ACAHTI_USER:-}"
token="${ACAHTI_TOKEN:-}"
if [[ -z "$token" || -z "$user" ]]; then
	echo "error: job Acahti identity is required" >&2
	exit 1
fi

registry="$(input REGISTRY)"
image="$(input IMAGE)"
image="${image:-python:3.12-slim-bookworm}"

index_env="$(
	ACAHTI_USER="$user" ACAHTI_TOKEN="$token" python3 - "$ROOT/$pkg_path/pyproject.toml" <<'PY'
import os, sys, tomllib
from pathlib import Path

data = tomllib.loads(Path(sys.argv[1]).read_text())
indexes = data.get("tool", {}).get("uv", {}).get("index", [])
if isinstance(indexes, dict):
    indexes = [indexes]
user, token = os.environ["ACAHTI_USER"], os.environ["ACAHTI_TOKEN"]
for idx in indexes:
    if (idx.get("authenticate") or "").lower() != "always":
        continue
    name = (idx.get("name") or "").strip()
    if not name:
        continue
    key = "".join(ch if ch.isalnum() else "_" for ch in name).upper()
    print(f"UV_INDEX_{key}_USERNAME={user}")
    print(f"UV_INDEX_{key}_PASSWORD={token}")
PY
)"

docker_env=(
	-e "UV_PUBLISH_USERNAME=${user}"
	-e "UV_PUBLISH_PASSWORD=${token}"
	-e "PUBLISH_URL=${registry}"
)
if [[ -n "$index_env" ]]; then
	while IFS= read -r line; do
		[[ -z "$line" ]] && continue
		docker_env+=(-e "$line")
	done <<<"$index_env"
fi

ensure_harbor_login
echo "==> uv build+publish ${pkg_path} (${image})"
docker run --rm --network=host \
	"${docker_env[@]}" \
	-v "${ROOT}/${pkg_path}:/app" \
	-w /app \
	"$image" \
	sh -ec 'command -v uv >/dev/null || pip install --no-cache-dir uv
		rm -rf dist
		uv build
		uv publish --publish-url "$PUBLISH_URL" dist/*'
echo "OK pypi-publish ${registry}"
