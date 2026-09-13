# Acahti

Self-hosted control plane for coding agents. One install: git, required-green checks, language packages, **one MCP, one token**.

Kernels are official **Forgejo** and **Woodpecker** images. This repo is the gateway, compose, and install contract. Do not fork those UIs.

Pinned versions live in [versions.env](versions.env) (`ACAHTI_VERSION` is this repo). Source of truth: **GitHub `lpythu/acahti`**, branch **`main` only**. A tag `vX.Y.Z` (must match `ACAHTI_VERSION`) is what deploys the acahti host. Never `compose down -v`.

## Architecture

```mermaid
flowchart LR
  people[people]
  agents[coding_agents]
  yourEdge[your_proxy_or_tunnel]
  gw[acahti_gateway]
  fj[forgejo]
  wp[woodpecker]
  pg[postgres]
  buildof[host_runner_buildof]

  people --> yourEdge
  agents --> yourEdge
  yourEdge --> gw
  gw --> fj
  gw --> wp
  fj --> pg
  wp --> pg
  wp --> buildof
  agents -->|"git HTTPS"| fj
```

Gateway is the only HTTP app this repo starts. Bind is `GATEWAY_BIND` (default `127.0.0.1:8080`). TLS and the public hostname are **out of tree**: point your reverse proxy or tunnel at that bind and set `ROOT_URL` / `DOMAIN` to the public URL. Public identity is **Acahti**: SPA, MCP, `/acahti/v1`, git HTTPS, and `/api/packages`. Forgejo and Woodpecker stay on the compose network (plus loopback for setup). `/ci` and Forgejo HTML (`/login/oauth`, `/user/login`, `/api/v1`) are not public.

## Host roles

| Role | Runs |
|---|---|
| **acahti** | gateway, Forgejo, Woodpecker server, one Postgres. GitHub Actions self-hosted runner for **this** repo (tag → `up.sh`). No Woodpecker agent. |
| **buildof** | one `woodpecker-agent` with `ROLE=both` (`build=true,deploy=true`) for island product repos |

Product repos **on this island** still use Woodpecker + [cicd_acahti](cicd_acahti/) (`ci.sh` / `cd.sh`) on buildof. `ssh office` / `ssh thk` only appear inside `cicd_acahti/kube.sh`. Those product branches stay `dev` / `test`.

**This repo** (the island itself): `main` only. Release: bump `ACAHTI_VERSION`, `git push origin main`, `bash scripts/tag-release.sh` → tag `v$ACAHTI_VERSION` → GitHub Actions on the **acahti** machine → `scripts/up.sh`.

## Install contract (for an agent)

Chicken and egg: the laptop agent SSHs to an empty host and follows this list. Do not ask a human to click through UIs. Scripts are non-interactive.

1. Probe with `bash scripts/detect.sh`. Stop if no sudo or memory &lt; 2G.
2. Set `DOMAIN` and `ROOT_URL`. Optional: `ACAHTI_ORG`, `ACAHTI_BUILD` (SSH spec for buildof, `ROLE=both`). Laptop or no proxy: `GATEWAY_BIND=0.0.0.0:8080`.
3. Put your reverse proxy or tunnel in front of `GATEWAY_BIND` (default `127.0.0.1:8080`). This repo does not ship Caddy or cloudflared.
4. On the acahti host, as a sudoer: `bash scripts/install.sh`.
   - `bootstrap.sh` — Docker, `/var/lib/acahti/{forgejo,woodpecker,postgres,gateway}`
   - `up.sh` — `compose up` (never `down -v`)
   - `configure.sh` — admin, org, Actions off, OAuth on the compose net, gateway tokens
   - if `ACAHTI_BUILD` is set, SSH-install one `ROLE=both` agent on that host
5. Print `ROOT_URL`, MCP `…/mcp`, invite URL `…/admin/users`. On failure stop and return logs; do not leave a half install.

gRPC is published as `WOODPECKER_GRPC_PUBLISH` (default `127.0.0.1:9000`). With `ACAHTI_BUILD`, install binds the LAN IP. Do not publish it to the internet (`lan` or `ssh-reverse`).

Skills: [skills/acahti-install/SKILL.md](skills/acahti-install/SKILL.md) (bootstrap) and [skills/acahti/SKILL.md](skills/acahti/SKILL.md) (use after URL + token).

## Web preview

Do not tag to look at UI. The SPA is Vite; `/ui` is proxied to the live island.

```bash
bash scripts/web-dev.sh
# http://127.0.0.1:5173  — log in with an island account
```

Override the API: `ACAHTI_DEV_ORIGIN=https://acahti.saidc.ai bash scripts/web-dev.sh`. Confirm there, then bump `ACAHTI_VERSION` and `bash scripts/tag-release.sh`.

## Usage

- Git: `https://$ROOT_URL/$org/$repo.git` (token as password). SSH optional: `ssh://git@$SSH_HOST:$GIT_SSH_PORT/$org/$repo.git`
- Git author on island remotes: MCP `whoami` then `setup_local` (`git config --local` only). Not GitHub `acahti/`.
- Packages: `https://$ROOT_URL/api/packages/$org/pypi/simple/` and `…/npm/`
- MCP: `https://$ROOT_URL/mcp` Bearer token from the account page
- Narrow REST: `/acahti/v1/…` same verbs as MCP
- Not public: Woodpecker `/ci`, Forgejo UI, Forgejo `/api/v1`
- Registration is closed
- Island upgrade: tag `vX.Y.Z` on `lpythu/acahti` (not a push to `main`)

Example values in docs: `https://acahti.example.com`, `org=acme`.

## Data

Bind-mount `${ACAHTI_DATA:-/var/lib/acahti}` only. After data exists, never `compose down -v` and never prune named volumes on that host.

## Versions

See [versions.env](versions.env). Tag the island as `v$ACAHTI_VERSION`. Woodpecker server and every host agent must share the same train (gRPC rejects a mix). Gateway is a Go static binary, memory cap 256M.
