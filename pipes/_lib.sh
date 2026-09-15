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

# FROM bases live on office Harbor. HK CD logs into ACR in a Woodpecker
# temp HOME, so reuse the Runner user's Harbor auth when the step secret
# was not injected.
ensure_harbor_login() {
	if [[ -n "${HARBOR_PASSWORD:-}" ]]; then
		printf '%s\n' "$HARBOR_PASSWORD" | docker login harbor.saidc -u 'robot$saidc' --password-stdin >/dev/null
		return
	fi
	local user real_home src dest
	user="$(id -un)"
	real_home="$(getent passwd "$user" | cut -d: -f6)"
	src="${real_home}/.docker/config.json"
	dest="${HOME}/.docker/config.json"
	[[ -f "$src" && "$src" != "$dest" ]] || return 0
	mkdir -p "${HOME}/.docker"
	python3 - "$src" "$dest" <<'PY'
import json, os, sys

src, dest = sys.argv[1], sys.argv[2]
with open(src, encoding="utf-8") as f:
    harbor = (json.load(f).get("auths") or {}).get("harbor.saidc")
if not harbor:
    raise SystemExit(0)
data = {"auths": {}}
if os.path.exists(dest):
    with open(dest, encoding="utf-8") as f:
        data = json.load(f)
data.setdefault("auths", {})["harbor.saidc"] = harbor
os.makedirs(os.path.dirname(dest) or ".", exist_ok=True)
with open(dest, "w", encoding="utf-8") as f:
    json.dump(data, f)
PY
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
