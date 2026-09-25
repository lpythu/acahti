# Compare

Acahti is not competing as “lighter GitHub”. It competes as **agent-native delivery**.

| | GitHub / GitLab | Gitea / Forgejo | Acahti |
|---|---|---|---|
| Primary user | Human in UI | Human in UI | **Agent via MCP**; human watches Board |
| Product center | Forge + social + market | Lightweight forge | **Git + CI + packages + one identity + green merge** |
| Agent access | APIs / official MCP bolted on | Minimal | **skill.md, OAuth MCP, checks_wait, inbox** |
| Identity | Separate tokens per surface | Git-focused | **Same identity for Git, npm, PyPI** |
| Merge policy | Configurable checks | Configurable | **Control-plane merge requires green latest pipeline** |
| Deploy model | Cloud default / heavy self-host | Very light Git | Self-hosted control plane; optional Cloud |

## When to use what

- Stay on **GitHub** when the network effect (OSS contributors, Actions ecosystem, apps) is the product.
- Use **Gitea / Forgejo** when you only need a light human forge.
- Use **Acahti** when coding agents must complete a delivery loop on infrastructure you own, and humans should mostly supervise.
