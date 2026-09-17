#!/usr/bin/env bash
set -euo pipefail
# shellcheck source=../_lib.sh
source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/_lib.sh"

require_input IMAGES
cd "$ROOT"

network="$(input NETWORK)"
network="${network:-host}"
builder="$(input BUILDER)"
builder="${builder:-$ACAHTI_BUILDER}"
push="$(input PUSH)"
push="${push:-true}"

secret_args=()
secret_files=()
cleanup_secrets() {
	if ((${#secret_files[@]})); then
		rm -f "${secret_files[@]}"
	fi
}
trap cleanup_secrets EXIT

add_bk_secret() {
	local id="$1" value="$2"
	[[ -z "$value" ]] && return 0
	local src
	src="$(mktemp)"
	secret_files+=("$src")
	printf '%s' "$value" >"$src"
	chmod 600 "$src"
	secret_args+=(--secret "id=${id},src=${src}")
}
add_bk_secret codeup_netrc "${CODEUP_NETRC:-}"
# Job identity as user-level npmrc / netrc (HTTP Basic). YAML does not name tokens.
# Dockerfiles mount id=npmrc → /root/.npmrc and id=netrc → /root/.netrc.
npmrc_body="$(format_acahti_npmrc)"
netrc_body="$(format_acahti_netrc)"
add_bk_secret npmrc "$npmrc_body"
add_bk_secret netrc "$netrc_body"
if [[ -n "$npmrc_body" || -n "$netrc_body" ]]; then
	echo "==> docker-build secrets npmrc netrc"
fi

ensure_host_builder() {
	if [[ "$builder" == "$ACAHTI_BUILDER" ]]; then
		ensure_acahti_builder "$builder" || exit 1
		return 0
	fi
	local info
	persist_buildx_config
	info="$(docker buildx inspect "$builder" 2>/dev/null)" || {
		echo "error: buildx builder ${builder} not found" >&2
		exit 1
	}
	if builder_uses_host_network "$info"; then
		return 0
	fi
	echo "error: builder ${builder} must use host network (docker driver, or --driver-opt network=host)" >&2
	exit 1
}

ensure_host_builder

build_one() {
	local primary="${1:-}"
	if [[ -z "$primary" ]]; then
		return 0
	fi
	shift
	local context="." file="" also=""
	local -a bargs=()
	local spec key val
	for spec in "$@"; do
		key="${spec%%=*}"
		val="${spec#*=}"
		case "$key" in
		context) context="$val" ;;
		file | dockerfile) file="$val" ;;
		also) also="$val" ;;
		*)
			if [[ "$spec" == *TOKEN* || "$spec" == *PASS* || "$spec" == *SECRET* ]]; then
				echo "error: build-arg must not contain secrets" >&2
				exit 1
			fi
			bargs+=(--build-arg "$spec")
			;;
		esac
	done
	local -a args=(
		--builder "$builder"
		--pull
		--provenance=false
		--network="$network"
		-t "$primary"
	)
	local extra
	local -a extras=()
	IFS=',' read -r -a extras <<<"$also"
	for extra in "${extras[@]+"${extras[@]}"}"; do
		extra="${extra#"${extra%%[![:space:]]*}"}"
		extra="${extra%"${extra##*[![:space:]]}"}"
		[[ -z "$extra" ]] && continue
		args+=(-t "$extra")
	done
	[[ -n "$file" ]] && args+=(-f "$file")
	if ((${#secret_args[@]})); then
		args+=("${secret_args[@]}")
	fi
	if ((${#bargs[@]})); then
		args+=("${bargs[@]}")
	fi
	if [[ "$push" == "true" ]]; then
		args+=(--push)
	else
		args+=(--output=type=cacheonly)
	fi
	args+=("$context")
	if [[ "$push" == "true" ]]; then
		echo "==> build+push ${primary}"
	else
		echo "==> build ${primary} (cache-only)"
	fi
	docker buildx build "${args[@]}"
	echo "OK docker-build ${primary}"
}

while IFS= read -r line; do
	# shellcheck disable=SC2086
	set -- $line
	[[ $# -eq 0 || -z "${1:-}" ]] && continue
	build_one "$@"
done < <(each_line "$(expand_ci "$(input IMAGES)")")
