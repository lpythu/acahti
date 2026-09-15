#!/usr/bin/env bash
# buildof: docker buildx → Harbor; ACR when ENV=hk or a tag.
# KIND=pypi|npm: compile the package; no image.
set -euo pipefail
here="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib.sh
source "${here}/lib.sh"
: "${ROOT:?set ROOT}"
: "${KIND:?set KIND}"
cd "$ROOT"

if [[ "$KIND" == "pypi" ]]; then
  command -v uv >/dev/null || {
    echo "uv required" >&2
    exit 1
  }
  rm -rf dist
  uv build
  echo "OK ci pypi"
  exit 0
fi
if [[ "$KIND" == "npm" ]]; then
  npm_build
  npm pack --ignore-scripts --dry-run "${ROOT}/${PKG_PATH}"
  echo "OK ci npm"
  exit 0
fi

require_repo
commit_id
harbor_login
npm_token_file

want_acr=0
if [[ "${ENV:-}" == "hk" || -n "${CI_COMMIT_TAG:-}" ]]; then
  want_acr=1
  acr_login
fi

build_args() {
  local spec
  for spec in "$@"; do
    printf '%s\n' "--build-arg" "$spec"
  done
  if [[ -n "${BUILD_ARGS:-}" ]]; then
    if [[ "${BUILD_ARGS}" == *TOKEN* || "${BUILD_ARGS}" == *PASS* || "${BUILD_ARGS}" == *SECRET* ]]; then
      echo "error: BUILD_ARGS must not contain secrets" >&2
      exit 1
    fi
    local extra
    read -r -a extra <<<"$BUILD_ARGS"
    for spec in "${extra[@]}"; do
      printf '%s\n' "--build-arg" "$spec"
    done
  fi
  if [[ -n "${BASE_IMAGE:-}" ]]; then
    printf '%s\n' "--build-arg" "BASE_IMAGE=${BASE_IMAGE}"
  fi
}

build_one() {
  local image="$1"
  shift
  local full="${REGISTRY}/${image}:${IMAGE_TAG}"
  local -a args=(
    --builder "${BUILDX_BUILDER:-default}"
    --pull
    --provenance=false
    --network="${DOCKER_BUILD_NETWORK:-host}"
    -t "$full"
    --load
  )
  if [[ -n "${NPM_TOKEN_FILE:-}" ]]; then
    args+=(--secret "id=npm_token,src=${NPM_TOKEN_FILE}")
  fi
  local pair
  while IFS= read -r pair; do
    [[ -n "$pair" ]] && args+=("$pair")
  done < <(build_args "$@")

  echo "==> build ${full}"
  docker buildx build "${args[@]}" .
  echo "==> push ${full}"
  docker push "$full"
  if [[ "$want_acr" -eq 1 ]]; then
    local acr="${ACR_REGISTRY}/${image}:${ACR_TAG}"
    echo "==> tag ${full} → ${acr}"
    docker tag "$full" "$acr"
    echo "==> push ${acr}"
    docker push "$acr"
    echo "k8s pull ref: ${ACR_VPC_REGISTRY}/${image}:${ACR_TAG}"
  fi
  echo "OK ci ${full}"
}

: "${IMAGES:?set IMAGES in .acahti/repo.env}"
while IFS= read -r line; do
  # shellcheck disable=SC2086
  set -- $line
  build_one "$@"
done < <(each_line "$IMAGES")
