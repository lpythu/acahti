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

npm_token_file() {
  local token="" f
  if [[ -n "${NPM_TOKEN:-}" ]]; then
    token="${NPM_TOKEN}"
  else
    for f in /root/.npm/saidc.token "${HOME}/.npm/saidc.token" /home/saidc/.npm/saidc.token; do
      if [[ -f "$f" ]]; then
        token="$(tr -d '\r\n' <"$f")"
        break
      fi
    done
  fi
  unset NPM_TOKEN
  if [[ -z "$token" ]]; then
    return 0
  fi
  NPM_TOKEN_FILE="$(mktemp)"
  chmod 0600 "$NPM_TOKEN_FILE"
  printf '%s' "$token" >"$NPM_TOKEN_FILE"
  export NPM_TOKEN_FILE
}

codeup_netrc_file() {
  local user="" pass="" f line
  if [[ -n "${CODEUP_NETRC_FILE:-}" && -f "${CODEUP_NETRC_FILE}" ]]; then
    export CODEUP_NETRC_FILE
    return 0
  fi
  if [[ -n "${CODEUP_USER:-}" && -n "${CODEUP_PASSWORD:-}" ]]; then
    user="${CODEUP_USER}"
    pass="${CODEUP_PASSWORD}"
  else
    for f in /root/.harbor/codeup.env "${HOME}/.harbor/codeup.env" /home/saidc/.harbor/codeup.env; do
      if [[ -f "$f" ]]; then
        # shellcheck disable=SC1090
        set -a && source "$f" && set +a
        user="${CODEUP_USER:-}"
        pass="${CODEUP_PASSWORD:-}"
        break
      fi
    done
  fi
  unset CODEUP_USER CODEUP_PASSWORD
  if [[ -z "$user" || -z "$pass" ]]; then
    return 0
  fi
  CODEUP_NETRC_FILE="$(mktemp)"
  chmod 0600 "$CODEUP_NETRC_FILE"
  printf 'machine codeup.aliyun.com login %s password %s\n' "$user" "$pass" >"$CODEUP_NETRC_FILE"
  export CODEUP_NETRC_FILE
}

npm_build() {
  : "${PKG_PATH:?set PKG_PATH (package directory)}"
  : "${PKG_BUILD:?set PKG_BUILD in .acahti/repo.env}"
  echo "==> PKG_BUILD"
  (cd "$ROOT" && eval "${PKG_BUILD}")
  if [[ ! -d "${ROOT}/${PKG_PATH}/dist" ]] || [[ -z "$(ls -A "${ROOT}/${PKG_PATH}/dist" 2>/dev/null)" ]]; then
    echo "error: ${PKG_PATH}/dist empty after PKG_BUILD" >&2
    exit 1
  fi
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

deploy_registry() {
  if [[ "$ENV" == "office" ]]; then
    echo "${REGISTRY}"
  else
    echo "${ACR_VPC_REGISTRY}"
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

helm_image_sets() {
  : "${CD_IMAGES:?set CD_IMAGES in .acahti/repo.env}"
  local tag reg line key path
  tag="$(deploy_tag)"
  reg="$(deploy_registry)"
  while IFS= read -r line; do
    # shellcheck disable=SC2086
    set -- $line
    key="$1"
    path="$2"
    if [[ -z "$key" || -z "$path" || -n "${3:-}" ]]; then
      echo "error: CD_IMAGES line must be '<workloadKey> <harbor/path>'" >&2
      exit 1
    fi
    printf '%s\n' "--set" "workloads.${key}.image.repository=${reg}/${path}"
    printf '%s\n' "--set" "workloads.${key}.image.tag=${tag}"
  done < <(each_line "$CD_IMAGES")
}

wait_public_urls() {
  local blob=""
  case "$ENV" in
  office) blob="${PUBLIC_WAIT_URLS_office:-${PUBLIC_WAIT_URLS:-}}" ;;
  hk) blob="${PUBLIC_WAIT_URLS_hk:-${PUBLIC_WAIT_URLS:-}}" ;;
  esac
  if [[ -z "$blob" ]]; then
    echo "error: set PUBLIC_WAIT_URLS or PUBLIC_WAIT_URLS_${ENV} in .acahti/repo.env" >&2
    exit 1
  fi
  if [[ "$blob" == "-" ]]; then
    echo "==> wait public urls skipped"
    return 0
  fi
  local url deadline code streak
  deadline=$((SECONDS + 180))
  IFS=';' read -r -a urls <<<"$blob"
  for url in "${urls[@]}"; do
    url="${url#"${url%%[![:space:]]*}"}"
    url="${url%"${url##*[![:space:]]}"}"
    [[ -z "$url" ]] && continue
    echo "==> wait ${url}"
    streak=0
    while ((SECONDS < deadline)); do
      code="$(curl -sS -o /dev/null -w '%{http_code}' -A acahti-cd/1.0 --max-time 10 "$url" || true)"
      if [[ "$code" != "502" && "$code" != "503" && "$code" != "504" && "$code" != "000" ]]; then
        streak=$((streak + 1))
        if ((streak >= 3)); then
          echo "OK ${url} ${code}"
          continue 2
        fi
        sleep 2
        continue
      fi
      streak=0
      sleep 3
    done
    echo "error: ${url} still ${code:-000} after 3m" >&2
    exit 1
  done
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
