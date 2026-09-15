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
envn="$(input ENV)"
dash=()
if [[ -n "${ARGOS_DASH:-}" ]]; then
	dashf="$(mktemp)"
	trap 'rm -f "$dashf"' EXIT
	printf '%s\n' "$ARGOS_DASH" >"$dashf"
	chmod 600 "$dashf"
	dash=(--dash "$dashf")
fi
while IFS= read -r sel; do
	[[ -z "$sel" ]] && continue
	# shellcheck disable=SC2086
	argos run ${sel} --env "$envn" "${dash[@]}"
done < <(each_item "$(input SELECTORS)")
echo "OK argos"
