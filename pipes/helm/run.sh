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
cfg="$(mktemp)"
trap 'rm -f "$cfg"' EXIT
printf '%s\n' "$data" >"$cfg"
chmod 600 "$cfg"
export KUBECONFIG="$cfg"

chart="$(input CHART)"
chart="${chart:-chart}"
chart_path="${ROOT}/${chart}"
if [[ ! -f "${chart_path}/Chart.yaml" ]]; then
	echo "error: chart not found at ${chart_path}" >&2
	exit 1
fi

timeout="$(input TIMEOUT)"
timeout="${timeout:-5m}"
release="$(input RELEASE)"
ns="$(input NAMESPACE)"

args=(
	upgrade --install "$release" "$chart_path"
	-n "$ns" --create-namespace
	--wait --timeout "$timeout"
)
if [[ "$(input TAKE_OWNERSHIP)" == "true" ]]; then
	args+=(--take-ownership)
fi
while IFS= read -r f; do
	[[ -z "$f" ]] && continue
	if [[ "$f" != /* ]]; then
		f="${ROOT}/${f}"
	fi
	if [[ ! -f "$f" ]]; then
		echo "error: values file not found: ${f}" >&2
		exit 1
	fi
	args+=(-f "$f")
done < <(each_line "$(input FILES)")
while IFS= read -r spec; do
	[[ -z "$spec" ]] && continue
	args+=(--set "$spec")
done < <(each_line "$(expand_ci "$(input SET)")")

echo "==> helm ${release} -n ${ns}"
helm "${args[@]}"
echo "OK helm ${release}"
