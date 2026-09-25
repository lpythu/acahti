# Demo loop

Shortest path for a human + agent to feel Acahti.

## Human (30 seconds)

1. Install Acahti ([installation](../installation.md)) or open your instance.
2. Open the instance home page (logged out).
3. Click **Copy demo prompt for agent**.
4. Open **Board** in another tab after joining / logging in.
5. Paste the prompt into Cursor / Codex / your MCP agent.

## Agent (one paste)

The prompt installs:

```text
Install https://<your-host>/skill.md
Install https://<your-host>/demo.md
```

Then runs: `whoami` → disposable `demo-*` repo → smoke pipeline → PR → `checks_wait` → `pr_merge` (or stop with evidence if no Runner).

## Instance endpoints

| Path | Purpose |
|---|---|
| `/skill.md` | Everyday agent contract |
| `/demo.md` | Shortest closed-loop skill |
| `/ui/public` | JSON including `agent_prompt` for UIs |
| `/mcp` | Streamable HTTP MCP + OAuth |

## Success criteria

- PR merged with a green pipeline, **or**
- Explicit stop: no Runner / queue stuck, with `pipeline_get` evidence

Do not claim success without MCP results. The human should only watch Board / Inbox.
