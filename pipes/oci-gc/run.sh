#!/usr/bin/env bash
set -euo pipefail
# shellcheck source=../_lib.sh
source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/_lib.sh"

require_input REPOS
command -v crane >/dev/null || {
	echo "error: crane required on the Runner" >&2
	exit 1
}
keep="$(input KEEP)"
keep="${keep:-2}"

tag_created() {
	local ref="$1"
	local cfg
	cfg="$(crane config "$ref" 2>/dev/null || true)"
	[[ -z "$cfg" ]] && return 1
	printf '%s' "$cfg" | python3 -c 'import json,sys; print(json.load(sys.stdin).get("created",""))' 2>/dev/null
}

gc_repo() {
	local repo="$1"
	local tag created
	local -a others=()
	local has_latest=0
	echo "==> oci-gc ${repo} keep=${keep} + latest"
	while IFS= read -r tag; do
		[[ -z "$tag" ]] && continue
		if [[ "$tag" == "latest" ]]; then
			has_latest=1
			continue
		fi
		created="$(tag_created "${repo}:${tag}" || true)"
		[[ -z "$created" ]] && created="0000-00-00T00:00:00Z"
		others+=("${created} ${tag}")
	done < <(crane ls "$repo" 2>/dev/null || true)
	if ((${#others[@]} == 0)); then
		echo "    no other tags"
		return 0
	fi
	local -a sorted=()
	# Newest first (ISO-8601 created sorts lexicographically).
	while IFS= read -r line; do
		[[ -z "$line" ]] && continue
		sorted+=("$line")
	done < <(printf '%s\n' "${others[@]}" | sort -r)
	local i=0
	local keep_n="$keep"
	for line in "${sorted[@]}"; do
		tag="${line#* }"
		if ((i < keep_n)); then
			echo "    keep ${tag}"
			i=$((i + 1))
			continue
		fi
		echo "    delete ${tag}"
		# ACR (and some other registries) reject tag-delete with DIGEST_INVALID.
		# Delete by digest; a refused delete must not fail a finished deploy.
		digest="$(crane digest "${repo}:${tag}" 2>/dev/null || true)"
		if [[ -n "$digest" ]] && crane delete "${repo}@${digest}"; then
			continue
		fi
		if crane delete "${repo}:${tag}"; then
			continue
		fi
		echo "    skip ${tag}"
	done
	if ((has_latest == 1)); then
		echo "    keep latest"
	fi
	echo "OK oci-gc ${repo}"
}

while IFS= read -r repo; do
	[[ -z "$repo" ]] && continue
	gc_repo "$repo"
done < <(each_line "$(input REPOS)")
