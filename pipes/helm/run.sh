#!/usr/bin/env bash
set -euo pipefail
# shellcheck source=../_lib.sh
source "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/_lib.sh"

require_input RELEASE
require_input NAMESPACE
data="${KUBECONFIG:-}"
if [[ -z "$data" ]]; then
	echo "error: KUBECONFIG secret is required" >&2
	exit 1
fi
if [[ "$data" == /* && "$data" != *$'\n'* ]]; then
	echo "error: KUBECONFIG must be the kubeconfig document from a pipeline secret, not a host path" >&2
	exit 1
fi

chart="$(input CHART)"
chart="${chart:-chart}"
if [[ ! -f "${ROOT}/${chart}/Chart.yaml" ]]; then
	echo "error: chart not found at ${ROOT}/${chart}" >&2
	exit 1
fi

timeout="$(input TIMEOUT)"
timeout="${timeout:-5m}"
release="$(input RELEASE)"
ns="$(input NAMESPACE)"
jump="$(input JUMP)"

files=()
while IFS= read -r f; do
	[[ -z "$f" ]] && continue
	rel="${f#"${ROOT}"/}"
	[[ "$rel" == /* ]] && {
		echo "error: values file must be in the workspace: ${f}" >&2
		exit 1
	}
	if [[ ! -f "${ROOT}/${rel}" ]]; then
		echo "error: values file not found: ${ROOT}/${rel}" >&2
		exit 1
	fi
	files+=("$rel")
done < <(each_line "$(input FILES)")

sets=()
while IFS= read -r spec; do
	[[ -z "$spec" ]] && continue
	sets+=("$spec")
done < <(each_line "$(expand_ci "$(input SET)")")

cfg="$(mktemp)"
cleanup() { rm -f "$cfg"; }
trap cleanup EXIT
printf '%s\n' "$data" >"$cfg"
chmod 600 "$cfg"

ws="$ROOT"
kube="$cfg"
if [[ -n "$jump" ]]; then
	remote="$(ssh -o BatchMode=yes "$jump" mktemp -d)"
	cleanup() {
		rm -f "$cfg"
		ssh -o BatchMode=yes "$jump" "rm -rf '${remote}'" || true
	}
	trap cleanup EXIT
	scp -o BatchMode=yes "$cfg" "${jump}:${remote}/kubeconfig"
	tar -C "$ROOT" --exclude .git -cf - . | ssh -o BatchMode=yes "$jump" "tar -C '${remote}' -xf -"
	ws="$remote"
	kube="${remote}/kubeconfig"
fi

args=(
	upgrade --install "$release" "${ws}/${chart}"
	-n "$ns" --create-namespace
	--wait --timeout "$timeout"
)
[[ "$(input TAKE_OWNERSHIP)" == "true" ]] && args+=(--take-ownership)
for f in "${files[@]}"; do
	args+=(-f "${ws}/${f}")
done
for spec in "${sets[@]}"; do
	args+=(--set "$spec")
done

echo "==> helm ${release} -n ${ns}${jump:+ via ${jump}}"
if [[ -z "$jump" ]]; then
	KUBECONFIG="$kube" helm "${args[@]}"
else
	ssh -o BatchMode=yes "$jump" "KUBECONFIG=$(printf '%q' "$kube") helm $(printf '%q ' "${args[@]}")"
fi
echo "OK helm ${release}"
