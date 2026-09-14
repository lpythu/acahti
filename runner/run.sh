#!/usr/bin/env bash
# Load .acahti/repo.env and run ci|cd|pkg.
set -euo pipefail
here="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

ROOT="${ROOT:-${CI_WORKSPACE:-}}"
if [[ -z "${ROOT:-}" || ! -d "$ROOT" ]]; then
  ROOT="$(pwd)"
fi
export ROOT

envf="${ROOT}/.acahti/repo.env"
if [[ ! -f "$envf" ]]; then
  echo "error: missing ${envf}" >&2
  exit 1
fi
# shellcheck disable=SC1090
source "$envf"
export KIND NS RELEASE IMAGE CHART ARGOS_SELECTORS PKG_PATH PKG_NAME
: "${KIND:?set KIND in .acahti/repo.env}"

export ACAHTI_RUNNER="${ACAHTI_RUNNER:-$here}"

cmd="${1:?usage: run.sh ci|cd|pkg}"
case "$cmd" in
ci) bash "${here}/ci.sh" ;;
cd) bash "${here}/cd.sh" ;;
pkg) bash "${here}/pkg.sh" ;;
*)
  echo "usage: run.sh ci|cd|pkg" >&2
  exit 1
  ;;
esac
