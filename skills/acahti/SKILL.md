---
name: acahti
description: Use Acahti. Install this file from $ROOT_URL/skill.md. Connect MCP at $ROOT_URL/mcp (OAuth). Before git commit, whoami and set local identity when the remote host is acahti.
---

# Use Acahti

This file is served by acahti. Do not copy it from GitHub.

```text
Install $ROOT_URL/skill.md
Join    $ROOT_URL/join     (invite from an admin)
```

## MCP

Connect Streamable HTTP MCP at `$ROOT_URL/mcp`. Use OAuth when the client challenges. Do not paste a Bearer token. Do not `git config --global`.

If the browser has no account, open `$ROOT_URL/join` with the admin invite, pick a username and password. If you already have an account, log in at `$ROOT_URL/login`.

## Git

- HTTPS: `$ROOT_URL/$ACAHTI_ORG/<repo>.git`
- Username: `whoami.login`
- Password: the OAuth access_token the MCP client already holds
- SSH optional: `ssh://git@$DOMAIN:2222/$ACAHTI_ORG/<repo>.git`

This is not about the `acahti/` product repo on GitHub. That remote stays GitHub.

Before any `git commit` in the current repo:

1. `git remote -v`
2. Call MCP `whoami`
3. If a remote URL host is in `apply_when_remote_host`, run `setup_local` (`git config --local` only)
4. If remotes are `github.com`, `codeup.aliyun.com`, or any other host, leave identity unchanged

`git_email` is `{login}@noreply.$DOMAIN`. Do not ask the user for an email.

## Packages

- PyPI: `$ROOT_URL/api/packages/$ACAHTI_ORG/pypi/simple/`
- npm: `$ROOT_URL/api/packages/$ACAHTI_ORG/npm/`
- REST: `$ROOT_URL/acahti/v1/…` same verbs as MCP

## Tools

`whoami` `repo_list` `repo_get` `repo_create` `branch_list` `ref_delete` `pr_create` `pr_list` `pr_get` `pr_comment` `pr_merge` `checks_wait` `pipeline_log` `pipeline_rerun` `pkg_list` `pkg_publish` `agent_status` `deploy_approve`

`pr_merge` only succeeds when commit checks are green. Do not push `main` / `release`; open a PR.
