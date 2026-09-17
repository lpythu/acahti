#!/usr/bin/env bash
set -euo pipefail
# shellcheck source=../_lib.sh
source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/_lib.sh"

# Local analog of oci-gc: bound BuildKit cache on the Runner. Never fail a job.
# Hourly timer only (agent.sh). docker-build must not call this.
# Auth stays in isolated HOME; builder instances use persist_buildx_config.

root_used_pct() {
	df -Pk / | awk 'NR==2 { gsub(/%/, "", $5); print $5 }'
}

# keep = min(avail - headroom, 25% of disk), headroom = max(20GB, 15% of disk),
# clamp 8GB..64GB. avail < 20GB → 8GB (warn). ACAHTI_BUILDKIT_KEEP overrides.
keep_storage() {
	local raw="${ACAHTI_BUILDKIT_KEEP:-}"
	if [[ -n "$raw" ]]; then
		if [[ "$raw" =~ ^[0-9]+$ ]]; then
			printf '%sGB' "$raw"
		else
			printf '%s' "$raw"
		fi
		return
	fi
	local total_k avail_k total_gb avail_gb headroom cap keep
	read -r total_k avail_k < <(df -Pk / | awk 'NR==2 {print $2, $4}')
	if [[ ! "$total_k" =~ ^[0-9]+$ || ! "$avail_k" =~ ^[0-9]+$ ]]; then
		printf '16GB'
		return
	fi
	total_gb=$((total_k / 1024 / 1024))
	avail_gb=$((avail_k / 1024 / 1024))
	if ((avail_gb < 20)); then
		echo "==> docker-gc warn: avail=${avail_gb}GB < 20GB, keep=8GB" >&2
		printf '8GB'
		return
	fi
	headroom=$((total_gb * 15 / 100))
	((headroom < 20)) && headroom=20
	cap=$((total_gb * 25 / 100))
	keep=$((avail_gb - headroom))
	((keep > cap)) && keep=$cap
	((keep < 8)) && keep=8
	((keep > 64)) && keep=64
	printf '%sGB' "$keep"
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
}

vacuum_journal() {
	journalctl --vacuum-size=200M >/dev/null 2>&1 || sudo -n journalctl --vacuum-size=200M >/dev/null 2>&1 || true
}

prune_builder() {
	local keep
	keep="$(keep_storage)"
	echo "==> docker-gc ${ACAHTI_BUILDER} keep-storage=${keep} root=$(root_used_pct || echo '?')%"
	# Unused layers only. --all would sweep cache mounts that --pull still needs.
	docker buildx prune --builder "$ACAHTI_BUILDER" --keep-storage "$keep" -f >/dev/null || true
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
