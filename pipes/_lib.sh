# Shared helpers for official pipes. Source only.
set -euo pipefail

ROOT="${CI_WORKSPACE:-$(pwd)}"

input() {
	local key="$1"
	local var="INPUT_${key}"
	printf '%s' "${!var:-}"
}

require_input() {
	local key="$1"
	local var="INPUT_${key}"
	if [[ -z "${!var:-}" ]]; then
		local name
		name="$(printf '%s' "$key" | tr '[:upper:]' '[:lower:]')"
		echo "error: with.${name} is required" >&2
		exit 1
	fi
}

each_line() {
	local blob="$1"
	local line
	while IFS= read -r line || [[ -n "$line" ]]; do
		line="${line#"${line%%[![:space:]]*}"}"
		line="${line%"${line##*[![:space:]]}"}"
		[[ -z "$line" || "$line" == \#* ]] && continue
		printf '%s\n' "$line"
	done <<<"$blob"
}

# Split on newlines or semicolons.
each_item() {
	local blob="$1"
	blob="${blob//;/$'\n'}"
	each_line "$blob"
}

# Public packages URL host + org. Expand injects ACAHTI_ROOT_URL / ACAHTI_ORG.
# Install auth (docker-build npmrc/netrc) still names this host; PUT uses
# acahti_packages_origin (LAN gateway), with Host = acahti_packages_public_host.
acahti_pkg_origin() {
	local origin="${ACAHTI_ROOT_URL:-${ROOT_URL:-${CI_FORGE_URL:-}}}"
	origin="${origin%/}"
	printf '%s' "$origin"
}

acahti_pkg_org() {
	printf '%s' "${ACAHTI_ORG:-${CI_REPO_OWNER:-}}"
}

acahti_pkg_host() {
	local origin="$1"
	local host="${origin#https://}"
	host="${host#http://}"
	host="${host%%/*}"
	printf '%s' "$host"
}

# Runner → gateway :8080. Never ACAHTI_ROOT_URL (that is the public tunnel).
acahti_packages_origin() {
	if [[ -n "${ACAHTI_GATEWAY:-}" ]]; then
		printf '%s' "${ACAHTI_GATEWAY%/}"
		return
	fi
	if [[ -r /etc/woodpecker/acahti-gateway ]]; then
		local file_origin
		file_origin="$(tr -d '[:space:]' </etc/woodpecker/acahti-gateway)"
		if [[ -n "$file_origin" ]]; then
			printf '%s' "${file_origin%/}"
			return
		fi
	fi
	local server="${WOODPECKER_SERVER:-}"
	local host="${server%%:*}"
	if [[ -z "$host" || "$host" == "127.0.0.1" || "$host" == "localhost" ]]; then
		printf '%s' "http://127.0.0.1:8080"
		return
	fi
	printf '%s' "http://${host}:8080"
}

acahti_packages_public_host() {
	local host
	host="$(acahti_pkg_host "$(acahti_pkg_origin)")"
	if [[ -n "$host" ]]; then
		printf '%s' "$host"
		return
	fi
	acahti_pkg_host "$(acahti_packages_origin)"
}

# User-level npmrc: HTTP Basic (username + base64 _password). Not Bearer.
format_acahti_npmrc() {
	local user="${ACAHTI_USER:-}" token="${ACAHTI_TOKEN:-}"
	[[ -n "$user" && -n "$token" ]] || return 0
	local origin org host
	origin="$(acahti_pkg_origin)"
	org="$(acahti_pkg_org)"
	host="$(acahti_pkg_host "$origin")"
	if [[ -z "$host" || -z "$org" ]]; then
		echo "error: ACAHTI_ROOT_URL and ACAHTI_ORG required to write npm package auth" >&2
		return 1
	fi
	local path="/api/packages/${org}/npm/"
	printf '//%s%s:username=%s\n' "$host" "$path" "$user"
	printf '//%s%s:_password=%s\n' "$host" "$path" "$(printf '%s' "$token" | base64 | tr -d '\n')"
	printf 'always-auth=true\n'
}

format_acahti_netrc() {
	local user="${ACAHTI_USER:-}" token="${ACAHTI_TOKEN:-}"
	[[ -n "$user" && -n "$token" ]] || return 0
	local origin host
	origin="$(acahti_pkg_origin)"
	host="$(acahti_pkg_host "$origin")"
	if [[ -z "$host" ]]; then
		echo "error: ACAHTI_ROOT_URL required to write pypi package auth" >&2
		return 1
	fi
	printf 'machine %s\nlogin %s\npassword %s\n' "$host" "$user" "$token"
}

truthy() {
	case "${1:-}" in
	true | TRUE | yes | YES | 1) return 0 ;;
	*) return 1 ;;
	esac
}

# Hostname from a docker-login registry value (strip scheme and path).
registry_host() {
	local h="${1:-}"
	h="${h#http://}"
	h="${h#https://}"
	h="${h%%/*}"
	printf '%s' "$h"
}

# HTTP/insecure is declared in product YAML (`docker-login` with.http). Merge
# into the shared builder config and recreate when the host is new.
ensure_registry_http() {
	local host="$1"
	host="$(registry_host "$host")"
	[[ -n "$host" ]] || return 0
	persist_buildx_config
	local cfg="${ACAHTI_BUILDKITD_CONFIG}"
	local dir
	dir="$(dirname "$cfg")"
	mkdir -p "$dir"
	local lock="${cfg}.lock"
	(
		flock 9
		if [[ -f "$cfg" ]] && grep -Fq "[registry.\"${host}\"]" "$cfg"; then
			exit 0
		fi
		printf '[registry."%s"]\n  http = true\n  insecure = true\n\n' "$host" >>"$cfg"
		docker buildx rm -f "$ACAHTI_BUILDER" >/dev/null 2>&1 || true
		create_acahti_builder "$ACAHTI_BUILDER"
	) 9>"$lock"
}

# Woodpecker only interpolates ${CI_COMMIT_TAG}, not bash ${CI_COMMIT_TAG#v}.
# Git tags are vX.Y.Z; OCI / helm tags are X.Y.Z.
expand_ci() {
	local s="$1"
	local tag="${CI_COMMIT_TAG:-}"
	tag="${tag#v}"
	s="${s//'${CI_COMMIT_TAG#v}'/$tag}"
	printf '%s' "$s"
}

# Isolated HOME per workflow would hide named buildx builders. Keep instances
# on the runner user's real ~/.docker/buildx (auth stays in isolated HOME).
persist_buildx_config() {
	if [[ -n "${BUILDX_CONFIG:-}" ]]; then
		mkdir -p "$BUILDX_CONFIG"
		return 0
	fi
	local home
	home="$(getent passwd "$(id -un)" | cut -d: -f6)"
	home="${home:-${HOME:-/}}"
	export BUILDX_CONFIG="${home}/.docker/buildx"
	mkdir -p "$BUILDX_CONFIG"
}

ACAHTI_BUILDER="${ACAHTI_BUILDER:-acahti}"
ACAHTI_BUILDKITD_CONFIG="${ACAHTI_BUILDKITD_CONFIG:-/etc/woodpecker/buildkitd.toml}"

# docker-container inspect prints `network: "host"` or `network=host`, not `Network: host`.
builder_uses_host_network() {
	local info="$1"
	local driver
	driver="$(printf '%s\n' "$info" | awk -F': *' '/^Driver:/{print $2; exit}')"
	if [[ "$driver" == "docker" ]]; then
		return 0
	fi
	printf '%s\n' "$info" | grep -qiE 'network[[:space:]]*[=:][[:space:]]*"?host'
}

create_acahti_builder() {
	local name="${1:-$ACAHTI_BUILDER}"
	local -a args=(--name "$name" --driver docker-container --driver-opt network=host)
	local err
	if [[ -s "$ACAHTI_BUILDKITD_CONFIG" ]]; then
		args+=(--config "$ACAHTI_BUILDKITD_CONFIG")
	fi
	echo "==> buildx create ${name} docker-container network=host"
	if err="$(docker buildx create "${args[@]}" 2>&1 >/dev/null)"; then
		return 0
	fi
	# Parallel jobs share one builder; create is not idempotent.
	if docker buildx inspect "$name" >/dev/null 2>&1; then
		echo "==> buildx ${name} already exists"
		return 0
	fi
	printf '%s\n' "$err" >&2
	return 1
}

ensure_acahti_builder() {
	persist_buildx_config
	local name="${1:-$ACAHTI_BUILDER}"
	local lock="${ACAHTI_BUILDKITD_CONFIG}.lock"
	(
		flock 9
		local info
		if info="$(docker buildx inspect "$name" 2>/dev/null)"; then
			if builder_uses_host_network "$info"; then
				exit 0
			fi
			echo "==> rebuild ${name} docker-container network=host"
			docker buildx rm -f "$name" >/dev/null 2>&1 || true
		fi
		create_acahti_builder "$name"
	) 9>"$lock"
}
