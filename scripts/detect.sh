#!/usr/bin/env bash
# Print install choices for an empty host. Does not change anything.
# stdout: KEY=value lines an agent can source.
set -euo pipefail

mem_kb="$(awk '/MemTotal:/ {print $2}' /proc/meminfo 2>/dev/null || echo 0)"
mem_mb="$((mem_kb / 1024))"
has_docker=0
command -v docker >/dev/null && has_docker=1
has_sudo=0
sudo -n true 2>/dev/null && has_sudo=1 || { [[ "$(id -u)" -eq 0 ]] && has_sudo=1; }

pub_ip=""
if command -v curl >/dev/null; then
  pub_ip="$(curl -fsS --max-time 3 https://ifconfig.me/ip 2>/dev/null || true)"
fi

# Loopback unless the installer already chose a bind. Laptop / no proxy: 0.0.0.0:8080.
bind="${GATEWAY_BIND:-127.0.0.1:8080}"

echo "MEM_MB=${mem_mb}"
echo "HAS_DOCKER=${has_docker}"
echo "HAS_SUDO=${has_sudo}"
echo "PUBLIC_IP=${pub_ip}"
echo "GATEWAY_BIND=${bind}"

if [[ "${has_sudo}" -ne 1 ]]; then
  echo "STOP=need sudo" >&2
  exit 2
fi
if [[ "${mem_mb}" -lt 1800 ]]; then
  echo "STOP=memory below 2G" >&2
  exit 2
fi
