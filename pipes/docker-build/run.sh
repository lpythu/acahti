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
add_bk_secret codeup_netrc "${CODEUP_NETRC:-}"

# Job identity → the same files engineers use locally. Dockerfiles COPY them;
# they must not format credentials themselves.
write_job_identity_files() {
	local user="${ACAHTI_USER:-}" token="${ACAHTI_TOKEN:-}"
	[[ -z "$user" || -z "$token" ]] && return 0

	local host="acahti.saidc.ai" org="saidc" line rest
	org="${CI_REPO_OWNER:-${ACAHTI_ORG:-saidc}}"
	if [[ -f .npmrc ]]; then
		line="$(grep -E '^@saidc:registry=' .npmrc | tail -n1 || true)"
		if [[ -n "$line" ]]; then
			rest="${line#*://}"
			host="${rest%%/*}"
			if [[ "$rest" == *"/api/packages/"* ]]; then
				org="${rest#*/api/packages/}"
				org="${org%%/*}"
			fi
		fi
	fi

	if [[ -f package.json || -f pnpm-lock.yaml || -f package-lock.json || -f yarn.lock ]]; then
		local pass npmrc=".npmrc"
		pass="$(printf %s "$token" | base64 | tr -d '\n')"
		if [[ -f "$npmrc" ]]; then
			grep -Ev '//.*:username=|//.*:_password=|^always-auth=' "$npmrc" >"${npmrc}.tmp" || true
			mv "${npmrc}.tmp" "$npmrc"
		fi
		if ! grep -qE '^@saidc:registry=' "$npmrc" 2>/dev/null; then
			printf '@saidc:registry=https://%s/api/packages/%s/npm/\n' "$host" "$org" >>"$npmrc"
		fi
		printf '//%s/api/packages/%s/npm/:username=%s\n' "$host" "$org" "$user" >>"$npmrc"
		printf '//%s/api/packages/%s/npm/:_password=%s\nalways-auth=true\n' "$host" "$org" "$pass" >>"$npmrc"
		echo "==> wrote .npmrc auth for ${user} @ ${host}"
	fi

	if [[ -f pyproject.toml || -f uv.lock ]]; then
		printf 'machine %s\nlogin %s\npassword %s\n' "$host" "$user" "$token" >.netrc
		chmod 600 .netrc
		echo "==> wrote .netrc for ${user} @ ${host}"
	fi
}

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

write_job_identity_files
ensure_host_builder
ensure_harbor_login

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
		--load
	)
	[[ -n "$file" ]] && args+=(-f "$file")
	if ((${#secret_args[@]})); then
		args+=("${secret_args[@]}")
	fi
	if ((${#bargs[@]})); then
		args+=("${bargs[@]}")
	fi
	args+=("$context")
	echo "==> build ${primary}"
	docker buildx build "${args[@]}"
	if [[ "$push" == "true" ]]; then
		echo "==> push ${primary}"
		docker push "$primary"
	fi
	local extra
	local -a extras=()
	IFS=',' read -r -a extras <<<"$also"
	for extra in "${extras[@]+"${extras[@]}"}"; do
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
	[[ $# -eq 0 || -z "${1:-}" ]] && continue
	build_one "$@"
done < <(each_line "$(expand_ci "$(input IMAGES)")")
