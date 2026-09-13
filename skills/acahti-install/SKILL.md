---
name: acahti-install
description: Install or upgrade Acahti on Linux over SSH. Use when the user asks to bootstrap Acahti, stand up the control plane, add the buildof host runner, or upgrade the pinned train.
---

# Install Acahti

Follow [../../README.md](../../README.md) as the contract. Do not invent a second installer. Versions are [../../versions.env](../../versions.env).

## Inputs (ask once if missing)

- `DOMAIN`, `ROOT_URL`
- SSH to the **acahti** host (sudoer, not root)
- Optional `ACAHTI_ORG` (default `acme`)
- `ACAHTI_BUILD` — SSH spec for **buildof** (one agent, `ROLE=both`)
- Edge: public IP → `EDGE=caddy`; no inbound + tunnel token → `EDGE=cloudflared`; else `EDGE=none`

Do not install a Woodpecker agent on the acahti host, office, or thk.

## Steps

1. `ssh` to the host. Copy this `acahti/` tree there (include `versions.env`).
2. `bash scripts/detect.sh` — stop if it exits 2.
3. Export `DOMAIN` `ROOT_URL` `EDGE` `ACAHTI_TLS` `ACAHTI_BUILD` and run `bash scripts/install.sh`.
4. Return the printed URL, MCP path, invite path, and admin username. Password stays in the host `.env` — do not paste it into chat unless the user asks.
5. If the runner was not installed by `install.sh`: `scp scripts/agent.sh` to buildof, then `sudo -E ROLE=both SERVER=<acahti-ip>:9000 SECRET=… bash agent.sh`.

Upgrade the island: bump `ACAHTI_VERSION`, push `main`, `bash scripts/tag-release.sh`. The host self-hosted runner checks out that tag and runs `scripts/up.sh` (never `compose down -v`). Then re-run `agent.sh` on buildof if `WOODPECKER_VERSION` changed.

Never publish Woodpecker gRPC `:9000` to the internet.
