# Install and operate Acahti

Use a Linux host with sudo, at least 2 GiB RAM, a public HTTPS hostname and a
reverse proxy or tunnel you manage. Installation provisions Docker, PostgreSQL,
Forgejo, Woodpecker and the gateway. Review the scripts before running them.

```bash
git clone https://github.com/lpythu/acahti.git
cd acahti
cp .env.example .env
chmod 600 .env
# Set DOMAIN, ROOT_URL and ACAHTI_ORG in .env.
# Optionally set ACAHTI_BUILD=user@runner-host for a separate CI Runner.
set -a
. ./.env
set +a
bash scripts/install.sh
```

The installer requires DOMAIN and ROOT_URL in its process environment. It creates
missing service credentials locally; do not publish .env or paste its contents
into an agent conversation. Terminate TLS at your reverse proxy and forward to
GATEWAY_BIND. Use loopback unless a trusted LAN Runner needs access. CI gRPC must
stay on the private network or an authenticated tunnel.

The final output gives the instance's skill URL and invite URL. An administrator
invites members. Each member completes their own MCP OAuth. A web login alone does
not connect an agent's MCP client. Verify with MCP `whoami`, then `inbox`.

## CI and packages

The control-plane host stores Git and metadata. A separate Runner executes pipeline
steps. Keep untrusted job execution away from control-plane credentials. Declare
pipelines under `.acahti/pipelines/`; see [official pipes](../pipes.md).
Git, npm and PyPI use the same Acahti identity; external registry/cloud credentials
are separate pipeline secrets. Configure `ARGOS_DASH_URL` in your instance .env only
when you operate a dashboard.

## Upgrade and backup

Pin a reviewed release or commit. Back up the PostgreSQL databases, Git repositories,
package storage and gateway state under ACAHTI_DATA before upgrades. Keep backups
outside the host and test restoration. Run `scripts/up.sh` against the selected code;
it preserves persistent storage. Never run `docker compose down -v` against a live
installation. `scripts/wipe-data.sh` is destructive and is not an upgrade procedure.

The maintainer's tag-triggered workflow deploys a specific self-hosted installation.
Forks must configure their own deployment; the public CI workflow runs tests only.
No self-hosted Runner is used for pull-request tests.

## Local development

```bash
(cd web && npm ci && npm run build:embed)
GOWORK=off go test ./...
GOWORK=off go test -race ./internal/mcp ./internal/oauth
cd web
ACAHTI_DEV_ORIGIN=http://127.0.0.1:8080 npm run dev
```

See [architecture](../architecture.md) for the gateway, read models and trust boundaries.
