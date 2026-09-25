# Security model (product view)

## Surfaces

- **Public:** SPA, MCP (`/mcp`), git HTTPS, package APIs, `skill.md` / `demo.md`
- **Private:** Forgejo UI, Woodpecker UI, CI gRPC — not exposed to the internet

## Agent auth

- Agents complete **individual OAuth**. Do not paste access tokens into chat.
- Tokens represent member authority. Per-task delegation is not in this release.
- Git HTTPS uses the Acahti login or the MCP access token via the system credential helper.

## Secrets

- Pipeline secrets are control-plane only. Values are never listed or returned.
- Agents with admin ACL may `secret_put` / `secret_list` (names only). Non-admins do not.
- External cloud credentials (registry, kubeconfig) stay as named secrets — never in git or YAML values.

## Execution

- Control plane and **Runner** are separate hosts. Untrusted jobs should not hold control-plane credentials.
- Official pipes expand on the gateway; the Runner executes `acahti-pipe` with injected env.

## Merge gate

- `pr_merge` succeeds only when the newest pipeline round on the head SHA is green.
- Branch protection and repository ACL remain explicit human policy.

## Honest limits

- A green pipeline means configured checks passed — not that the software is defect-free.
- Humans still approve gated deploys (`deploy_approve`) and manage org membership.
