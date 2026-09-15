---
name: acahti
description: Use Acahti git, PRs, pipelines, and commit checks. Use when committing or pushing to an acahti remote, opening or merging a PR, reading pipelines, or when the user mentions acahti.saidc.ai, whoami, or island CI. Install from $ROOT_URL/skill.md. Connect MCP at $ROOT_URL/mcp (OAuth).
---

# Use Acahti

This file is served by acahti. Do not copy it from GitHub. If you are reading a cached copy, GET `$ROOT_URL/skill.md` now and follow that. Discover MCP tools this session; do not trust a cached tool list.

```text
Install $ROOT_URL/skill.md
Join    $ROOT_URL/join     (invite from an admin)
```

## MCP

Connect Streamable HTTP MCP at `$ROOT_URL/mcp`. Use OAuth when the client challenges. Do not paste a Bearer token. Do not `git config --global`.

If the browser has no account, open `$ROOT_URL/join` with the admin invite, pick a username and password. If you already have an account, log in at `$ROOT_URL/login`.

## Git

HTTPS only. Clone `$ROOT_URL/$ACAHTI_ORG/<repo>.git`. Username `whoami.login`. Password is the OAuth access_token the MCP client already holds — use the system git credential helper. On 401, tell the user to complete MCP OAuth. Do not read `secrets/` or ask them to paste a token.

Clone is always `$ACAHTI_ORG/<repo>`. A team (Platform, ModelCamp, …) is access only; do not clone `modelcamp/<repo>`.

`ssh://`, `:2222`, and bare LAN IPs are not island git. `git remote set-url` or `add` named `acahti` to `$ROOT_URL/$ACAHTI_ORG/<repo>.git`.

If Codeup (or another host) is `origin`, keep it. Fetch and push the island on remote `acahti`. Do not `git fetch --all` to sync the island.

This is not about the `acahti/` product repo on GitHub. That remote stays GitHub.

Before any `git commit` in the current repo:

1. `git remote -v`
2. Call MCP `whoami`
3. If **any** remote URL host is in `apply_when_remote_host`, run `setup_local` (`git config --local` only)
4. Keep the laptop identity only when **no** remote is an acahti host

`git_name` is the commit author the admin set on the user (default `login`). `git_email` is `{login}@noreply.$DOMAIN`. Run `setup_local` with `whoami.git_name` and `whoami.git_email`. Do not ask the user for a name or email.

`dev` and `test` are protected. If push is declined, open a PR. Do not push `main` / `release`; open a PR.

## After push

Trigger CI with `git push`, a tag, or opening a PR. Do not use `pipeline_trigger` as a substitute for official release.

1. `git rev-parse HEAD`
2. `checks_wait` `{owner, name, sha}` — snapshot of the latest pipeline round. Poll until `done` is true. Do not pass `timeout_sec`
3. Failed: `pipeline_list` `{repo: owner/name, sha}` (sha prefix) → `pipeline_get` → `pipeline_log` (omit `step`)
4. Fix and push, or `pipeline_rerun`. `pipeline_cancel` only for a stuck run
5. Green (`ok` and `done`): `pr_merge`. `blocked`: `deploy_approve`
6. Island triage: `inbox` `{section: pipes|prs}` (default pipes: latest blocked/failed per repo)

`pr_merge` succeeds when the newest pipeline number on the head SHA is green. Close leftover heads with `pr_close` then `ref_delete` (`dev`, `heads/dev`, or `refs/heads/dev`).

## Packages

- PyPI: `$ROOT_URL/api/packages/$ACAHTI_ORG/pypi/simple/`
- npm: `$ROOT_URL/api/packages/$ACAHTI_ORG/npm/`
- REST: `$ROOT_URL/acahti/v1/…` same verbs as MCP

## Tools

`whoami` `repo_list` `repo_get` `repo_create` `branch_list` `ref_delete` `pr_create` `pr_list` `pr_get` `pr_comment` `pr_comments` `pr_merge` `pr_close` `checks_wait` `pipeline_list` `pipeline_get` `pipeline_log` `pipeline_rerun` `pipeline_trigger` `pipeline_cancel` `pipeline_delete` `inbox` `pkg_list` `pkg_publish` `agent_status` `deploy_approve`
