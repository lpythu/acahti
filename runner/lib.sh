# Shared helpers for acahti/runner. Source only.
set -euo pipefail

REGISTRY="${REGISTRY:-harbor.saidc}"
ACR_REGISTRY="${ACR_REGISTRY:-saidc-registry.cn-hongkong.cr.aliyuncs.com}"
ACR_VPC_REGISTRY="${ACR_VPC_REGISTRY:-saidc-registry-vpc.cn-hongkong.cr.aliyuncs.com}"

require_env() {
  ENV="${ENV:-office}"
  if [[ "$ENV" != "office" && "$ENV" != "hk" ]]; then
    echo "ENV=office|hk" >&2
    exit 1
  fi
}

require_repo() {
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
}

npm_token() {
  if [[ -n "${NPM_TOKEN:-}" ]]; then
    return 0
  fi
  local f
  for f in /root/.npm/saidc.token "${HOME}/.npm/saidc.token" /home/saidc/.npm/saidc.token; do
    if [[ -f "$f" ]]; then
      NPM_TOKEN="$(tr -d '\r\n' <"$f")"
      export NPM_TOKEN
      return 0
    fi
  done
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

acahti_packages_url() {
  echo "${ACAHTI_PACKAGES_URL:-${ACAHTI_ROOT_URL:-https://acahti.saidc.ai}/api/packages/${ACAHTI_ORG:-saidc}}"
}

acahti_publish_token() {
  if [[ -n "${ACAHTI_PUBLISH_TOKEN:-}" ]]; then
    printf '%s' "$ACAHTI_PUBLISH_TOKEN"
    return 0
  fi
  local f
  for f in /root/.harbor/acahti.env "${HOME}/.harbor/acahti.env"; do
    if [[ -f "$f" ]]; then
      # shellcheck disable=SC1090
      set -a && source "$f" && set +a
      if [[ -n "${ACAHTI_PUBLISH_TOKEN:-}" ]]; then
        printf '%s' "$ACAHTI_PUBLISH_TOKEN"
        return 0
      fi
    fi
  done
  echo "error: set ACAHTI_PUBLISH_TOKEN (or /root/.harbor/acahti.env)" >&2
  exit 1
}
