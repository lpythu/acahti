# Positioning

**Acahti is an agent delivery control plane** — not “another Git host”.

## One line

**Agents ship. You watch.**

Self-hosted Git, CI checks, language packages, and MCP identity in one loop from push to checked merge. Bring your coding agent. Humans supervise on the Board.

## Who does what

| Actor | Role |
|---|---|
| Coding agent | Connects with OAuth, pushes, waits on checks, reads failed logs, opens/merges PRs, publishes packages |
| Human | Invites, branch protection, secrets, ACL, Board/Inbox, gated `deploy_approve` |

Agents drive the **delivery loop**. Humans own **policy and exceptions**. Acahti does not host a model and does not autonomously fix code.

## Category

| Term | Meaning |
|---|---|
| Control plane | Public gateway for SPA, MCP, git HTTPS, packages |
| Kernels | Forgejo (Git/PRs/packages) + Woodpecker (CI on Runner) behind the gateway |
| Skill | Instance-served `skill.md` / `demo.md` for agents |
| Checked handoff | `pr_merge` requires the latest pipeline on the PR head to be green |

## Non-goals

- Full GitHub API / social / marketplace parity
- Automatic migration of issues, Actions, and third-party apps
- Replacing Cursor / Codex / other agents
- Claiming agents can perform every admin action
