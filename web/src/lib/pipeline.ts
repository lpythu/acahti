import { splitRepo, type Pipeline, type Step } from "@/lib/api"
import type { MessageKey } from "@/i18n/messages"

export type Stage = { name: string; state: string }

export type Job = {
  name: string
  state: string
  steps: Step[]
}

function jobRank(name: string) {
  const n = name.toLowerCase()
  if (/^(ci|build|test|lint|check)/.test(n)) return 0
  if (/^(cd|deploy|release)/.test(n)) return 2
  return 1
}

function failedStatus(state?: string) {
  return /^(failure|error|failed|killed|declined)$/i.test(state || "")
}

function pendingStatus(state?: string) {
  return /^(running|pending|blocked)$/i.test(state || "")
}

function jobFromRuntime(p: Pipeline, j: NonNullable<Pipeline["jobs"]>[number]): Job {
  return {
    name: j.name,
    state: j.state || p.status,
    steps: j.children?.length ? j.children : [{ pid: j.pid || 0, name: j.name, state: j.state || p.status, error: p.error }],
  }
}

function placeholderJob(name: string, state: string, error?: string): Job {
  return { name, state, steps: [{ pid: 0, name, state, error }] }
}

function usableName(name?: string) {
  const n = (name || "").trim()
  if (!n || n === "." || n === "pipeline") return ""
  return n
}

function executedJob(j: Job) {
  return j.steps.some((s) => s.type === "clone" || s.type === "commands")
}

function yamlScalars(raw: string) {
  const bracket = raw.match(/\[([^\]]*)\]/)
  const body = bracket ? bracket[1] : raw
  return body
    .split(",")
    .map((s) => s.trim().replace(/^['"]|['"]$/g, ""))
    .filter(Boolean)
}

export function parseWhen(content: string): { events: string[]; branches: string[] }[] {
  const lines = content.split(/\r?\n/)
  let start = -1
  for (let i = 0; i < lines.length; i++) {
    if (/^when:\s*$/.test(lines[i]) || /^when:\s+\S/.test(lines[i])) {
      start = i
      break
    }
  }
  if (start < 0) return [{ events: [], branches: [] }]
  const block: string[] = []
  for (let i = start + 1; i < lines.length; i++) {
    if (/^\S/.test(lines[i])) break
    block.push(lines[i])
  }
  const rules: { events: string[]; branches: string[] }[] = []
  let cur: { events: string[]; branches: string[] } | null = null
  for (const line of block) {
    if (/^\s*-\s+/.test(line) || (/^\s*-\s*$/.test(line) && cur)) {
      cur = { events: [], branches: [] }
      rules.push(cur)
    }
    if (!cur) {
      cur = { events: [], branches: [] }
      rules.push(cur)
    }
    const ev = line.match(/^\s*-?\s*event:\s*(.*)$/)
    if (ev) cur.events = yamlScalars(ev[1])
    const br = line.match(/^\s*-?\s*branch:\s*(.*)$/)
    if (br) cur.branches = yamlScalars(br[1])
  }
  return rules.length ? rules : [{ events: [], branches: [] }]
}

export function matchesWhen(content: string, event?: string, branch?: string) {
  const ev = (event || "").toLowerCase()
  const br = (branch || "").replace(/^refs\/(heads|tags)\//, "")
  return parseWhen(content).some((rule) => {
    const events = rule.events.map((e) => e.toLowerCase())
    const branches = rule.branches
    const eventOk = !events.length || events.includes(ev) || (ev === "release" && events.includes("tag"))
    const branchOk = !branches.length || branches.includes(br)
    return eventOk && branchOk
  })
}

export function declaredJobNames(files?: { name?: string; path?: string; content?: string }[], event?: string, branch?: string): string[] {
  const names: string[] = []
  const seen = new Set<string>()
  for (const f of files || []) {
    const base = usableName((f.name || f.path?.split("/").pop() || "").replace(/\.ya?ml$/i, ""))
    if (!base || seen.has(base)) continue
    if (f.content != null && !matchesWhen(f.content, event, branch)) continue
    seen.add(base)
    names.push(base)
  }
  return names
}

export function jobsOf(p: Pipeline, flat?: Step[], declared?: string[]): Job[] {
  const runtime: Job[] = []
  if (p.jobs?.length) {
    for (const j of p.jobs) {
      const name = usableName(j.name)
      if (!name) continue
      runtime.push({ ...jobFromRuntime(p, j), name })
    }
  }
  const ran = runtime.filter(executedJob)
  const names = (declared || []).map(usableName).filter(Boolean)
  const source = ran.length ? ran : []
  if (!names.length) {
    return source.sort((a, b) => jobRank(a.name) - jobRank(b.name) || a.name.localeCompare(b.name))
  }
  const byName = new Map(source.map((j) => [j.name, j]))
  const skip = pendingStatus(p.status) ? "pending" : "skipped"
  const steps = (flat || p.steps || []).filter((s) => usableName(s.name))
  const failName =
    source.length === 0 && failedStatus(p.status)
      ? names.slice().sort((a, b) => jobRank(a) - jobRank(b) || a.localeCompare(b))[0]
      : ""
  const out: Job[] = []
  for (const name of names) {
    const hit = byName.get(name)
    if (hit) {
      out.push(hit)
      byName.delete(name)
      continue
    }
    if (name === failName) {
      out.push({
        name,
        state: p.status,
        steps: steps.length ? steps.map((s) => ({ ...s, name: s.name || name })) : [{ pid: 0, name, state: p.status, error: p.error }],
      })
      continue
    }
    out.push(placeholderJob(name, skip))
  }
  return out.sort((a, b) => jobRank(a.name) - jobRank(b.name) || a.name.localeCompare(b.name))
}

export function jobDotsOf(p: Pipeline, flat?: Step[], declared?: string[]): Stage[] {
  return jobsOf(p, flat || p.steps, declared).map((j) => ({ name: j.name, state: j.state }))
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
  const { name } = splitRepo(p.repo)
  const ref = shortRef(p.ref) || p.branch || p.title || ""
  return {
    author: p.author || "—",
    repo: name,
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
