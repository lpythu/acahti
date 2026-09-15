# kube / helm helpers. Source after lib.sh.
# ssh office / ssh thk only lives here.

kube_setup() {
  require_env
  ACAHTI_HELM_HOST=""
  case "$ENV" in
  office)
    if [[ -n "${KUBECONFIG_OFFICE:-}" && -f "${KUBECONFIG_OFFICE}" ]]; then
      export KUBECONFIG="${KUBECONFIG_OFFICE}"
      return 0
    fi
    if [[ -f "${HOME}/.kube/office" ]]; then
      export KUBECONFIG="${HOME}/.kube/office"
      return 0
    fi
    if [[ -f "${HOME}/.kube/config" ]] && kubectl --kubeconfig="${HOME}/.kube/config" config current-context 2>/dev/null | grep -qi office; then
      export KUBECONFIG="${HOME}/.kube/config"
      return 0
    fi
    if [[ -f "${HOME}/.kube/config" ]]; then
      export KUBECONFIG="${HOME}/.kube/config"
      return 0
    fi
    ACAHTI_HELM_HOST="${HELM_HOST_OFFICE:-office}"
    ;;
  hk)
    if [[ -n "${KUBECONFIG_HK:-}" && -f "${KUBECONFIG_HK}" ]]; then
      export KUBECONFIG="${KUBECONFIG_HK}"
      return 0
    fi
    if [[ -f "${HOME}/.kube/hk" ]]; then
      export KUBECONFIG="${HOME}/.kube/hk"
      return 0
    fi
    ACAHTI_HELM_HOST="${HELM_HOST_HK:-thk}"
    ;;
  esac
}

helm_upgrade() {
  local chart="$1"
  local release="$2"
  local ns="$3"
  shift 3
  local -a sets=("$@")
  local args=(
    upgrade --install "$release" "$chart"
    -n "$ns" --create-namespace
    --take-ownership
    --wait --timeout 5m
  )
  if [[ -f "${chart}/values.yaml" ]]; then
    args+=(-f "${chart}/values.yaml")
  fi
  if [[ -f "${chart}/values-${ENV}.yaml" ]]; then
    args+=(-f "${chart}/values-${ENV}.yaml")
  fi
  args+=("${sets[@]}")
  kube_setup
  if [[ -z "${ACAHTI_HELM_HOST}" ]]; then
    helm "${args[@]}"
    return 0
  fi
  local remote quoted="" item
  remote="$(ssh -o BatchMode=yes "${ACAHTI_HELM_HOST}" mktemp -d)"
  tar -C "$(dirname "$chart")" -cf - "$(basename "$chart")" |
    ssh -o BatchMode=yes "${ACAHTI_HELM_HOST}" "tar -C '${remote}' -xf -"
  for item in "${sets[@]}"; do
    quoted+=" $(printf '%q' "$item")"
  done
  ssh -o BatchMode=yes "${ACAHTI_HELM_HOST}" \
    "helm upgrade --install '${release}' '${remote}/$(basename "$chart")' \
      -n '${ns}' --create-namespace \
      -f '${remote}/$(basename "$chart")/values.yaml' \
      -f '${remote}/$(basename "$chart")/values-${ENV}.yaml' \
      ${quoted} \
      --take-ownership --wait --timeout 5m"
  ssh -o BatchMode=yes "${ACAHTI_HELM_HOST}" "rm -rf '${remote}'"
}
