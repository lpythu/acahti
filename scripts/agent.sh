#!/usr/bin/env bash
# Install a Woodpecker local/host agent on a build or deploy machine (or both).
#
# Required:
#   SERVER=192.0.2.10:9000     # acahti host gRPC
#   SECRET=...                 # /var/lib/acahti/agent.secret
# Role (one of):
#   ROLE=build|deploy|both     # standard labels
#   LABELS=key=value,...       # custom extra machine (overrides ROLE labels)
#
# Optional:
#   AGENT_NAME=builder         # WOODPECKER_HOSTNAME
#   VERSION=3.18.1
#   MODE=lan|ssh-reverse       # ssh-reverse expects WOODPECKER_SERVER already 127.0.0.1:9000
#
# Run: sudo -E bash scripts/agent.sh
set -euo pipefail

SECRET="${SECRET:?set SECRET from the acahti host agent.secret}"
SERVER="${SERVER:?set SERVER=host:9000 (gRPC)}"
if [[ -z "${WOODPECKER_AGENT_VERSION:-}" && -f "$(cd "$(dirname "$0")/.." && pwd)/versions.env" ]]; then
  # shellcheck disable=SC1091
  set -a && source "$(cd "$(dirname "$0")/.." && pwd)/versions.env" && set +a
fi
VERSION="${WOODPECKER_AGENT_VERSION:-${WOODPECKER_VERSION:-3.18.1}}"
MODE="${MODE:-lan}"
ROLE="${ROLE:-}"
LABELS="${LABELS:-}"
AGENT_NAME="${AGENT_NAME:-$(hostname -s)}"

if [[ -z "${LABELS}" ]]; then
  case "${ROLE}" in
    build) LABELS="build=true,backend=local" ;;
    deploy) LABELS="deploy=true,backend=local" ;;
    both)
      LABELS="build=true,deploy=true,backend=local"
      ROLE="both"
      ;;
    *)
      echo "set ROLE=build|deploy|both or LABELS=..." >&2
      exit 1
      ;;
  esac
fi

ARCH="$(uname -m)"
case "$ARCH" in
  x86_64 | amd64) ARCH=amd64 ;;
  aarch64 | arm64) ARCH=arm64 ;;
  *)
    echo "unsupported arch: ${ARCH}" >&2
    exit 1
    ;;
esac

if [[ "$(id -u)" -ne 0 ]]; then
  echo "run with sudo -E" >&2
  exit 1
fi

if command -v apt-get >/dev/null; then
  apt-get install -y git-lfs >/dev/null || true
elif command -v dnf >/dev/null; then
  dnf install -y git-lfs >/dev/null || true
fi

run_user="${SUDO_USER:-}"
if [[ -z "${run_user}" ]] || ! id -u "$run_user" >/dev/null 2>&1; then
  run_user=root
fi
run_home="$(getent passwd "$run_user" | cut -d: -f6)"
run_home="${run_home:-/}"
bindir=/usr/local/bin
tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"' EXIT

echo "==> woodpecker-agent ${VERSION} ${ARCH} labels=${LABELS}"
need_fetch=1
if [[ -x "${bindir}/woodpecker-agent" && "${FORCE_AGENT:-}" != "1" ]]; then
  have="$("${bindir}/woodpecker-agent" --version 2>/dev/null || true)"
  if [[ "${have}" == *"${VERSION}"* ]]; then
    echo "    using existing ${bindir}/woodpecker-agent (${VERSION})"
    need_fetch=0
  fi
fi
if [[ "${need_fetch}" -eq 1 ]]; then
  url="https://github.com/woodpecker-ci/woodpecker/releases/download/v${VERSION}/woodpecker-agent_linux_${ARCH}.tar.gz"
  curl -fsSL "$url" -o "${tmpdir}/agent.tgz"
  tar -C "$tmpdir" -xzf "${tmpdir}/agent.tgz"
  install -m 0755 "${tmpdir}/woodpecker-agent" "${bindir}/woodpecker-agent"
fi

if [[ ! -x "${bindir}/plugin-git" ]]; then
  plugin_url="https://github.com/woodpecker-ci/plugin-git/releases/download/2.10.1/linux-${ARCH}_plugin-git"
  if curl -fsSL "$plugin_url" -o "${tmpdir}/plugin-git"; then
    install -m 0755 "${tmpdir}/plugin-git" "${bindir}/plugin-git"
  fi
fi

runner_dst=/usr/local/lib/acahti/runner
runner_src="$(cd "$(dirname "$0")/../runner" && pwd)"
if [[ -f "${runner_src}/run.sh" ]]; then
  install -d -m 0755 "${runner_dst}"
  cp -a "${runner_src}/." "${runner_dst}/"
  ACAHTI_RUNNER="${runner_dst}"
fi
ACAHTI_RUNNER="${ACAHTI_RUNNER:-}"
if [[ -z "$ACAHTI_RUNNER" || ! -f "${ACAHTI_RUNNER}/run.sh" ]]; then
  echo "error: run agent.sh from an acahti tree so it can install runner/" >&2
  exit 1
fi

install -d -m 0755 /etc/woodpecker
cat >/etc/woodpecker/agent.env <<EOF
WOODPECKER_SERVER=${SERVER}
WOODPECKER_AGENT_SECRET=${SECRET}
WOODPECKER_BACKEND=local
WOODPECKER_HOSTNAME=${AGENT_NAME}
WOODPECKER_AGENT_LABELS=${LABELS}
WOODPECKER_MAX_WORKFLOWS=4
WOODPECKER_HEALTHCHECK=false
ACAHTI_RUNNER=${ACAHTI_RUNNER}
EOF
chmod 600 /etc/woodpecker/agent.env
chown root:root /etc/woodpecker/agent.env
# Drop a stale agent id from a previous control plane.
if [[ "${RESET_AGENT_ID:-}" == "1" || ! -s /etc/woodpecker/agent.conf ]]; then
  rm -f /etc/woodpecker/agent.conf
fi
touch /etc/woodpecker/agent.conf
chown "${run_user}:${run_user}" /etc/woodpecker/agent.conf
chmod 600 /etc/woodpecker/agent.conf

cat >/etc/systemd/system/woodpecker-agent.service <<EOF
[Unit]
Description=Acahti CI agent (${AGENT_NAME})
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=${run_user}
Group=${run_user}
EnvironmentFile=/etc/woodpecker/agent.env
ExecStart=${bindir}/woodpecker-agent
Restart=always
RestartSec=5
WorkingDirectory=${run_home}

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload
systemctl enable woodpecker-agent.service
systemctl restart woodpecker-agent.service
systemctl --no-pager --full status woodpecker-agent.service || true
echo "OK: agent ${AGENT_NAME} mode=${MODE} -> ${SERVER} labels=${LABELS}"
