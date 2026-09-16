#!/usr/bin/env bash
set -euo pipefail
# shellcheck source=../_lib.sh
source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/_lib.sh"

# Local analog of oci-gc: bound BuildKit cache on the Runner. Never fail a job.
# Auth stays in isolated HOME; builder instances use persist_buildx_config.

root_used_pct() {
	df -P / | awk 'NR==2 { gsub(/%/, "", $5); print $5 }'
}

keep_storage() {
	local pct
	pct="$(root_used_pct || echo 0)"
	if [[ "$pct" =~ ^[0-9]+$ ]] && ((pct >= 85)); then
		printf '4GB'
		return
	fi
	printf '16GB'
}

list_builders() {
	docker buildx ls 2>/dev/null | awk '
		NR == 1 { next }
		/^[[:space:]]/ { next }
		{
			name = $1
			sub(/\*$/, "", name)
			if (name != "") print name
		}
	'
}

rm_other_builders() {
	local name
	while IFS= read -r name; do
		[[ -z "$name" || "$name" == "$ACAHTI_BUILDER" || "$name" == default ]] && continue
		echo "==> docker-gc rm builder ${name}"
		docker buildx rm --force "$name" >/dev/null 2>&1 || true
	done < <(list_builders)
}

rm_stale_buildkit_volumes() {
	local vol
	while IFS= read -r vol; do
		[[ -z "$vol" ]] && continue
		[[ "$vol" == buildx_buildkit_${ACAHTI_BUILDER}* ]] && continue
		echo "==> docker-gc rm volume ${vol}"
		docker volume rm "$vol" >/dev/null 2>&1 || true
	done < <(docker volume ls -q 2>/dev/null | grep '^buildx_buildkit_' || true)
}

rm_stale_workspaces() {
	command -v find >/dev/null || return 0
	find /tmp -maxdepth 1 -type d -name 'woodpecker-local-*' -mmin +360 -exec rm -rf {} + 2>/dev/null || true
}

rm_leftovers() {
	local name
	while IFS= read -r name; do
		[[ -z "$name" ]] && continue
		echo "==> docker-gc rm leftover ${name}"
		docker rm -f "$name" >/dev/null 2>&1 || true
	done < <(docker ps -a --filter name=turbomesh-storage-dev- --format '{{.Names}}' 2>/dev/null || true)
	if [[ -d /root/yunxiao ]]; then
		echo "==> docker-gc rm /root/yunxiao"
		sudo -n rm -rf /root/yunxiao 2>/dev/null || rm -rf /root/yunxiao 2>/dev/null || true
	fi
}

vacuum_journal() {
	journalctl --vacuum-size=200M >/dev/null 2>&1 || sudo -n journalctl --vacuum-size=200M >/dev/null 2>&1 || true
}

prune_builder() {
	local keep
	keep="$(keep_storage)"
	echo "==> docker-gc ${ACAHTI_BUILDER} keep-storage=${keep} root=$(root_used_pct || echo '?')%"
	docker buildx prune --builder "$ACAHTI_BUILDER" --all --keep-storage "$keep" -f >/dev/null || true
	# Previous CI used the docker driver; that cache sits in containerd, not the acahti volume.
	docker buildx prune --builder default --all -f >/dev/null || true
}

if ! command -v docker >/dev/null; then
	echo "OK docker-gc skipped (no docker)"
	exit 0
fi

ensure_acahti_builder || true
rm_other_builders || true
rm_stale_buildkit_volumes || true
prune_builder || true
docker image prune -f >/dev/null 2>&1 || true
rm_stale_workspaces || true
rm_leftovers || true
vacuum_journal || true
echo "OK docker-gc"
