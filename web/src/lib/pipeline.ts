import { repoName, type Pipeline, type Step } from "@/lib/api"
import type { MessageKey } from "@/i18n/messages"

export type Job = {
  name: string
  state: string
  wait?: string
  queue_position?: number
  agent?: string
  steps: Step[]
}

function usableName(name?: string) {
  const n = (name || "").trim()
  if (!n || n === "." || n === "pipeline") return ""
  return n
}

export function jobsOf(p: Pipeline): Job[] {
  const out: Job[] = []
  for (const j of p.jobs || []) {
    const name = usableName(j.name)
    if (!name) continue
    out.push({
      name,
      state: j.state,
      wait: j.wait,
      queue_position: j.queue_position,
      agent: j.agent,
      steps: j.steps?.length ? j.steps : [],
    })
  }
  return out
}

export function jobDotsOf(p: Pipeline): { name: string; state: string }[] {
  return jobsOf(p).map((j) => ({ name: j.name, state: j.state }))
}

export function asPipeline(data: unknown): Pipeline | null {
  if (!data || typeof data !== "object") return null
  const p = data as Pipeline
  if (!p.repo || !p.number) return null
  return p
}

const FAILED = new Set(["failure", "error", "killed", "declined"])

export function failedStatus(status?: string) {
  return FAILED.has((status || "").toLowerCase())
}

export function pipeFilterMatch(status: string, filter: string) {
  const s = (status || "").toLowerCase()
  switch (filter) {
    case "failed":
      return FAILED.has(s)
    case "blocked":
      return s === "blocked"
    case "running":
      return s === "running" || s === "pending"
    case "success":
      return s === "success"
    default:
      return true
  }
}

export function upsertRun<T extends { items?: Pipeline[] }>(page: T | null, next: Pipeline, pageNo: number): T | null {
  if (!page) return page
  const items = [...(page.items || [])]
  const i = items.findIndex((p) => p.repo === next.repo && p.number === next.number)
  if (i >= 0) {
    items[i] = next
    return { ...page, items }
  }
  if (pageNo <= 1) {
    items.unshift(next)
    return { ...page, items }
  }
  return page
}

export function upsertHead<T extends { items?: Pipeline[] }>(page: T | null, next: Pipeline, pageNo: number): T | null {
  if (!page) return page
  const items = [...(page.items || [])]
  const i = items.findIndex((p) => p.repo === next.repo)
  if (i >= 0) {
    if (next.number < items[i].number) return { ...page, items }
    if (next.number === items[i].number) {
      items[i] = next
      return { ...page, items }
    }
    items.splice(i, 1)
  }
  if (pageNo <= 1) {
    items.unshift(next)
    return { ...page, items }
  }
  return { ...page, items }
}

function shortRef(ref?: string) {
  if (!ref) return ""
  return ref.replace(/^refs\/(heads|tags)\//, "")
}

export type TriggerKind = "push" | "tag" | "pr" | "cron" | "manual"

export function triggerKind(event?: string, ref?: string): TriggerKind {
  const ev = (event || "").toLowerCase()
  if (ev === "tag" || ev === "release" || (ref || "").startsWith("refs/tags/")) return "tag"
  switch (ev) {
    case "push":
      return "push"
    case "pull_request":
    case "pull_request_closed":
    case "pull_request_metadata":
      return "pr"
    case "cron":
      return "cron"
    default:
      return "manual"
  }
}

export function triggerKey(event?: string, ref?: string): MessageKey {
  switch (triggerKind(event, ref)) {
    case "push":
      return "triggerPush"
    case "tag":
      return "triggerTag"
    case "pr":
      return "triggerPR"
    case "cron":
      return "triggerCron"
    default:
      return "triggerManual"
  }
}

export function triggerVars(p: Pipeline): Record<string, string> {
  const ref = shortRef(p.ref) || p.branch || p.title || ""
  return {
    author: p.author || "—",
    repo: repoName(p.repo),
    branch: p.branch || ref,
    ref,
  }
}

export function runTitle(p: Pipeline) {
  const raw = (p.message || p.title || "").trim()
  return raw.split("\n")[0] || p.event || "pipeline"
}

export function runRef(p: Pipeline) {
  return shortRef(p.ref) || p.branch || ""
}

export function runEventKey(event?: string, ref?: string): MessageKey {
  switch (triggerKind(event, ref)) {
    case "push":
      return "runPush"
    case "tag":
      return "runTag"
    case "pr":
      return "runPR"
    case "cron":
      return "runCron"
    default:
      return "runManual"
  }
}

export function inFlight(status?: string) {
  switch ((status || "").toLowerCase()) {
    case "running":
    case "started":
    case "pending":
    case "created":
    case "blocked":
      return true
    default:
      return false
  }
}

export function waitLine(p: { status?: string; wait?: string; queue_position?: number; agent?: string }): {
  key: MessageKey
  vars?: Record<string, string>
} | null {
  if ((p.status || "").toLowerCase() === "blocked") return { key: "statusBlocked" }
  if (!inFlight(p.status)) return null
  if (p.wait === "queue") {
    if (p.queue_position && p.queue_position > 0) return { key: "waitQueuedN", vars: { n: String(p.queue_position) } }
    return { key: "statusQueued" }
  }
  if (p.wait === "deps") return { key: "statusWaiting" }
  if (p.wait === "concurrency") return { key: "statusSlot" }
  return null
}

export function namedSecrets(files: { content?: string }[] | undefined): string[] {
  const names = new Set<string>()
  for (const f of files || []) {
    const text = f.content || ""
    for (const m of text.matchAll(/secrets:\s*\[([^\]]+)\]/g)) {
      for (const part of m[1].split(",")) {
        const n = part.replace(/['"]/g, "").trim()
        if (n) names.add(n)
      }
    }
    for (const m of text.matchAll(/secrets:\s*\n((?:[ \t]+[A-Za-z0-9_]+:[ \t]*[A-Za-z0-9_]+\n?)+)/g)) {
      for (const line of m[1].split("\n")) {
        const kv = line.match(/^[ \t]+[A-Za-z0-9_]+:[ \t]*([A-Za-z0-9_]+)\s*$/)
        if (kv) names.add(kv[1])
      }
    }
  }
  return [...names].sort()
}
