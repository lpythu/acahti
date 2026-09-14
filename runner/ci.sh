#!/usr/bin/env bash
# buildof: docker buildx → Harbor; also ACR when ENV=hk or branch is test.
set -euo pipefail
here="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib.sh
source "${here}/lib.sh"
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
