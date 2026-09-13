# Shared helpers for acahti/cicd_acahti. Source only.
set -euo pipefail

REGISTRY="${REGISTRY:-harbor.saidc}"
ACR_REGISTRY="${ACR_REGISTRY:-saidc-registry.cn-hongkong.cr.aliyuncs.com}"
ACR_VPC_REGISTRY="${ACR_VPC_REGISTRY:-saidc-registry-vpc.cn-hongkong.cr.aliyuncs.com}"
BASE_IMAGE="${BASE_IMAGE:-${REGISTRY}/base/saidc-uv:0.12.0}"

require_env() {
  ENV="${ENV:-office}"
  if [[ "$ENV" != "office" && "$ENV" != "hk" ]]; then
    echo "ENV=office|hk" >&2
    exit 1
  fi
}

require_repo() {
  : "${IMAGE:?set IMAGE (e.g. tm/tm-cs)}"
  : "${ROOT:?set ROOT to the git checkout}"
}

commit_id() {
  CI_COMMIT_ID="${CI_COMMIT_ID:-${CI_COMMIT_SHA:-}}"
  if [[ -z "${CI_COMMIT_ID}" ]]; then
    CI_COMMIT_ID="$(git -C "$ROOT" rev-parse HEAD 2>/dev/null || true)"
  fi
  if [[ -z "${CI_COMMIT_ID}" ]]; then
    echo "error: CI_COMMIT_ID unset" >&2
    exit 1
  fi
  export CI_COMMIT_ID
  export IMAGE_TAG="dev-${CI_COMMIT_ID}"
  export ACR_TAG="${CI_COMMIT_ID}"
  export FULL_IMAGE="${REGISTRY}/${IMAGE}:${IMAGE_TAG}"
  export ACR_IMAGE="${ACR_REGISTRY}/${IMAGE}:${ACR_TAG}"
  export ACR_VPC_IMAGE="${ACR_VPC_REGISTRY}/${IMAGE}:${ACR_TAG}"
}

harbor_login() {
  local secret=""
  for secret in /root/.harbor/robot-saidc.secret \
    "${HOME}/.harbor/robot-saidc.secret" \
    /home/saidc/.harbor/robot-saidc.secret; do
    if [[ -f "$secret" ]]; then
      docker login "$REGISTRY" -u 'robot$saidc' --password-stdin <"$secret" >/dev/null
      return 0
    fi
  done
  echo "error: missing Harbor robot secret" >&2
  exit 1
}

acr_login() {
  local secret=""
  for secret in /root/.acr/saidc.secret "${HOME}/.acr/saidc.secret" /home/saidc/.acr/saidc.secret; do
    if [[ -f "$secret" ]]; then
      local user pass
      user="$(sed -n '1p' "$secret" | tr -d '\r')"
      pass="$(sed -n '2p' "$secret" | tr -d '\r')"
      printf '%s\n' "$pass" | docker login "$ACR_REGISTRY" -u "$user" --password-stdin >/dev/null
      return 0
    fi
  done
  echo "error: missing ACR secret" >&2
  exit 1
}

find_saidc_ws() {
  local cand
  if [[ -n "${SAIDC_WS:-}" && -f "${SAIDC_WS}/acahti/cicd_acahti/lib.sh" ]]; then
    return 0
  fi
  for cand in \
    "$(cd "${ROOT}/.." && pwd)" \
    /root/saidc-ws \
    /home/saidc/saidc-ws \
    "${HOME}/saidc-ws" \
    "${HOME}/Projects/saidc-ws"; do
    if [[ -f "${cand}/acahti/cicd_acahti/lib.sh" ]]; then
      SAIDC_WS="$cand"
      return 0
    fi
  done
  echo "error: set SAIDC_WS to the workspace that contains acahti/cicd_acahti" >&2
  exit 1
}

find_argos() {
  local cand
  if [[ -n "${ARGOS_ROOT:-}" && -d "${ARGOS_ROOT}" ]]; then
    return 0
  fi
  find_saidc_ws
  if [[ -d "${SAIDC_WS}/argos" ]]; then
    ARGOS_ROOT="${SAIDC_WS}/argos"
    return 0
  fi
  for cand in /root/saidc-ws/argos /home/saidc/saidc-ws/argos \
    "$HOME/saidc-ws/argos" "$HOME/Projects/saidc-ws/argos"; do
    if [[ -d "$cand" ]]; then
      ARGOS_ROOT="$cand"
      return 0
    fi
  done
  echo "error: argos required; set ARGOS_ROOT" >&2
  exit 1
}

deploy_image() {
  if [[ "$ENV" == "office" ]]; then
    echo "${REGISTRY}/${IMAGE}:${IMAGE_TAG}"
  else
    echo "${ACR_VPC_IMAGE}"
  fi
}

deploy_tag() {
  if [[ "$ENV" == "office" ]]; then
    echo "${IMAGE_TAG}"
  else
    echo "${ACR_TAG}"
  fi
}

deploy_repo() {
  if [[ "$ENV" == "office" ]]; then
    echo "${REGISTRY}/${IMAGE}"
  else
    echo "${ACR_VPC_REGISTRY}/${IMAGE}"
  fi
}
