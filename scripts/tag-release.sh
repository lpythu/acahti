#!/usr/bin/env bash
# Cut v$ACAHTI_VERSION from main and push the tag (triggers host deploy).
set -euo pipefail
root="$(cd "$(dirname "$0")/.." && pwd)"
cd "$root"
# shellcheck disable=SC1091
source "${root}/versions.env"
ver="${ACAHTI_VERSION:?set ACAHTI_VERSION in versions.env}"
tag="v${ver}"
if [[ ! "$ver" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
  echo "ACAHTI_VERSION must be X.Y.Z" >&2
  exit 1
fi
branch="$(git rev-parse --abbrev-ref HEAD)"
if [[ "$branch" != "main" ]]; then
  echo "tag from main only (now ${branch})" >&2
  exit 1
fi
if [[ -n "$(git status --porcelain)" ]]; then
  echo "working tree dirty" >&2
  exit 1
fi
if git rev-parse "$tag" >/dev/null 2>&1; then
  echo "tag ${tag} already exists" >&2
  exit 1
fi
git tag -a "$tag" -m "acahti ${tag}"
git push origin main
git push origin "$tag"
echo "OK: pushed ${tag}"
