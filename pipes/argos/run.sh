#!/usr/bin/env bash
set -euo pipefail
# shellcheck source=../_lib.sh
source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/_lib.sh"

require_input ENV
require_input SELECTORS
command -v argos >/dev/null || {
	echo "error: argos CLI required on the Runner" >&2
	exit 1
}
if ! argos run --help 2>/dev/null | grep -q -- '--dash'; then
	echo "error: argos CLI on the Runner is too old (need --dash); upgrade argospy" >&2
	exit 1
fi
if [[ -z "${ARGOS_DASH_URL:-}" || -z "${ARGOS_TOKEN:-}" ]]; then
	echo "error: ARGOS_DASH_URL and ARGOS_TOKEN secrets are required" >&2
	exit 1
fi
envn="$(input ENV)"
dashf="$(mktemp)"
trap 'rm -f "$dashf"' EXIT
printf 'ARGOS_DASH_URL=%s\nARGOS_TOKEN=%s\n' "$ARGOS_DASH_URL" "$ARGOS_TOKEN" >"$dashf"
chmod 600 "$dashf"
echo "==> argos $(argos --version 2>/dev/null || command -v argos)"
while IFS= read -r sel; do
	[[ -z "$sel" ]] && continue
	# shellcheck disable=SC2086
	argos run ${sel} --env "$envn" --dash "$dashf"
done < <(each_item "$(input SELECTORS)")
echo "OK argos"
