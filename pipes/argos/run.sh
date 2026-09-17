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
if [[ -z "${ARGOS_DASH:-}" ]]; then
	echo "error: ARGOS_DASH is required (org secret argos_dash = ARGOS_TOKEN from /cli)" >&2
	exit 1
fi
if [[ "$ARGOS_DASH" == *$'\n'* || "$ARGOS_DASH" == *'='* ]]; then
	echo "error: argos_dash must be the ARGOS_TOKEN value only, not a dash.env file" >&2
	exit 1
fi
envn="$(input ENV)"
dash_url="$(input DASH_URL)"
export ARGOS_DASH_URL="${dash_url:-https://argos.saidc.ai}"
export ARGOS_TOKEN="$ARGOS_DASH"
echo "==> argos $(argos --version 2>/dev/null || command -v argos) dash=${ARGOS_DASH_URL}"
while IFS= read -r sel; do
	[[ -z "$sel" ]] && continue
	# shellcheck disable=SC2086
	argos run ${sel} --env "$envn" --dash
done < <(each_item "$(input SELECTORS)")
echo "OK argos"
