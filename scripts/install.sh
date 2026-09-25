#!/usr/bin/env bash
# Runner install contract. Non-interactive. Run on the acahti host as a sudoer.
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

echo
echo "Install ${ROOT_URL}/skill.md"
echo "Install ${ROOT_URL}/demo.md"
echo "Join    ${ROOT_URL}/join     (invite from an admin)"
echo "Paste the home-page demo prompt to an agent, then watch Board."
echo "Next time: Install https://lpythu.github.io/acahti/install.md  (agent-driven upgrade/bootstrap)"
echo "    admin   ${ACAHTI_ADMIN_USER}  (password in ${root}/.env)"
