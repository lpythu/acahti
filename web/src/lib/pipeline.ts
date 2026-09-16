import { repoName, type Pipeline, type Step } from "@/lib/api"
import type { MessageKey } from "@/i18n/messages"

export type Job = {
  name: string
  state: string
  steps: Step[]
}

function usableName(name?: string) {
  const n = (name || "").trim()
  if (!n || n === "." || n === "pipeline") return ""
  return n
}

export function jobsOf(p: Pipeline): Job[] {
  return (p.jobs || [])
    .map((j) => {
      const name = usableName(j.name)
      if (!name) return null
      return {
        name,
        state: j.state || p.status,
        steps: j.steps?.length ? j.steps : [],
      }
    })
    .filter((j): j is Job => j != null)
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

export function runLane(p: Pipeline) {
  return `${p.repo}\0${(p.ref || p.branch || "").trim()}`
}

export function upsertHead<T extends { items?: Pipeline[] }>(page: T | null, next: Pipeline, pageNo: number): T | null {
  if (!page) return page
  const items = [...(page.items || [])]
  const i = items.findIndex((p) => runLane(p) === runLane(next))
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

export function triggerKey(event?: string): MessageKey {
  switch ((event || "").toLowerCase()) {
    case "push":
      return "triggerPush"
    case "tag":
    case "release":
      return "triggerTag"
    case "pull_request":
    case "pull_request_closed":
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

export function runEventKey(event?: string): MessageKey {
  switch ((event || "").toLowerCase()) {
    case "push":
      return "runPush"
    case "tag":
    case "release":
      return "runTag"
    case "pull_request":
    case "pull_request_closed":
      return "runPR"
    case "cron":
      return "runCron"
    default:
      return "runManual"
  }
}
