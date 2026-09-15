#!/usr/bin/env bash
set -euo pipefail
# shellcheck source=../_lib.sh
source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/_lib.sh"

require_input REGISTRY
registry="$(input REGISTRY)"
user="$(input USERNAME)"
user="${DOCKER_USERNAME:-$user}"
pass="${DOCKER_PASSWORD:-}"
if [[ -z "$user" || -z "$pass" ]]; then
	echo "error: with.username (or DOCKER_USERNAME) and DOCKER_PASSWORD secret are required" >&2
	exit 1
fi
printf '%s\n' "$pass" | docker login "$registry" -u "$user" --password-stdin >/dev/null
# FROM bases live on office Harbor. HK ACR login must not drop that pull.
if [[ -n "${HARBOR_PASSWORD:-}" ]]; then
	printf '%s\n' "$HARBOR_PASSWORD" | docker login harbor.saidc -u 'robot$saidc' --password-stdin >/dev/null
fi
echo "OK docker-login ${registry}"
