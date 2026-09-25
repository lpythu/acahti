# Install and operate Acahti

**Humans do not run the installer.** Paste one line into your coding agent; it gathers config, SSHs to your host, and runs the scripts.

```text
Install https://lpythu.github.io/acahti/install.md
```

Mirror (same skill):  
`https://raw.githubusercontent.com/lpythu/acahti/main/skills/acahti-install/SKILL.md`

You answer questions (domain, SSH, optional Runner). The agent writes `.env`, runs `scripts/install.sh`, and returns `Install …/skill.md` / `Join` / demo prompts. Do not paste `.env` secrets into chat.

## What you need ready

- Linux host with sudo, **≥ 2 GiB RAM**
- Public HTTPS hostname (you terminate TLS at your reverse proxy or tunnel)
- SSH access the agent can use (sudoer, not root)
- Optional second host for the **Runner** (`ACAHTI_BUILD`) — CI must not run on the control plane

## After the agent finishes

1. Open `$ROOT_URL` — copy the **demo prompt** (or `Install $ROOT_URL/demo.md`).
2. Admin invites members; each member completes MCP OAuth (web login alone does not connect MCP).
3. Open **Board** and watch while the agent runs the demo loop. See [product demo](product/demo.md).

## CI and packages

The control-plane host stores Git and metadata. A separate Runner executes pipeline steps. Declare pipelines under `.acahti/pipelines/`; see [official pipes](../pipes.md). Git, npm and PyPI use the same Acahti identity; external registry credentials are pipeline secrets.

## Upgrade and backup

Ask your agent to follow the **Upgrade** section of the install skill. Pin a reviewed release or commit. Back up PostgreSQL, Git repositories, package storage and gateway state under `ACAHTI_DATA` before upgrades. Never `docker compose down -v` on a live install. `scripts/wipe-data.sh` is destructive.

The maintainer's tag-triggered workflow deploys one specific installation. Forks configure their own deploy; public CI runs tests only.

## Local development (contributors)

```bash
(cd web && npm ci && npm run build:embed)
GOWORK=off go test ./...
GOWORK=off go test -race ./internal/mcp ./internal/oauth
cd web
ACAHTI_DEV_ORIGIN=http://127.0.0.1:8080 npm run dev
```

See [architecture](../architecture.md). Script reference for agents: `scripts/install.sh`, `scripts/up.sh`, `scripts/agent.sh`, `.env.example`.
