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

# FROM bases live on office Harbor. HK CD logs into ACR; do not overwrite
# an existing Harbor robot login from docker-login.
ensure_harbor_login() {
	local dest="${HOME}/.docker/config.json"
	if [[ -f "$dest" ]] && grep -q '"harbor.saidc"' "$dest"; then
		return
	fi
	if [[ -n "${HARBOR_PASSWORD:-}" ]]; then
		printf '%s\n' "$HARBOR_PASSWORD" | docker login harbor.saidc -u 'robot$saidc' --password-stdin >/dev/null
		return
	fi
	local pw
	pw="$(getent passwd "$(id -un)" | cut -d: -f6)/.harbor/robot-saidc.secret"
	[[ -f "$pw" ]] || return 0
	docker login harbor.saidc -u 'robot$saidc' --password-stdin <"$pw" >/dev/null
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

ensure_acahti_builder() {
	persist_buildx_config
	local name="${1:-$ACAHTI_BUILDER}"
	local info driver
	if info="$(docker buildx inspect "$name" 2>/dev/null)"; then
		driver="$(printf '%s\n' "$info" | awk -F': *' '/^Driver:/{print $2; exit}')"
		if [[ "$driver" == "docker" ]]; then
			return 0
		fi
		if printf '%s\n' "$info" | grep -qiE 'Network:[[:space:]]*host'; then
			return 0
		fi
		echo "error: builder ${name} must use host network" >&2
		return 1
	fi
	echo "==> buildx create ${name} docker-container network=host"
	docker buildx create --name "$name" --driver docker-container --driver-opt network=host >/dev/null
}
