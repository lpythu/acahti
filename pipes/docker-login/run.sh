#!/usr/bin/env bash
set -euo pipefail
# shellcheck source=../_lib.sh
source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/_lib.sh"

require_input REGISTRY
registry="$(input REGISTRY)"
host="$(registry_host "$registry")"
if [[ -z "$host" ]]; then
	echo "error: with.registry must be a hostname (optional http:// or https:// prefix)" >&2
	exit 1
fi
user="$(input USERNAME)"
user="${DOCKER_USERNAME:-$user}"
pass="${DOCKER_PASSWORD:-}"
if [[ -z "$user" || -z "$pass" ]]; then
	echo "error: with.username (or DOCKER_USERNAME) and DOCKER_PASSWORD secret are required" >&2
	exit 1
fi
printf '%s\n' "$pass" | docker login "$host" -u "$user" --password-stdin >/dev/null
if truthy "$(input HTTP)" || [[ "$registry" == http://* ]]; then
	ensure_registry_http "$host"
fi
echo "OK docker-login ${host}"
