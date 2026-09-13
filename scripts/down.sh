#!/usr/bin/env bash
# Stop containers. Does not delete /var/lib/acahti.
set -euo pipefail
# shellcheck source=lib.sh
source "$(cd "$(dirname "$0")" && pwd)/lib.sh"
ensure_env
root="$(acahti_root)"
cd "$root"
compose_args
"${COMPOSE[@]}" down
echo "OK: stopped. data left in ${ACAHTI_DATA:-/var/lib/acahti}"
