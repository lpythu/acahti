#!/usr/bin/env bash
set -euo pipefail
# shellcheck source=../_lib.sh
source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/_lib.sh"

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

origin="$(acahti_packages_origin)"
org="$(acahti_pkg_org)"
public_host="$(acahti_packages_public_host)"
if [[ -z "$org" || -z "$public_host" ]]; then
	echo "error: ACAHTI_ORG and ACAHTI_ROOT_URL are required" >&2
	exit 1
fi

image="$(input IMAGE)"
image="${image:-python:3.12-slim-bookworm}"
dest="$(mktemp -d)"
trap 'rm -rf "$dest"' EXIT

echo "==> uv build ${pkg_path} (${image})"
docker run --rm --network=host \
	-v "${ROOT}/${pkg_path}:/app" \
	-v "${dest}:/out" \
	-w /app \
	"$image" \
	sh -ec 'command -v uv >/dev/null || pip install --no-cache-dir uv
		rm -rf dist
		uv build
		cp dist/* /out/'

shopt -s nullglob
arts=( "$dest"/* )
if ((${#arts[@]} == 0)); then
	echo "error: ${pkg_path} produced no dist" >&2
	exit 1
fi
echo "==> pypi PUT ${origin}/api/packages/${org}/pypi"
ACAHTI_ADMIN_TOKEN="${token}" ACAHTI_ORG="${org}" ACAHTI_ADMIN_USER="${user}" \
	ORIGIN="${origin}" HOST_HEADER="${public_host}" X_FORWARDED_PROTO=https \
	python3 "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/pypi-put.py" "${arts[@]}"
echo "OK pypi-publish-acahti ${pkg_path}"
