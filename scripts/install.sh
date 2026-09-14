#!/usr/bin/env bash
# Agent install contract. Non-interactive. Run on the acahti host as a sudoer.
# Required in the environment or .env: DOMAIN, ROOT_URL.
set -euo pipefail
# shellcheck source=lib.sh
source "$(cd "$(dirname "$0")" && pwd)/lib.sh"

root="$(acahti_root)"
cd "$root"

if [[ -z "${DOMAIN:-}" || -z "${ROOT_URL:-}" ]]; then
  echo "set DOMAIN and ROOT_URL" >&2
  exit 1
fi

echo "==> detect"
# shellcheck disable=SC1091
eval "$(bash "${root}/scripts/detect.sh")"
_upsert_env DOMAIN "${DOMAIN}"
_upsert_env ROOT_URL "${ROOT_URL}"
_upsert_env GATEWAY_BIND "${GATEWAY_BIND:-127.0.0.1:8080}"

echo "==> bootstrap"
bash "${root}/scripts/bootstrap.sh"

echo "==> up"
bash "${root}/scripts/up.sh"

# shellcheck disable=SC1091
set -a
source "${root}/.env"
set +a
secret="$(sudo cat "${ACAHTI_DATA}/agent.secret")"
acahti_ip="$(hostname -I 2>/dev/null | awk '{print $1}')"
acahti_ip="${ACAHTI_GRPC_HOST:-${acahti_ip}}"

install_remote() {
  local spec="$1" role="$2"
  [[ -z "${spec}" ]] && return 0
  echo "==> agent ${role} on ${spec}"
  ssh -o BatchMode=yes "${spec}" "mkdir -p /tmp/acahti-scripts"
  scp -o BatchMode=yes "${root}/scripts/agent.sh" "${spec}:/tmp/acahti-scripts/agent.sh"
  ssh -o BatchMode=yes "${spec}" \
    "sudo -E ROLE=${role} SERVER=${acahti_ip}:9000 SECRET=${secret} bash /tmp/acahti-scripts/agent.sh"
}

# Default: one ROLE=both on buildof (ACAHTI_BUILD). Do not install on the control plane.
if [[ -n "${ACAHTI_BUILD:-}" ]]; then
  install_remote "${ACAHTI_BUILD}" both
elif [[ -n "${ACAHTI_DEPLOY:-}" ]]; then
  install_remote "${ACAHTI_DEPLOY}" both
fi

echo
echo "Install ${ROOT_URL}/skill.md"
echo "Join    ${ROOT_URL}/join     (invite from an admin)"
echo "    admin   ${ACAHTI_ADMIN_USER}  (password in ${root}/.env)"
