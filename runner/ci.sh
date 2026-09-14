#!/usr/bin/env bash
# buildof: docker buildx → Harbor; also ACR when ENV=hk or branch is test.
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
  : "${PKG_PATH:?set PKG_PATH (package directory)}"
  npm pack --ignore-scripts --dry-run "${ROOT}/${PKG_PATH}"
  echo "OK ci npm"
  exit 0
fi

require_repo
commit_id
cd "$ROOT"
harbor_login

echo "==> build ${FULL_IMAGE}"
docker buildx build \
  --builder "${BUILDX_BUILDER:-default}" \
  --pull \
  --provenance=false \
  --network="${DOCKER_BUILD_NETWORK:-host}" \
  --build-arg "BASE_IMAGE=${BASE_IMAGE}" \
  -t "${FULL_IMAGE}" \
  --load \
  .

echo "==> push ${FULL_IMAGE}"
docker push "${FULL_IMAGE}"

branch="${CI_COMMIT_BRANCH:-${CI_COMMIT_REF_NAME:-}}"
if [[ "${ENV:-}" == "hk" || "$branch" == "test" ]]; then
  acr_login
  echo "==> tag ${FULL_IMAGE} → ${ACR_IMAGE}"
  docker tag "$FULL_IMAGE" "$ACR_IMAGE"
  echo "==> push ${ACR_IMAGE}"
  docker push "$ACR_IMAGE"
  echo "k8s pull ref: ${ACR_VPC_IMAGE}"
fi

echo "OK ci ${FULL_IMAGE}"
