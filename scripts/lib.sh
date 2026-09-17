# Shared helpers. Source only; do not execute.
# shellcheck shell=bash

acahti_root() {
  cd "$(dirname "${BASH_SOURCE[0]}")/.."
  pwd
}

gen_secret() {
  openssl rand -hex 24
}

_upsert_env() {
  local key="$1" val="$2" root file
  root="$(acahti_root)"
  file="${root}/.env"
  if grep -q "^${key}=" "$file" 2>/dev/null; then
    python3 - "$file" "$key" "$val" <<'PY'
import pathlib, sys
path, key, val = pathlib.Path(sys.argv[1]), sys.argv[2], sys.argv[3]
lines = path.read_text().splitlines()
out = []
found = False
for line in lines:
    if line.startswith(key + "="):
        out.append(f"{key}={val}")
        found = True
    else:
        out.append(line)
if not found:
    out.append(f"{key}={val}")
path.write_text("\n".join(out) + "\n")
PY
  else
    printf '%s=%s\n' "$key" "$val" >>"$file"
  fi
}

need_cmd() {
  command -v "$1" >/dev/null || {
    echo "need $1" >&2
    exit 1
  }
}

compose_args() {
  local root
  root="$(acahti_root)"
  COMPOSE=(docker compose --env-file "${root}/.env")
  if [[ -f "${root}/versions.env" ]]; then
    COMPOSE+=(--env-file "${root}/versions.env")
  fi
}

ensure_env() {
  local root
  root="$(acahti_root)"
  if [[ ! -f "${root}/.env" ]]; then
    cp "${root}/.env.example" "${root}/.env"
  fi
  # shellcheck disable=SC1091
  set -a
  source "${root}/.env"
  set +a
  : "${DOMAIN:=acahti.example.com}"
  : "${ROOT_URL:=https://${DOMAIN}}"
  : "${ACAHTI_ORG:=acme}"
  : "${ACAHTI_DATA:=/var/lib/acahti}"
  : "${ACAHTI_ADMIN_USER:=acahti}"
  : "${ACAHTI_ADMIN_EMAIL:=${ACAHTI_ADMIN_USER}@noreply.${DOMAIN}}"
  : "${GATEWAY_BIND:=127.0.0.1:8080}"
  if [[ -z "${POSTGRES_PASSWORD:-}" ]]; then
    POSTGRES_PASSWORD="$(gen_secret)"
    _upsert_env POSTGRES_PASSWORD "${POSTGRES_PASSWORD}"
  fi
  if [[ -z "${WOODPECKER_AGENT_SECRET:-}" ]]; then
    WOODPECKER_AGENT_SECRET="$(gen_secret)"
    _upsert_env WOODPECKER_AGENT_SECRET "${WOODPECKER_AGENT_SECRET}"
  fi
  if [[ -z "${WOODPECKER_GRPC_SECRET:-}" ]]; then
    WOODPECKER_GRPC_SECRET="$(gen_secret)"
    _upsert_env WOODPECKER_GRPC_SECRET "${WOODPECKER_GRPC_SECRET}"
  fi
  if [[ -z "${ACAHTI_ADMIN_PASSWORD:-}" ]]; then
    ACAHTI_ADMIN_PASSWORD="$(gen_secret)"
    _upsert_env ACAHTI_ADMIN_PASSWORD "${ACAHTI_ADMIN_PASSWORD}"
  fi
  if [[ -z "${ACAHTI_SESSION_SECRET:-}" ]]; then
    ACAHTI_SESSION_SECRET="$(gen_secret)"
    _upsert_env ACAHTI_SESSION_SECRET "${ACAHTI_SESSION_SECRET}"
  fi
  if [[ -z "${ACAHTI_CONFIG_TOKEN:-}" ]]; then
    ACAHTI_CONFIG_TOKEN="$(gen_secret)"
    _upsert_env ACAHTI_CONFIG_TOKEN "${ACAHTI_CONFIG_TOKEN}"
  fi
  export DOMAIN ROOT_URL ACAHTI_ORG ACAHTI_DATA ACAHTI_ADMIN_USER ACAHTI_ADMIN_EMAIL
  export GATEWAY_BIND
  export ACAHTI_ADMIN_PASSWORD POSTGRES_PASSWORD WOODPECKER_AGENT_SECRET WOODPECKER_GRPC_SECRET
  export WOODPECKER_FORGEJO_CLIENT WOODPECKER_FORGEJO_SECRET WOODPECKER_TOKEN
  export ACAHTI_ADMIN_TOKEN ACAHTI_SESSION_SECRET ACAHTI_CONFIG_TOKEN
  export FORGEJO_UID="${FORGEJO_UID:-$(id -u)}"
  export FORGEJO_GID="${FORGEJO_GID:-$(id -g)}"
  export ACAHTI_BUILD="${ACAHTI_BUILD:-}"
  export ACAHTI_DEPLOY="${ACAHTI_DEPLOY:-}"
  export ACAHTI_GRPC_HOST="${ACAHTI_GRPC_HOST:-}"
  _upsert_env FORGEJO_UID "${FORGEJO_UID}"
  _upsert_env FORGEJO_GID "${FORGEJO_GID}"
  _upsert_env DOMAIN "${DOMAIN}"
  _upsert_env ROOT_URL "${ROOT_URL}"
  _upsert_env ACAHTI_ORG "${ACAHTI_ORG}"
  _upsert_env GATEWAY_BIND "${GATEWAY_BIND}"
}

# Copy official pipes and restart the Runner on ACAHTI_BUILD (or ACAHTI_DEPLOY).
# Control plane and Runner must share the same train; tag upgrades call this from up.sh.
sync_runner() {
  local spec="${ACAHTI_BUILD:-${ACAHTI_DEPLOY:-}}"
  if [[ -z "$spec" ]]; then
    echo "warn: ACAHTI_BUILD unset; Runner pipes not synced" >&2
    return 0
  fi
  local root secret host
  root="$(acahti_root)"
  secret="${WOODPECKER_AGENT_SECRET:-}"
  if [[ -z "$secret" ]]; then
    echo "error: WOODPECKER_AGENT_SECRET missing; cannot sync Runner" >&2
    return 1
  fi
  if [[ ! -d "${root}/pipes" || ! -f "${root}/scripts/agent.sh" ]]; then
    echo "error: official pipes missing next to agent.sh" >&2
    return 1
  fi
  host="$(hostname -I 2>/dev/null | awk '{print $1}')"
  host="${ACAHTI_GRPC_HOST:-${host}}"
  if [[ -z "$host" ]]; then
    echo "error: set ACAHTI_GRPC_HOST (acahti LAN IP for Runner gRPC)" >&2
    return 1
  fi
  echo "==> runner pipes on ${spec}"
  ssh -o BatchMode=yes "${spec}" "mkdir -p /tmp/acahti-scripts"
  tar -C "${root}" -czf /tmp/acahti-runner.tgz scripts/agent.sh pipes
  scp -o BatchMode=yes /tmp/acahti-runner.tgz "${spec}:/tmp/acahti-runner.tgz"
  rm -f /tmp/acahti-runner.tgz
  ssh -o BatchMode=yes "${spec}" \
    "tar -C /tmp/acahti-scripts --strip-components=0 -xzf /tmp/acahti-runner.tgz && sudo -E ROLE=both SERVER=${host}:9000 SECRET=${secret} bash /tmp/acahti-scripts/scripts/agent.sh"
}
