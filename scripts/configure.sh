#!/usr/bin/env bash
# After up.sh: admin, org, OAuth, gateway tokens, default policy.
set -euo pipefail
# shellcheck source=lib.sh
source "$(cd "$(dirname "$0")" && pwd)/lib.sh"

need_cmd docker
need_cmd python3
ensure_env
root="$(acahti_root)"
cd "$root"
compose_args

echo "==> wait for Forgejo on 127.0.0.1:3000"
ok=0
for _ in $(seq 1 90); do
  if curl -fsS -o /dev/null http://127.0.0.1:3000/; then
    ok=1
    break
  fi
  sleep 2
done
if [[ "$ok" -ne 1 ]]; then
  echo "Forgejo did not become ready on :3000" >&2
  "${COMPOSE[@]}" logs --tail=80 forgejo
  exit 1
fi

fj() {
  "${COMPOSE[@]}" exec -T --user "${FORGEJO_UID}:${FORGEJO_GID}" forgejo forgejo "$@"
}

echo "==> admin ${ACAHTI_ADMIN_USER}"
if ! fj admin user list 2>/dev/null | grep -q "${ACAHTI_ADMIN_USER}"; then
  fj admin user create \
    --admin \
    --username "${ACAHTI_ADMIN_USER}" \
    --password "${ACAHTI_ADMIN_PASSWORD}" \
    --email "${ACAHTI_ADMIN_EMAIL}" \
    --must-change-password=false
fi

token_file="${ACAHTI_DATA}/admin.token"
if [[ ! -s "${token_file}" ]]; then
  token="$(fj admin user generate-access-token \
    --username "${ACAHTI_ADMIN_USER}" \
    --token-name acahti-bootstrap \
    --scopes all \
    --raw)"
  printf '%s\n' "${token}" | sudo tee "${token_file}" >/dev/null
  sudo chmod 600 "${token_file}"
  sudo chown "${FORGEJO_UID}:${FORGEJO_GID}" "${token_file}"
fi
token="$(sudo cat "${token_file}")"
_upsert_env ACAHTI_ADMIN_TOKEN "${token}"
ACAHTI_ADMIN_TOKEN="${token}"

api() {
  local method="$1" path="$2" data="${3:-}"
  if [[ -n "$data" ]]; then
    curl -fsS -X "$method" \
      -H "Authorization: token ${token}" \
      -H "Content-Type: application/json" \
      -d "$data" \
      "http://127.0.0.1:3000${path}"
  else
    curl -fsS -X "$method" \
      -H "Authorization: token ${token}" \
      "http://127.0.0.1:3000${path}"
  fi
}

echo "==> org ${ACAHTI_ORG}"
if ! api GET "/api/v1/orgs/${ACAHTI_ORG}" >/dev/null 2>&1; then
  api POST /api/v1/orgs "{\"username\":\"${ACAHTI_ORG}\",\"full_name\":\"${ACAHTI_ORG}\",\"visibility\":\"private\"}" >/dev/null
fi

echo "==> Woodpecker OAuth app"
if [[ -z "${WOODPECKER_FORGEJO_CLIENT:-}" || -z "${WOODPECKER_FORGEJO_SECRET:-}" ]]; then
  redirect="${ROOT_URL}/ci/authorize"
  loopback="http://127.0.0.1:8000/ci/authorize"
  body="$(python3 -c "import json; print(json.dumps({
    'name': 'acahti-ci',
    'confidential_client': True,
    'redirect_uris': ['${redirect}', '${loopback}'],
  }))")"
  resp="$(api POST /api/v1/user/applications/oauth2 "${body}")"
  client="$(python3 -c "import json,sys; print(json.load(sys.stdin)['client_id'])" <<<"${resp}")"
  secret="$(python3 -c "import json,sys; print(json.load(sys.stdin)['client_secret'])" <<<"${resp}")"
  _upsert_env WOODPECKER_FORGEJO_CLIENT "${client}"
  _upsert_env WOODPECKER_FORGEJO_SECRET "${secret}"
  WOODPECKER_FORGEJO_CLIENT="${client}"
  WOODPECKER_FORGEJO_SECRET="${secret}"
  "${COMPOSE[@]}" up -d woodpecker
fi

echo "==> wait for Woodpecker on 127.0.0.1:8000"
ok=0
for _ in $(seq 1 60); do
  if curl -fsS -o /dev/null http://127.0.0.1:8000/ci >/dev/null 2>&1 \
    || curl -fsS -o /dev/null http://127.0.0.1:8000/ >/dev/null 2>&1; then
    ok=1
    break
  fi
  sleep 2
done
if [[ "$ok" -ne 1 ]]; then
  echo "Woodpecker did not become ready on :8000" >&2
  "${COMPOSE[@]}" logs --tail=80 woodpecker
  exit 1
fi

if [[ -z "${WOODPECKER_TOKEN:-}" ]]; then
  echo "==> Woodpecker token"
  wp_token="$(ACAHTI_ADMIN_USER="${ACAHTI_ADMIN_USER}" ACAHTI_ADMIN_PASSWORD="${ACAHTI_ADMIN_PASSWORD}" \
    ROOT_URL="${ROOT_URL}" DOMAIN="${DOMAIN}" \
    python3 "${root}/scripts/woodpecker-oauth.py")"
  _upsert_env WOODPECKER_TOKEN "${wp_token}"
  WOODPECKER_TOKEN="${wp_token}"
  printf '%s\n' "${wp_token}" | sudo tee "${ACAHTI_DATA}/woodpecker.token" >/dev/null
  sudo chmod 600 "${ACAHTI_DATA}/woodpecker.token"
fi

printf '%s\n' "${WOODPECKER_AGENT_SECRET}" | sudo tee "${ACAHTI_DATA}/agent.secret" >/dev/null
sudo chmod 600 "${ACAHTI_DATA}/agent.secret"

hook_url="${ROOT_URL}/hooks/forgejo"
# Prefer compose-net so hooks work before public DNS.
existing="$(api GET "/api/v1/orgs/${ACAHTI_ORG}/hooks" || echo '[]')"
if ! python3 -c "import json,sys; hooks=json.loads(sys.argv[1]); sys.exit(0 if any('/hooks/forgejo' in (h.get('config') or {}).get('url','') for h in hooks) else 1)" "${existing}"; then
  api POST "/api/v1/orgs/${ACAHTI_ORG}/hooks" "$(python3 -c "import json; print(json.dumps({
    'type': 'gitea',
    'active': True,
    'events': ['create','delete','push','pull_request','pull_request_assign','pull_request_review','pull_request_review_request','pull_request_comment','pull_request_reject','release'],
    'config': {'url': 'http://gateway:8080/hooks/forgejo', 'content_type': 'json', 'http_method': 'post'},
  }))")" >/dev/null || true
fi

# Reload gateway with tokens.
"${COMPOSE[@]}" up -d gateway

echo "OK: admin=${ACAHTI_ADMIN_USER}  org=${ACAHTI_ORG}  mcp=${ROOT_URL}/mcp"
echo "    password is in ${root}/.env (ACAHTI_ADMIN_PASSWORD); do not commit"
echo "    invite users at ${ROOT_URL}/users"
