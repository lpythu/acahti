#!/usr/bin/env bash
# Install an Acahti Runner (local/host executor) on a build or deploy machine (or both).
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
#   WOODPECKER_MAX_WORKFLOWS   # pin parallel workflows; default is host nproc/mem
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
if [[ -n "${WOODPECKER_MAX_WORKFLOWS:-}" ]]; then
  MAX_WORKFLOWS="${WOODPECKER_MAX_WORKFLOWS}"
else
  cpus="$(nproc 2>/dev/null || echo 1)"
  mem_gb="$(awk '/MemTotal:/ {printf "%d", $2/1024/1024}' /proc/meminfo 2>/dev/null || echo 0)"
  if [[ "${mem_gb}" -lt 1 ]]; then
    mem_gb=1
  fi
  by_cpu=$((cpus / 2))
  if [[ "${by_cpu}" -lt 1 ]]; then
    by_cpu=1
  fi
  headroom=$((mem_gb - 2))
  if [[ "${headroom}" -lt 1 ]]; then
    headroom=1
  fi
  by_mem=$((headroom / 4))
  if [[ "${by_mem}" -lt 1 ]]; then
    by_mem=1
  fi
  MAX_WORKFLOWS="${by_cpu}"
  if [[ "${by_mem}" -lt "${MAX_WORKFLOWS}" ]]; then
    MAX_WORKFLOWS="${by_mem}"
  fi
  if [[ "${MAX_WORKFLOWS}" -gt 16 ]]; then
    MAX_WORKFLOWS=16
  fi
  echo "==> capacity cpus=${cpus} memGiB=${mem_gb} max_workflows=${MAX_WORKFLOWS}"
fi

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

echo "==> woodpecker-agent ${VERSION} ${ARCH} labels=${LABELS} max_workflows=${MAX_WORKFLOWS}"
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

install -d -m 0755 /usr/local/lib/acahti/pipes
pipes_src=""
here="$(cd "$(dirname "$0")" && pwd)"
if [[ -d "${here}/pipes" ]]; then
  pipes_src="${here}/pipes"
elif [[ -d "${here}/../pipes" ]]; then
  pipes_src="$(cd "${here}/../pipes" && pwd)"
fi
if [[ -z "${pipes_src}" ]]; then
  echo "error: official pipes not next to agent.sh (expected pipes/)" >&2
  exit 1
fi
cp -a "${pipes_src}/." /usr/local/lib/acahti/pipes/
install -m 0755 /usr/local/lib/acahti/pipes/acahti-pipe /usr/local/bin/acahti-pipe

helm_ver="${HELM_VERSION:-3.18.4}"
if ! command -v helm >/dev/null || ! helm version --short 2>/dev/null | grep -q "v${helm_ver}"; then
  echo "==> helm ${helm_ver}"
  curl -fsSL "https://get.helm.sh/helm-v${helm_ver}-linux-${ARCH}.tar.gz" -o "${tmpdir}/helm.tgz"
  tar -C "$tmpdir" -xzf "${tmpdir}/helm.tgz"
  install -m 0755 "${tmpdir}/linux-${ARCH}/helm" "${bindir}/helm"
fi
crane_ver="${CRANE_VERSION:-0.20.6}"
crane_arch="$ARCH"
[[ "$ARCH" == amd64 ]] && crane_arch=x86_64
if ! command -v crane >/dev/null; then
  echo "==> crane ${crane_ver}"
  curl -fsSL "https://github.com/google/go-containerregistry/releases/download/v${crane_ver}/go-containerregistry_Linux_${crane_arch}.tar.gz" -o "${tmpdir}/crane.tgz"
  tar -C "$tmpdir" -xzf "${tmpdir}/crane.tgz" crane
  install -m 0755 "${tmpdir}/crane" "${bindir}/crane"
fi

if ! command -v argos >/dev/null || ! argos run --help 2>/dev/null | grep -q -- '--dash'; then
  echo "==> argospy (CLI --dash)"
  python3 -m pip install -U 'argospy>=0.5.1'
fi

install -d -m 0755 /etc/woodpecker
buildx_cfg="${run_home}/.docker/buildx"
install -d -o "${run_user}" -g "${run_user}" -m 0755 "${run_home}/.docker" "${buildx_cfg}"
# Jobs append HTTP registries via docker-login with.http. No hostname in this script.
touch /etc/woodpecker/buildkitd.toml
chown "${run_user}:${run_user}" /etc/woodpecker/buildkitd.toml
chmod 644 /etc/woodpecker/buildkitd.toml

