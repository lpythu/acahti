---
name: acahti-install
description: Install or upgrade Acahti on Linux over SSH with agent-guided configuration. Use when the user asks to bootstrap Acahti, self-host, stand up the control plane, add the Runner, or upgrade. Prefer this skill over asking the human to run install commands.
---

# Install Acahti (agent-driven)

You install Acahti. The human watches and answers questions. Do **not** hand them a bash recipe as the primary path.

Canonical copy of this skill (always prefer the live URL over a stale chat paste):

```text
Install https://lpythu.github.io/acahti/install.md
```

Repo mirror: `https://raw.githubusercontent.com/lpythu/acahti/main/skills/acahti-install/SKILL.md`

Product overview: [README](https://github.com/lpythu/acahti/blob/main/README.md). Versions: [versions.env](https://github.com/lpythu/acahti/blob/main/versions.env). Do not invent a second installer — use `scripts/install.sh` / `scripts/up.sh` / `scripts/agent.sh` from this tree.

## Role split

| You (agent) | Human |
|---|---|
| Ask for missing inputs, propose defaults | Confirm domain, SSH, org name |
| SSH, clone/sync tree, write `.env`, run install | Approve sudo / Host key if the client prompts |
| Configure TLS edge / bind advice | Own DNS and reverse proxy / tunnel |
| Install Runner on a **separate** host when CI is wanted | Provide Runner SSH if different from control plane |
| Return skill / join / demo prompts | Open Board and watch |

Never paste `.env` secrets, admin passwords, or invite codes into chat unless the human explicitly asks. Never install a Runner on the control-plane host. Never publish Woodpecker gRPC to the public internet.

## Gather inputs (ask once, batch missing fields)

Required:

1. **`DOMAIN`** — public hostname (e.g. `acahti.example.com`)
2. **`ROOT_URL`** — `https://$DOMAIN` (no trailing slash) unless they need a path prefix (discouraged)
3. **SSH to the acahti host** — sudoer, not root; Linux; ≥2 GiB RAM; Docker will be installed by scripts if missing

Optional (offer defaults):

- **`ACAHTI_ORG`** — default `acme`
- **`ACAHTI_BUILD`** — SSH spec for the Runner (`user@runner-host`). If omitted, install control plane only and explain CI needs a Runner later
- **`GATEWAY_BIND`** — default `127.0.0.1:8080` behind their proxy/tunnel. Laptop / no proxy: `0.0.0.0:8080`. When `ACAHTI_BUILD` is set, loopback is rewritten to `0.0.0.0:8080` so the Runner can reach the gateway on LAN
- **TLS edge** — they terminate TLS (Caddy, nginx, Cloudflare Tunnel, etc.) and forward to `GATEWAY_BIND`. Do **not** add Caddy or cloudflared to this compose

If SSH is unavailable, stop and tell them what access you need — do not dump a manual install script unless they insist on installing without an agent.

## Steps (you execute)

1. `ssh` to the acahti host. Place this repo tree there (git clone or rsync), including `versions.env`.
2. `bash scripts/detect.sh` — stop if exit code 2; fix the reported problem with the human.
3. Ensure `.env` exists (`cp -n .env.example .env`, `chmod 600 .env`). Set `DOMAIN`, `ROOT_URL`, `ACAHTI_ORG`, and any optional vars. Prefer writing values yourself over asking them to edit files.
4. Export the env and run `bash scripts/install.sh` (non-interactive). On failure, diagnose from logs; do not ask the human to “try the README bash block”.
5. Confirm their reverse proxy / tunnel points at `GATEWAY_BIND`. Help them verify `curl -fsS "$ROOT_URL/healthz"` (or equivalent) succeeds over HTTPS.
6. If `ACAHTI_BUILD` was set and the Runner was not installed by `install.sh`: copy `scripts/agent.sh` **and** `pipes/` to the Runner host, then  
   `sudo -E ROLE=both SERVER=<acahti-lan-ip>:9000 SECRET=… bash agent.sh`  
   (`SECRET` = `WOODPECKER_AGENT_SECRET` from the control-plane `.env` — use it on the Runner, do not echo it in chat).
7. **Return to the human** (safe outputs only):
   - `Install $ROOT_URL/skill.md`
   - `Install $ROOT_URL/demo.md`
   - `Join    $ROOT_URL/join` (admin creates invites in the UI — do not invent codes)
   - Admin username (password stays in host `.env`)
   - One-line: paste the demo prompt, open Board, watch

## Upgrade

Pin a reviewed tag or commit. Back up Postgres + `ACAHTI_DATA` first. On the host: refresh the tree (keep `.env`), run `bash scripts/up.sh` (never `compose down -v`). `up.sh` syncs `pipes/` to `ACAHTI_BUILD` when set. The maintainer tag workflow is optional; forks configure their own deploy.

## After install — delivery loop

Point them at the instance home page demo prompt, or:

```text
Install $ROOT_URL/skill.md
Install $ROOT_URL/demo.md
```

Then follow `$ROOT_URL/demo.md` for the shortest agent delivery loop while they watch Board.
