import { api, splitRepo, type FileBlob, type Pipeline, type Step } from "@/lib/api"
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

export function declaredJobNames(files?: { name?: string; path?: string }[]): string[] {
  const names: string[] = []
  const seen = new Set<string>()
  for (const f of files || []) {
    const base = (f.name || f.path?.split("/").pop() || "").replace(/\.ya?ml$/i, "")
    if (!base || base === "pipeline" || seen.has(base)) continue
    seen.add(base)
    names.push(base)
  }
  return names
}

export async function loadDeclaredJobNames(owner: string, name: string, ref: string): Promise<string[]> {
  if (!ref) return []
  try {
    const dir = await api.contents(owner, name, { ref, path: ".acahti/pipelines" })
    return declaredJobNames((dir.items || []).filter((e) => e.type === "file" || e.type === "blob"))
  } catch {
    return []
  }
}

export function jobsOf(p: Pipeline, flat?: Step[], declared?: string[]): Job[] {
  const runtime: Job[] = []
  if (p.jobs?.length) {
    for (const j of p.jobs) {
      if (!j.name || j.name === "pipeline") continue
      runtime.push(jobFromRuntime(p, j))
    }
  }
  const names = (declared || []).filter((n) => n && n !== "pipeline")
  if (!names.length) {
    return runtime.sort((a, b) => jobRank(a.name) - jobRank(b.name) || a.name.localeCompare(b.name))
  }
  const byName = new Map(runtime.map((j) => [j.name, j]))
  const skip = pendingStatus(p.status) ? "pending" : "skipped"
  const steps = flat?.length ? flat : p.steps || []
  const failName =
    runtime.length === 0 && failedStatus(p.status)
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
        steps: steps.length
          ? steps.map((s) => ({ ...s, name: s.name || name }))
          : [{ pid: 0, name, state: p.status, error: p.error }],
      })
      continue
    }
    out.push(placeholderJob(name, skip))
  }
  for (const j of byName.values()) out.push(j)
  return out.sort((a, b) => jobRank(a.name) - jobRank(b.name) || a.name.localeCompare(b.name))
}

export function jobDotsOf(p: Pipeline, flat?: Step[], declared?: string[]): Stage[] {
  return jobsOf(p, flat || p.steps, declared).map((j) => ({ name: j.name, state: j.state }))
}

export async function loadPipelineFiles(owner: string, name: string, ref: string): Promise<FileBlob[]> {
  if (!ref) return []
  const out: FileBlob[] = []
  const seen = new Set<string>()

  async function add(path: string) {
    if (!path || seen.has(path)) return
    seen.add(path)
    try {
      const ov = await api.contents(owner, name, { ref, path })
      if (ov.file) out.push(ov.file)
    } catch {
      /* missing */
    }
  }

  let dirEntries: { path?: string; name: string; type: string }[] = []
  try {
    const dir = await api.contents(owner, name, { ref, path: ".acahti/pipelines" })
    dirEntries = dir.items || []
  } catch {
    dirEntries = []
  }

  const paths = dirEntries
    .filter((e) => e.type === "file" || e.type === "blob")
    .filter((e) => /\.ya?ml$/i.test(e.name))
    .map((e) => e.path || `.acahti/pipelines/${e.name}`)
  await Promise.all(paths.map(add))
  return out.sort((a, b) => a.path.localeCompare(b.path))
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
