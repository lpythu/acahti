#!/usr/bin/env bash
set -euo pipefail
# shellcheck source=../_lib.sh
source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/_lib.sh"

require_input IMAGES
cd "$ROOT"

network="$(input NETWORK)"
network="${network:-host}"
builder="$(input BUILDER)"
builder="${builder:-default}"
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
add_bk_secret acahti_user "${ACAHTI_USER:-}"
add_bk_secret acahti "${ACAHTI_TOKEN:-}"
add_bk_secret codeup_netrc "${CODEUP_NETRC:-}"

ensure_host_builder() {
	local info driver
	info="$(docker buildx inspect "$builder" 2>/dev/null)" || {
		echo "error: buildx builder ${builder} not found" >&2
		exit 1
	}
	driver="$(printf '%s\n' "$info" | awk -F': *' '/^Driver:/{print $2; exit}')"
	if [[ "$driver" == "docker" ]]; then
		return 0
	fi
	if printf '%s\n' "$info" | grep -qiE 'Network:[[:space:]]*host'; then
		return 0
	fi
	echo "error: builder ${builder} must use host network (docker driver, or --driver-opt network=host)" >&2
	exit 1
}

ensure_host_builder
ensure_harbor_login

build_one() {
	local primary="$1"
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
		--load
	)
	[[ -n "$file" ]] && args+=(-f "$file")
	args+=("${secret_args[@]}" "${bargs[@]}" "$context")
	echo "==> build ${primary}"
	docker buildx build "${args[@]}"
	if [[ "$push" == "true" ]]; then
		echo "==> push ${primary}"
		docker push "$primary"
	fi
	local extra
	IFS=',' read -r -a extras <<<"$also"
	for extra in "${extras[@]}"; do
		extra="${extra#"${extra%%[![:space:]]*}"}"
		extra="${extra%"${extra##*[![:space:]]}"}"
		[[ -z "$extra" ]] && continue
		echo "==> tag ${primary} → ${extra}"
		docker tag "$primary" "$extra"
		if [[ "$push" == "true" ]]; then
			echo "==> push ${extra}"
			docker push "$extra"
		fi
	done
	echo "OK docker-build ${primary}"
}

while IFS= read -r line; do
	# shellcheck disable=SC2086
	set -- $line
	build_one "$@"
done < <(each_line "$(expand_ci "$(input IMAGES)")")
