#!/usr/bin/env bash
# Publish language packages to this island, or OSS for KIND=oss.
set -euo pipefail
here="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib.sh
source "${here}/lib.sh"
: "${ROOT:?set ROOT}"
: "${KIND:?set KIND}"
cd "$ROOT"

pkg_pypi() {
  local token url
  token="$(acahti_publish_token)"
  url="$(acahti_packages_url)/pypi"
  if [[ -f go.mod ]] && command -v go >/dev/null 2>&1; then
    echo "==> go test"
    export GOPROXY="${GOPROXY:-https://goproxy.cn,https://mirrors.aliyun.com/goproxy/,direct}"
    export GOPRIVATE="${GOPRIVATE:-go.saidc.ai}"
    export GONOSUMDB="${GONOSUMDB:-go.saidc.ai}"
    export GOWORK="${GOWORK:-off}"
    go test ./...
  fi
  command -v uv >/dev/null || {
    echo "uv required" >&2
    exit 1
  }
  rm -rf dist
  uv build
  echo "==> uv publish ${url}"
  UV_PUBLISH_TOKEN="${token}" uv publish --publish-url "$url" dist/*
  if [[ -n "${PKG_POST:-}" && -f "${ROOT}/${PKG_POST}" ]]; then
    echo "==> PKG_POST ${PKG_POST}"
    bash "${ROOT}/${PKG_POST}"
  fi
  echo "OK pypi $(acahti_packages_url)"
}

pkg_npm() {
  local token dest
  token="$(acahti_publish_token)"
  dest="$(mktemp -d)"
  trap 'rm -rf "$dest"' RETURN
  npm_build
  echo "==> npm pack ${PKG_PATH}"
  npm pack --pack-destination "$dest" --ignore-scripts "${ROOT}/${PKG_PATH}"
  echo "==> npm PUT $(acahti_packages_url)/npm"
  ACAHTI_ADMIN_TOKEN="${token}" ACAHTI_ORG="${ACAHTI_ORG:-saidc}" \
    ORIGIN="${ACAHTI_ROOT_URL:-https://acahti.saidc.ai}" \
    python3 "${here}/../scripts/npm-put.py" "$dest"/*.tgz
  echo "OK npm $(acahti_packages_url)"
}

case "$KIND" in
pypi) pkg_pypi ;;
npm) pkg_npm ;;
oss) bash "${here}/pkg-oss.sh" ;;
*)
  echo "error: KIND=${KIND} has no pkg job" >&2
  exit 1
  ;;
esac
