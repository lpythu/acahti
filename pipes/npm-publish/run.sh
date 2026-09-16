#!/usr/bin/env bash
set -euo pipefail
# shellcheck source=../_lib.sh
source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/_lib.sh"

require_input PATH
require_input REGISTRY
pkg_path="$(input PATH)"
if [[ ! -f "${ROOT}/${pkg_path}/package.json" ]]; then
	echo "error: ${pkg_path}/package.json missing" >&2
	exit 1
fi

token="${ACAHTI_TOKEN:-}"
user="${ACAHTI_USER:-}"
if [[ -z "$token" || -z "$user" ]]; then
	echo "error: job Acahti identity is required" >&2
	exit 1
fi

registry="$(input REGISTRY)"
origin="$(input ORIGIN)"
if [[ -z "$origin" ]]; then
	origin="${registry%/api/packages/*}"
fi
org="$(input ORG)"
if [[ -z "$org" ]]; then
	org="${registry##*/api/packages/}"
	org="${org%%/*}"
fi
user="$(input USER)"
user="${user:-$ACAHTI_USER}"
host="${origin#https://}"
host="${host#http://}"
host="${host%%/*}"
auth_path="/api/packages/${org}/npm/"

image="$(input IMAGE)"
image="${image:-node:22-alpine}"
pkg_name="$(python3 -c "import json,sys; print(json.load(open(sys.argv[1]))['name'])" "${ROOT}/${pkg_path}/package.json")"

dest="$(mktemp -d)"
npmrc="$(mktemp)"
trap 'rm -rf "$dest"; rm -f "$npmrc"' EXIT
chmod 600 "$npmrc"
{
	printf '//%s%s:username=%s\n' "$host" "$auth_path" "$user"
	printf '//%s%s:_password=%s\n' "$host" "$auth_path" "$(printf '%s' "$token" | base64 | tr -d '\n')"
	printf 'always-auth=true\n'
} >"$npmrc"

ensure_harbor_login
echo "==> npm install+build ${pkg_name} (${image})"
docker run --rm --network=host \
	-e PKG_NAME="$pkg_name" \
	-e PKG_PATH="$pkg_path" \
	-v "${ROOT}:/app" \
	-v "${npmrc}:/root/.npmrc:ro" \
	-v "${dest}:/out" \
	-w /app \
	"$image" \
	sh -ec 'corepack enable
		pnpm install --frozen-lockfile --ignore-scripts
		pnpm --filter "$PKG_NAME" build
		cd "$PKG_PATH"
		npm pack --pack-destination /out --ignore-scripts'

shopt -s nullglob
tgz=( "$dest"/*.tgz )
if ((${#tgz[@]} == 0)); then
	echo "error: ${pkg_path} produced no tarball" >&2
	exit 1
fi
echo "==> npm PUT ${registry}"
ACAHTI_ADMIN_TOKEN="${token}" ACAHTI_ORG="${org}" ACAHTI_ADMIN_USER="${user}" \
	ORIGIN="${origin}" HOST_HEADER="${host}" X_FORWARDED_PROTO=https \
	python3 "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/npm-put.py" "${tgz[@]}"
echo "OK npm-publish ${registry}"
