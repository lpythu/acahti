#!/usr/bin/env bash
# Preview the SPA. /ui is proxied to the host :8080 (LAN or Tailscale), not Cloudflare.
set -euo pipefail
root="$(cd "$(dirname "$0")/.." && pwd)"
cd "${root}/web"

on_office_lan() {
  route -n get 192.168.0.180 2>/dev/null | grep -qE 'interface: en'
}

reachable() {
  curl -sf --max-time 0.8 "http://${1}:8080/health" >/dev/null
}

pick_origin() {
  if on_office_lan && reachable 192.168.0.180; then
    echo "http://192.168.0.180:8080"
    return 0
  fi
  if reachable 100.95.96.72; then
    echo "http://100.95.96.72:8080"
    return 0
  fi
  if reachable 192.168.0.180; then
    echo "http://192.168.0.180:8080"
    return 0
  fi
  return 1
}

origin="${ACAHTI_DEV_ORIGIN:-}"
if [[ -z "$origin" ]]; then
  origin="$(pick_origin)" || {
    echo "host :8080 not reachable (tried 192.168.0.180 and 100.95.96.72)" >&2
    exit 1
  }
fi

echo "API ${origin}"
export ACAHTI_DEV_ORIGIN="$origin"
if [[ ! -d node_modules ]]; then
  npm install
fi
exec npm run dev
