#!/usr/bin/env bash
# After compose is up: admin, org, CI OAuth on the compose net, gateway tokens.
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
if [[ "${ok}" -ne 1 ]]; then
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
    --email "${ACAHTI_ADMIN_USER}@noreply.${DOMAIN}" \
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
oauth_json="$(ACAHTI_ADMIN_TOKEN="${token}" python3 "${root}/scripts/woodpecker-oauth-app.py")"
client_id="$(python3 -c 'import json,sys; print(json.load(sys.stdin)["client_id"])' <<<"${oauth_json}")"
client_secret="$(python3 -c 'import json,sys; print(json.load(sys.stdin)["client_secret"])' <<<"${oauth_json}")"
_upsert_env WOODPECKER_FORGEJO_CLIENT "${client_id}"
_upsert_env WOODPECKER_FORGEJO_SECRET "${client_secret}"
WOODPECKER_FORGEJO_CLIENT="${client_id}"
WOODPECKER_FORGEJO_SECRET="${client_secret}"
"${COMPOSE[@]}" up -d woodpecker

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
if [[ "${ok}" -ne 1 ]]; then
  echo "Woodpecker did not become ready on :8000" >&2
  "${COMPOSE[@]}" logs --tail=80 woodpecker
  exit 1
fi

echo "==> Woodpecker forge session"
wp_token="$(ACAHTI_ADMIN_USER="${ACAHTI_ADMIN_USER}" ACAHTI_ADMIN_PASSWORD="${ACAHTI_ADMIN_PASSWORD}" \
  ROOT_URL="${ROOT_URL}" DOMAIN="${DOMAIN}" \
  python3 "${root}/scripts/woodpecker-oauth.py")"
_upsert_env WOODPECKER_TOKEN "${wp_token}"
WOODPECKER_TOKEN="${wp_token}"
printf '%s\n' "${wp_token}" | sudo tee "${ACAHTI_DATA}/woodpecker.token" >/dev/null
sudo chmod 600 "${ACAHTI_DATA}/woodpecker.token"

if ! curl -fsS \
  -H "Authorization: Bearer ${wp_token}" \
  -H "Cookie: user_sess=${wp_token}" \
  "http://127.0.0.1:8000/ci/api/user/repos?page=1&perPage=1" >/dev/null; then
  echo "Woodpecker cannot list Forgejo repos; forge OAuth is dead" >&2
  "${COMPOSE[@]}" logs --tail=80 woodpecker
  exit 1
fi

printf '%s\n' "${WOODPECKER_AGENT_SECRET}" | sudo tee "${ACAHTI_DATA}/agent.secret" >/dev/null
sudo chmod 600 "${ACAHTI_DATA}/agent.secret"

existing="$(api GET "/api/v1/orgs/${ACAHTI_ORG}/hooks" || echo '[]')"
if ! python3 -c "import json,sys; hooks=json.loads(sys.argv[1]); sys.exit(0 if any('/hooks/forgejo' in (h.get('config') or {}).get('url','') for h in hooks) else 1)" "${existing}"; then
  api POST "/api/v1/orgs/${ACAHTI_ORG}/hooks" "$(python3 -c "import json; print(json.dumps({
    'type': 'gitea',
    'active': True,
    'events': ['create','delete','push','pull_request','pull_request_assign','pull_request_review','pull_request_review_request','pull_request_comment','pull_request_reject','release'],
    'config': {'url': 'http://gateway:8080/hooks/forgejo', 'content_type': 'json', 'http_method': 'post'},
  }))")" >/dev/null || true
fi

"${COMPOSE[@]}" up -d gateway

echo "Install ${ROOT_URL}/skill.md"
echo "Join    ${ROOT_URL}/join     (invite from an admin)"
echo "    admin=${ACAHTI_ADMIN_USER}  org=${ACAHTI_ORG}"
echo "    password is in ${root}/.env (ACAHTI_ADMIN_PASSWORD); do not commit"
