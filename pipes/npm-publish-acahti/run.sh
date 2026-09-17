#!/usr/bin/env bash
set -euo pipefail
# shellcheck source=../_lib.sh
source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/_lib.sh"

require_input PATH
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

origin="$(acahti_packages_origin)"
org="$(acahti_pkg_org)"
public_host="$(acahti_packages_public_host)"
if [[ -z "$org" || -z "$public_host" ]]; then
	echo "error: ACAHTI_ORG and ACAHTI_ROOT_URL are required" >&2
	exit 1
fi
lan_host="$(acahti_pkg_host "$origin")"
auth_path="/api/packages/${org}/npm/"

image="$(input IMAGE)"
image="${image:-node:22-alpine}"
pkg_name="$(python3 -c "import json,sys; print(json.load(open(sys.argv[1]))['name'])" "${ROOT}/${pkg_path}/package.json")"

dest="$(mktemp -d)"
npmrc="$(mktemp)"
overlay="$(mktemp)"
pkg_overlay="$(mktemp)"
trap 'rm -rf "$dest"; rm -f "$npmrc" "$overlay" "$pkg_overlay"' EXIT
chmod 600 "$npmrc"
{
	# Frozen lockfiles keep tarball URLs on the public host; PUT uses the LAN
	# gateway. Auth both so pnpm never fetches Acahti packages anonymously.
	# Do not rewrite lockfile tarball URLs: pnpm supply-chain policy requires
	# they match the registry's published metadata.
	for host in "$lan_host" "$public_host"; do
		[[ -n "$host" ]] || continue
		printf '//%s%s:username=%s\n' "$host" "$auth_path" "$user"
		printf '//%s%s:_password=%s\n' "$host" "$auth_path" "$(printf '%s' "$token" | base64 | tr -d '\n')"
	done
	printf 'always-auth=true\n'
} >"$npmrc"

rewrite_npmrc() {
	local src="$1" out="$2"
	if [[ -f "$src" ]]; then
		sed -E "s#https?://[^/]+/api/packages/#${origin}/api/packages/#g" "$src" >"$out"
	else
		printf '@%s:registry=%s/api/packages/%s/npm/\n' "$org" "$origin" "$org" >"$out"
	fi
}
rewrite_npmrc "${ROOT}/.npmrc" "$overlay"
rewrite_npmrc "${ROOT}/${pkg_path}/.npmrc" "$pkg_overlay"

docker_npmrc_mounts=(
	-v "${npmrc}:/root/.npmrc:ro"
	-v "${overlay}:/app/.npmrc:ro"
)
if [[ "$pkg_path" != "." && -f "${ROOT}/${pkg_path}/.npmrc" ]]; then
	docker_npmrc_mounts+=(-v "${pkg_overlay}:/app/${pkg_path}/.npmrc:ro")
fi

echo "==> npm install+build ${pkg_name} (${image}) origin ${origin}"
docker run --rm --network=host \
	-e PKG_NAME="$pkg_name" \
	-e PKG_PATH="$pkg_path" \
	-v "${ROOT}:/app" \
	"${docker_npmrc_mounts[@]}" \
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
echo "==> npm PUT ${origin}/api/packages/${org}/npm"
ACAHTI_ADMIN_TOKEN="${token}" ACAHTI_ORG="${org}" ACAHTI_ADMIN_USER="${user}" \
	ORIGIN="${origin}" HOST_HEADER="${public_host}" X_FORWARDED_PROTO=https \
	python3 "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/npm-put.py" "${tgz[@]}"
echo "OK npm-publish-acahti ${pkg_name}"
