#!/usr/bin/env bash
# Stop compose and empty ACAHTI_DATA. Requires WIPE=1. Does not prune Docker.
set -euo pipefail
# shellcheck source=lib.sh
source "$(cd "$(dirname "$0")" && pwd)/lib.sh"
ensure_env
if [[ "${WIPE:-}" != "1" ]]; then
  echo "refusing: set WIPE=1 to empty ${ACAHTI_DATA}" >&2
  exit 1
fi
root="$(acahti_root)"
cd "$root"
compose_args
"${COMPOSE[@]}" down || true
sudo rm -rf "${ACAHTI_DATA:?}/forgejo" "${ACAHTI_DATA}/woodpecker" "${ACAHTI_DATA}/postgres" "${ACAHTI_DATA}/caddy" "${ACAHTI_DATA}/gateway"
sudo rm -f "${ACAHTI_DATA}/admin.token" "${ACAHTI_DATA}/agent.secret" "${ACAHTI_DATA}/woodpecker.token"
echo "OK: emptied ${ACAHTI_DATA}. run bootstrap.sh next."
