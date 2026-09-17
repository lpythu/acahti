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
if [[ -z "${ARGOS_DASH:-}" ]]; then
	echo "error: ARGOS_DASH is required (org secret argos_dash = dash.env from /cli)" >&2
	exit 1
fi
dashf="$(mktemp)"
trap 'rm -f "$dashf"' EXIT
printf '%s\n' "$ARGOS_DASH" >"$dashf"
chmod 600 "$dashf"
if ! grep -q '^ARGOS_DASH_URL=.' "$dashf" || ! grep -q '^ARGOS_TOKEN=.' "$dashf"; then
	echo "error: ARGOS_DASH must be dash.env (ARGOS_DASH_URL and ARGOS_TOKEN)" >&2
	exit 1
fi
while IFS= read -r sel; do
	[[ -z "$sel" ]] && continue
	# shellcheck disable=SC2086
	argos run ${sel} --env "$envn" --dash "$dashf"
done < <(each_item "$(input SELECTORS)")
echo "OK argos"
