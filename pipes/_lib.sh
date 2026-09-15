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

# Woodpecker only interpolates ${CI_COMMIT_TAG}, not bash ${CI_COMMIT_TAG#v}.
# Git tags are vX.Y.Z; OCI / helm tags are X.Y.Z.
expand_ci() {
	local s="$1"
	local tag="${CI_COMMIT_TAG:-}"
	tag="${tag#v}"
	s="${s//'${CI_COMMIT_TAG#v}'/$tag}"
	printf '%s' "$s"
}
