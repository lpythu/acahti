#!/usr/bin/env bash
# Empty Linux host: Docker + data dirs. Run as a sudoer (not root).
set -euo pipefail
# shellcheck source=lib.sh
source "$(cd "$(dirname "$0")" && pwd)/lib.sh"

if [[ "$(id -u)" -eq 0 ]]; then
  echo "run as a sudoer, not root (Forgejo bind uses this uid)" >&2
  exit 1
fi

need_cmd sudo
ensure_env

echo "==> docker"
if ! command -v docker >/dev/null; then
  curl -fsSL https://get.docker.com | sudo sh
fi
sudo usermod -aG docker "${USER}" || true
if [[ ! -f /etc/docker/daemon.json ]]; then
  sudo tee /etc/docker/daemon.json >/dev/null <<'EOF'
{
  "log-driver": "json-file",
  "log-opts": {
    "max-size": "100m",
    "max-file": "3"
  }
}
EOF
  sudo systemctl restart docker || true
fi
if ! docker info >/dev/null 2>&1; then
  echo "log out/in or run: newgrp docker  (then re-run bootstrap / install)" >&2
fi

echo "==> ${ACAHTI_DATA}"
sudo mkdir -p "${ACAHTI_DATA}"/{forgejo,woodpecker,postgres,gateway}
sudo chown "${FORGEJO_UID}:${FORGEJO_GID}" "${ACAHTI_DATA}"
sudo chown -R "${FORGEJO_UID}:${FORGEJO_GID}" "${ACAHTI_DATA}/forgejo" "${ACAHTI_DATA}/woodpecker" "${ACAHTI_DATA}/gateway"
# Official postgres 16-alpine runs as uid 70.
sudo chown -R 70:70 "${ACAHTI_DATA}/postgres"
sudo chmod 755 "${ACAHTI_DATA}"

echo "==> compose plugin"
if ! docker compose version >/dev/null 2>&1; then
  if command -v apt-get >/dev/null; then
    sudo apt-get update -y
    sudo apt-get install -y docker-compose-plugin
  else
    echo "install docker compose plugin" >&2
    exit 1
  fi
fi

echo "OK: bootstrap. Next: scripts/up.sh && scripts/configure.sh"
