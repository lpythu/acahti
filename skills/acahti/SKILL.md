---
name: acahti
description: Use an existing Acahti island (git, PRs, required checks, packages) via MCP after the user has ROOT_URL and a token.
---

# Use Acahti

Fill these from the admin (not from this file):

- `ROOT_URL` — e.g. `https://acahti.example.com`
- `ACAHTI_ORG` — e.g. `acme`
- Bearer token from the board **MCP token** page

## Endpoints

- MCP: `$ROOT_URL/mcp` (Streamable HTTP, Authorization: Bearer)
- Git: `https://$ROOT_URL/$ACAHTI_ORG/<repo>.git` (token as password)
- PyPI: `$ROOT_URL/api/packages/$ACAHTI_ORG/pypi/simple/`
- npm: `$ROOT_URL/api/packages/$ACAHTI_ORG/npm/`
- REST: `$ROOT_URL/acahti/v1/…`

## Tools

`repo_list` `repo_get` `repo_create` `branch_list` `ref_delete` `pr_create` `pr_list` `pr_get` `pr_comment` `pr_merge` `checks_wait` `pipeline_log` `pipeline_rerun` `pkg_list` `pkg_publish` `agent_status` `deploy_approve`

`pr_merge` only succeeds when commit checks are green. Do not push `main` / `release`; open a PR.

Large npm PUTs that hang on an HTTP edge: run `scripts/npm-put.py` on the acahti host against origin HTTP.
