#!/usr/bin/env bash
# Start Acahti. Never pass --volumes / -v.
set -euo pipefail
# shellcheck source=lib.sh
source "$(cd "$(dirname "$0")" && pwd)/lib.sh"

need_cmd docker
ensure_env
root="$(acahti_root)"
cd "$root"
if [[ -f "${root}/versions.env" ]]; then
  set -a
  # shellcheck disable=SC1091
  source "${root}/versions.env"
  set +a
fi

if [[ ! -d "${ACAHTI_DATA}/forgejo" ]]; then
  echo "run scripts/bootstrap.sh first" >&2
  exit 1
fi

df_pct="$(df -P / | awk 'NR==2 {gsub(/%/,"",$5); print $5}')"
if [[ "${df_pct}" -ge 95 ]]; then
  echo "root filesystem ${df_pct}% full; refuse to start" >&2
  exit 1
fi

compose_args
"${COMPOSE[@]}" up -d --build
bash "${root}/scripts/configure.sh"
"${COMPOSE[@]}" ps
echo "OK: gateway ${GATEWAY_BIND}  public ${ROOT_URL}"
echo "    git HTTPS ${ROOT_URL}   woodpecker grpc :9000"
