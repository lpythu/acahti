---
name: acahti-demo
description: Run the shortest Acahti agent delivery loop. Use when the user asks for an Acahti demo, smoke test, or first closed loop on an instance. Install from $ROOT_URL/demo.md.
---

# Acahti demo loop

This file is served by the instance. If cached, GET `$ROOT_URL/demo.md` now and follow that.

```text
Install $ROOT_URL/skill.md
Install $ROOT_URL/demo.md
```

Goal: in one session, prove **agent-driven delivery** — you do the work; the human only watches Board / Inbox.

## Constraints

- Connect MCP at `$ROOT_URL/mcp` with OAuth. Do not paste tokens into chat.
- Do not `git config --global`. Follow `skill.md` identity rules for Acahti remotes only.
- Prefer a disposable repo name `demo-<yyyymmdd>` under `$ACAHTI_ORG`.
- Keep the change trivial (README line or `hello.txt`). Do not deploy, do not touch org secrets.

## Loop (do in order)

1. `whoami` — confirm instance URL and git identity.
2. `repo_list` — if `demo-*` already exists and you can push, reuse it; else `repo_create` `{name: "demo-<yyyymmdd>"}`.
3. Clone `$ROOT_URL/$ACAHTI_ORG/<repo>.git` (HTTPS). Apply `setup_local` from `whoami` before committing.
4. Create branch `demo/loop`. Add `.acahti/pipelines/ci.yml`:

```yaml
steps:
  smoke:
    commands:
      - echo "acahti-demo ok"
```

5. Commit and push the branch. Open a PR with `pr_create` (title `demo: agent delivery loop`).
6. `checks_wait` on the head SHA — poll until `done` is true (do not pass `timeout_sec`).
   - Failed: `pipeline_log` → fix → push → wait again.
   - No runner / stuck queued: stop, tell the human to register a Runner (`ACAHTI_BUILD`), paste `pipeline_get` wait state. Do not invent a green result.
7. When latest pipeline on the head is green: `pr_merge`.
8. Reply with: repo URL, PR number, pipeline number, and one sentence that the human watched Board while the agent ran the loop.

## Done means

PR merged (or explicitly blocked on missing Runner with evidence). Do not claim success without MCP results.
