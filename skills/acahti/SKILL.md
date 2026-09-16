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

HTTPS only. Clone `$ROOT_URL/$ACAHTI_ORG/<repo>.git`. Username is the Acahti login. Password is either the account password (same as `/login`) or the OAuth `access_token` the MCP client already holds — agents use the system git credential helper with the token. On 401 for a human, check the login password; for an agent, complete MCP OAuth. Do not read `secrets/` or ask them to paste a token.

Clone is always `$ACAHTI_ORG/<repo>`. A team (Platform, ModelCamp, …) is access only; do not clone `modelcamp/<repo>`.

`ssh://`, `:2222`, and bare LAN IPs are not island git. `git remote set-url` or `add` named `acahti` to `$ROOT_URL/$ACAHTI_ORG/<repo>.git`.

If Codeup (or another host) is `origin`, keep it. Fetch and push the island on remote `acahti`. Do not `git fetch --all` to sync the island.

Before any `git commit` in the current repo:

1. `git remote -v`
2. Call MCP `whoami`
3. Gate **only** on remote URL host — never on directory or repo name
4. If **any** remote URL host is in `apply_when_remote_host`, run `setup_local` (`git config --local` only)
5. If **no** remote host matches, do **not** change `user.name` / `user.email`. If local author was wrongly set to `*@noreply.$DOMAIN`, `git config --local --unset user.name` and `user.email` so the laptop identity applies

Use `whoami.git_name` / `whoami.git_email` as the git author. Do not ask the user for a name or email.

Protection is per repository, not a global train. Before push: `repo_get` `{owner, name}` and `branch_list` `{owner, name}`. Direct-push branches with `protected: false`. For `protected: true`, open a PR (`pr_create`; `base` defaults to `repo_get.default_branch`). Do not assume `dev` / `test` / `main` / `release`. If the remote declines a push, open a PR to the default branch. Operators set default / protection on the repo **Branches** page in the web UI.

## After push

## Write a pipeline

Put YAML in `.acahti/pipelines/`. Call official pipes with `pipe: <name>@v1` and `with:`. One-offs use `commands:`. Do not vendor `.acahti/scripts`. Do not write `uses:` or `KIND=`.

Git tags are `vX.Y.Z`. OCI / helm `image.tag` never includes `v` (office `dev-{sha}`; HK `${CI_COMMIT_TAG#v}` → `X.Y.Z`; `latest` is a pointer). Do not write `image.tag=${CI_COMMIT_TAG}`.

Trigger CI with `git push`, a tag, or opening a PR. Do not use `pipeline_trigger` as a substitute for official release.

1. `git rev-parse HEAD`
2. `checks_wait` `{owner, name, sha}` — snapshot of the latest pipeline round. Poll until `done` is true. Do not pass `timeout_sec`
3. Failed: `pipeline_list` `{repo: owner/name, sha}` (sha prefix) → `pipeline_get` (jobs, steps, and wait) → `pipeline_log` (omit `step`)
4. Fix and push, or `pipeline_rerun`. `pipeline_cancel` only for a stuck run
5. Green (`ok` and `done`): `pr_merge`. `blocked`: `deploy_approve`
6. Island triage: `inbox` `{section: pipes|prs}` (default pipes: latest blocked/failed per repo)

`pr_merge` succeeds when the newest pipeline number on the head SHA is green. Close leftover heads with `pr_close` then `ref_delete` (`dev`, `heads/dev`, or `refs/heads/dev`).

## Pipeline secrets

Island-external only (Harbor/ACR, kubeconfig, OSS, argos). YAML names them (`KUBECONFIG: kubeconfig_office`, `DOCKER_PASSWORD: harbor_password`). Do not put values in git, chat, or `with:`.

Acahti npm/pypi use the job's Acahti identity (the user who triggered the run). Do not write `NPM_TOKEN` or a publish token.

In YAML `commands:`, use `$VAR` for step env. `${VAR}` is interpolated before the shell runs and is empty unless it is job context (`${CI_COMMIT_SHA}`).

`repo_get.can_manage_secrets` or `whoami.org_admin` → `secret_list` (repo scope is the effective set) → `secret_put`. Never print `value`. There is no `secret_get`. Do not slurp workspace `secrets/` unless the operator names a file.

Non-admin agent: do not call `secret_*`. If a pipe fails missing a secret, tell the operator Repo → Secrets (or Admin → Org secrets). Do not ask them to paste a token into the chat.

Laptop `secrets/` is operator tooling (Cloudflare, dash.env download), not Runner or pipeline state.

## Packages

Same credentials as git. Username is the Acahti login. Password is the login password or MCP `access_token`.

- npm: `$ROOT_URL/api/packages/$ACAHTI_ORG/npm/`
- PyPI: `$ROOT_URL/api/packages/$ACAHTI_ORG/pypi/simple/`
- REST: `$ROOT_URL/acahti/v1/…` same verbs as MCP

Repo `.npmrc` / `[[tool.uv.index]]` name the registry only (scopes like `@saidc` are the app’s file, not the pipe). Laptop auth is `~/.npmrc` (`_password` = **base64** of the password or MCP token) or `UV_INDEX_<INDEX>_USERNAME` / `UV_INDEX_<INDEX>_PASSWORD` / `~/.netrc`. CI: `docker-build` forwards job identity as BuildKit secrets `acahti_user` / `acahti_token`; Dockerfiles mount them on install. YAML does not name npm tokens. YAML names Harbor/ACR as `BASE_IMAGE=…`.

Org members who can see repos can install. Publish uses the triggering user's identity (`npm-publish` / `pypi-publish` / `pkg_publish`). Those pipes install, build, and upload in a container; YAML does not `docker run` or write `.npmrc` / `.netrc`. Org admins remove a version with `pkg_delete`.

## Tools

`whoami` `repo_list` `repo_get` `repo_create` `branch_list` `ref_delete` `pr_create` `pr_list` `pr_get` `pr_comment` `pr_comments` `pr_merge` `pr_close` `checks_wait` `pipeline_list` `pipeline_get` `pipeline_log` `pipeline_rerun` `pipeline_trigger` `pipeline_cancel` `pipeline_delete` `inbox` `pkg_list` `pkg_publish` `pkg_delete` `agent_status` `deploy_approve` `secret_list` `secret_put` `secret_delete`
