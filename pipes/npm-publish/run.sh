#!/usr/bin/env bash
set -euo pipefail
# shellcheck source=../_lib.sh
source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/_lib.sh"

require_input PATH
require_input REGISTRY
pkg_path="$(input PATH)"
if [[ ! -d "${ROOT}/${pkg_path}/dist" ]] || [[ -z "$(ls -A "${ROOT}/${pkg_path}/dist" 2>/dev/null)" ]]; then
	echo "error: ${pkg_path}/dist empty; build the package first" >&2
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

image="$(input IMAGE)"
image="${image:-node:22-alpine}"
dest="$(mktemp -d)"
trap 'rm -rf "$dest"' EXIT
echo "==> npm pack ${pkg_path} (${image})"
docker run --rm --network=host \
	-v "${ROOT}/${pkg_path}:/pkg:ro" \
	-v "${dest}:/out" \
	-w /pkg \
	"$image" \
	npm pack --pack-destination /out --ignore-scripts
echo "==> npm PUT ${registry}"
ACAHTI_ADMIN_TOKEN="${token}" ACAHTI_ORG="${org}" ACAHTI_ADMIN_USER="${user}" \
	ORIGIN="${origin}" HOST_HEADER="${host}" X_FORWARDED_PROTO=https \
	python3 "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/npm-put.py" "$dest"/*.tgz
echo "OK npm-publish ${registry}"
