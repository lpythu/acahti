#!/usr/bin/env bash
# buildof: docker buildx → Harbor; also ACR when ENV=hk or branch is test.
# KIND=pypi|npm: compile the package; no image.
# Dockerfile owns FROM. Pass extra --build-arg only when repo.env / env sets them.
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
  : "${PKG_PATH:?set PKG_PATH (package directory)}"
  npm pack --ignore-scripts --dry-run "${ROOT}/${PKG_PATH}"
  echo "OK ci npm"
  exit 0
fi

require_repo
commit_id
harbor_login
npm_token

want_acr=0
branch="${CI_COMMIT_BRANCH:-${CI_COMMIT_REF_NAME:-}}"
if [[ "${ENV:-}" == "hk" || "$branch" == "test" ]]; then
  want_acr=1
  acr_login
fi

# Collect --build-arg from: per-image tokens, BUILD_ARGS, and known env keys.
build_args() {
  local spec
  for spec in "$@"; do
    printf '%s\n' "--build-arg" "$spec"
  done
  if [[ -n "${BUILD_ARGS:-}" ]]; then
    local extra
    read -r -a extra <<< "$BUILD_ARGS"
    for spec in "${extra[@]}"; do
      printf '%s\n' "--build-arg" "$spec"
    done
  fi
  if [[ -n "${BASE_IMAGE:-}" ]]; then
    printf '%s\n' "--build-arg" "BASE_IMAGE=${BASE_IMAGE}"
  fi
  if [[ -n "${NPM_TOKEN:-}" ]]; then
    printf '%s\n' "--build-arg" "NPM_TOKEN=${NPM_TOKEN}"
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

if [[ -n "${IMAGES:-}" ]]; then
  while IFS= read -r line || [[ -n "$line" ]]; do
    line="${line#"${line%%[![:space:]]*}"}"
    line="${line%"${line##*[![:space:]]}"}"
    [[ -z "$line" || "$line" == \#* ]] && continue
    # shellcheck disable=SC2206
    fields=($line)
    build_one "${fields[@]}"
  done <<< "$IMAGES"
else
  : "${IMAGE:?set IMAGE (or IMAGES) in .acahti/repo.env}"
  build_one "$IMAGE"
fi