if command -v docker >/dev/null; then
  echo "==> buildx builder acahti"
  sudo -u "${run_user}" env BUILDX_CONFIG="${buildx_cfg}" docker buildx rm -f acahti >/dev/null 2>&1 || true
  create_args=(--name acahti --driver docker-container --driver-opt network=host)
  if [[ -s /etc/woodpecker/buildkitd.toml ]]; then
    create_args+=(--config /etc/woodpecker/buildkitd.toml)
  fi
  sudo -u "${run_user}" env BUILDX_CONFIG="${buildx_cfg}" docker buildx create "${create_args[@]}" >/dev/null
fi

cat >"${tmpdir}/agent.env" <<EOF
WOODPECKER_SERVER=${SERVER}
WOODPECKER_AGENT_SECRET=${SECRET}
WOODPECKER_BACKEND=local
WOODPECKER_HOSTNAME=${AGENT_NAME}
WOODPECKER_AGENT_LABELS=${LABELS}
WOODPECKER_MAX_WORKFLOWS=${MAX_WORKFLOWS}
WOODPECKER_HEALTHCHECK=false
BUILDX_CONFIG=${buildx_cfg}
ACAHTI_BUILDKITD_CONFIG=/etc/woodpecker/buildkitd.toml
EOF
chmod 600 "${tmpdir}/agent.env"
# Drop a stale agent id from a previous control plane.
if [[ "${RESET_AGENT_ID:-}" == "1" || ! -s /etc/woodpecker/agent.conf ]]; then
  rm -f /etc/woodpecker/agent.conf
fi
touch /etc/woodpecker/agent.conf
chown "${run_user}:${run_user}" /etc/woodpecker/agent.conf
chmod 600 /etc/woodpecker/agent.conf

cat >"${tmpdir}/agent.service" <<EOF
[Unit]
Description=Acahti Runner (${AGENT_NAME})
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

need_restart=0
if [[ ! -f /etc/woodpecker/agent.env ]] || ! cmp -s "${tmpdir}/agent.env" /etc/woodpecker/agent.env; then
  need_restart=1
fi
if [[ ! -f /etc/systemd/system/woodpecker-agent.service ]] || ! cmp -s "${tmpdir}/agent.service" /etc/systemd/system/woodpecker-agent.service; then
  need_restart=1
fi
if ! systemctl is-active --quiet woodpecker-agent.service; then
  need_restart=1
fi
install -m 0600 "${tmpdir}/agent.env" /etc/woodpecker/agent.env
chown root:root /etc/woodpecker/agent.env
install -m 0644 "${tmpdir}/agent.service" /etc/systemd/system/woodpecker-agent.service

cat >"${tmpdir}/docker-gc.service" <<EOF
[Unit]
Description=Acahti local docker GC
After=docker.service
Requires=docker.service

[Service]
Type=oneshot
User=${run_user}
Group=${run_user}
Environment=BUILDX_CONFIG=${buildx_cfg}
ExecStart=/usr/local/bin/acahti-pipe docker-gc
WorkingDirectory=${run_home}
EOF
cat >"${tmpdir}/docker-gc.timer" <<'EOF'
[Unit]
Description=Acahti local docker GC hourly

[Timer]
OnBootSec=2min
OnUnitActiveSec=1h
Persistent=true

[Install]
WantedBy=timers.target
EOF
install -m 0644 "${tmpdir}/docker-gc.service" /etc/systemd/system/acahti-docker-gc.service
install -m 0644 "${tmpdir}/docker-gc.timer" /etc/systemd/system/acahti-docker-gc.timer

if ((need_restart)); then
  systemctl daemon-reload
  systemctl enable woodpecker-agent.service
  systemctl restart woodpecker-agent.service
  systemctl --no-pager --full status woodpecker-agent.service || true
  echo "OK: runner ${AGENT_NAME} restarted mode=${MODE} -> ${SERVER} labels=${LABELS} pipes=/usr/local/lib/acahti/pipes"
else
  echo "OK: runner ${AGENT_NAME} already active; pipes updated in place"
fi
systemctl daemon-reload
systemctl enable --now acahti-docker-gc.timer
systemctl start --no-block acahti-docker-gc.service || true
systemctl --no-pager --full status acahti-docker-gc.timer || true
