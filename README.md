# Acahti

**One connection from code to checked delivery.**

[![CI](https://github.com/lpythu/acahti/actions/workflows/ci.yml/badge.svg)](https://github.com/lpythu/acahti/actions/workflows/ci.yml)
[![License: Apache-2.0](https://img.shields.io/badge/license-Apache--2.0-blue)](LICENSE)

[Install](docs/installation.md) · [Architecture](architecture.md) · [Pipeline pipes](pipes.md) · [Agent plugins](https://github.com/lpythu/acahti-plugin) · [Stack guide](https://github.com/benchyard/stack)

Acahti is a self-hosted delivery control plane for people and coding agents. It
combines Git repositories, pull requests, commit checks, CI logs and language
packages behind one identity and one MCP endpoint. Bring your coding agent;
Acahti does not host a model or require a particular editor.

![Acahti delivery workflow](docs/assets/delivery.svg)

## Why Acahti

Writing code is only one step. An agent also needs to identify the right repository,
read failed checks, repair a change, and hand over an artifact someone can trust.
Acahti provides a consistent workflow for those operations:

- **One identity:** Git, npm and PyPI share the member's Acahti identity. Agents use
  OAuth; external cloud credentials remain separately managed pipeline secrets.
- **Structured feedback:** query the commit's checks, fetch failed-step logs, rerun
  a pipeline and inspect its queue state through MCP.
- **Checked handoff:** the PR merge operation requires a successful latest pipeline
  on the PR head. Branch protection and repository access remain explicit policy.
- **An actionable inbox:** find failed or blocked runs and PRs ready for review.
- **Your infrastructure:** a Go gateway fronts Forgejo and Woodpecker. Mature Git
  and CI engines do the storage/execution work; Acahti owns the shared workflow.

GitHub's official MCP already supports repository and CI operations. Acahti's focus
is a cohesive, self-hosted installation with integrated package identity and a
small delivery workflow—not exclusive access to agents or universal GitHub API parity.

## Connect an agent

Install Acahti on a host using the [installation guide](docs/installation.md), then
use the URL served by **your** instance:

```text
Install https://acahti.example.com/skill.md
```

For editor integration, use [acahti-plugin](https://github.com/lpythu/acahti-plugin).
Connect the instance's `/mcp` endpoint and complete individual OAuth. Verify with
`whoami`; the result contains the instance URL, Git identity and current skill URL.
Do not paste tokens into chats or change your global Git identity.

A typical agent workflow, within the user's authorized scope:

```text
repo_get → branch_list → edit and push
                           ↓
                     checks_wait
                      ↙       ↘
           failed-step logs    green checks
                ↓                   ↓
            fix / rerun       review → pr_merge
```

The agent decides and performs repairs. Acahti supplies authenticated operations
and results; it does not autonomously fix pipelines by itself. A green pipeline
means its configured checks passed, not that the software has no defects.

## Choose what you need

| Need | Component |
|---|---|
| Git, CI, packages and MCP identity | **Acahti** |
| Cursor / Codex installation guidance | [Acahti Plugin](https://github.com/lpythu/acahti-plugin) |
| Shared tasks and execution UI | [Benchyard Console](https://github.com/benchyard/benchyard-console) |
| Persistent dev environment and preview | [Skheri](https://github.com/benchyard/skheri) |
| Once / soak checks and evidence | [Argos](https://github.com/lpythu/argos) |

These are separate products. Acahti can be used without Benchyard or Skheri.
Acahti is a Git host; adding it as another remote does not automatically migrate
GitHub issues, integrations or CI settings. See the [stack guide](https://github.com/benchyard/stack)
for a staged adoption path.

## Development and contribution

```bash
(cd web && npm ci && npm run build:embed)
GOWORK=off go test ./...
GOWORK=off go test -race ./internal/mcp ./internal/oauth
```

[Contributing](CONTRIBUTING.md) · [Security](SECURITY.md) · [Third-party components](THIRD_PARTY_NOTICES.md)

Acahti gateway and integration code: [Apache-2.0](LICENSE). Forgejo, Woodpecker and
other third-party components retain their own licenses. Current tokens represent
member authority; per-task agent delegation and automatic GitHub migration are
not features of this release.
