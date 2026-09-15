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
add_bk_secret npm_token "${NPM_TOKEN:-}"
add_bk_secret codeup_netrc "${CODEUP_NETRC:-}"

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
