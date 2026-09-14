#!/usr/bin/env bash
# helm the chart, then argos (argospy on PATH; packs from argos-pack).
set -euo pipefail
here="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib.sh
source "${here}/lib.sh"
# shellcheck source=kube.sh
source "${here}/kube.sh"
require_env
: "${NS:?set NS}"
: "${RELEASE:?set RELEASE}"
CHART="${CHART:-chart}"
: "${ROOT:?set ROOT}"
chart_path="${ROOT}/${CHART}"
if [[ ! -f "${chart_path}/Chart.yaml" ]]; then
  echo "error: chart not found at ${chart_path}" >&2
  exit 1
fi

if [[ "${KIND:-}" == "helm" ]]; then
  echo "==> helm ${RELEASE} ENV=${ENV} (no image)"
  helm_upgrade "$chart_path" "$RELEASE" "$NS"
  echo "OK helm ${RELEASE} ENV=${ENV}"
else
  require_repo
  commit_id
  : "${IMAGE:?set IMAGE in .acahti/repo.env}"
  repo_img="$(deploy_repo)"
  tag="$(deploy_tag)"
  echo "==> helm ${RELEASE} ENV=${ENV} ${repo_img}:${tag}"
  helm_upgrade "$chart_path" "$RELEASE" "$NS" "$repo_img" "$tag"
  echo "OK helm ${RELEASE} ENV=${ENV} tag=${tag}"
fi

: "${ARGOS_SELECTORS:?set ARGOS_SELECTORS (semicolon-separated argos invocations)}"
command -v argos >/dev/null || {
  echo "error: argos CLI required on the agent (pip install argospy && pip install -e argos-pack)" >&2
  exit 1
}

oldifs="$IFS"
IFS=';'
for sel in ${ARGOS_SELECTORS}; do
  sel="${sel#"${sel%%[![:space:]]*}"}"
  sel="${sel%"${sel##*[![:space:]]}"}"
  [[ -z "$sel" ]] && continue
  IFS="$oldifs"
  # shellcheck disable=SC2086
  argos run ${sel} --env "$ENV"
  IFS=';'
done
IFS="$oldifs"
