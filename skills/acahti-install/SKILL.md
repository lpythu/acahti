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
- Gateway bind: default `127.0.0.1:8080`. Point the host’s own proxy or tunnel at it. Laptop / no proxy: `GATEWAY_BIND=0.0.0.0:8080`. Do not add Caddy or cloudflared to this compose.

Do not install a Woodpecker agent on the acahti host, office, or thk.

## Steps

1. `ssh` to the host. Copy this `acahti/` tree there (include `versions.env`).
2. `bash scripts/detect.sh` — stop if it exits 2.
3. Export `DOMAIN` `ROOT_URL` `ACAHTI_BUILD` (and `GATEWAY_BIND` if not loopback) and run `bash scripts/install.sh`.
4. Return the printed Use contract (the two `Install` / `Join` lines) and the admin username. Password stays in the host `.env` — do not paste it into chat unless the user asks. Do not put the invite code in skill.md.
5. If the runner was not installed by `install.sh`: `scp scripts/agent.sh` to buildof, then `sudo -E ROLE=both SERVER=<acahti-ip>:9000 SECRET=… bash agent.sh`.

Preview the SPA (no deploy) from the workstation workspace (`./dev-acahti` → `http://127.0.0.1:5173`, `/ui` → host `:8080`). Confirm there, then upgrade: bump `ACAHTI_VERSION`, push `main`, `bash scripts/tag-release.sh`. The GitHub Actions runner **on the acahti host** (`labels: acahti`) pulls that tag as a tarball (GitHub API; `git` to github.com is blocked on this LAN), rsyncs into `/home/saidc/acahti` (keeps `.env`), and runs `scripts/up.sh` (compose + configure; never `compose down -v`). Woodpecker `ROLE=both` stays on buildof; re-run `agent.sh` there only if `WOODPECKER_VERSION` changed.

Never publish Woodpecker gRPC to the internet. Bind is `WOODPECKER_GRPC_PUBLISH` (loopback, or LAN IP when `ACAHTI_BUILD` is set).
