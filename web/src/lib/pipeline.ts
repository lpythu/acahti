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

export function jobsOf(p: Pipeline, flat?: Step[]): Job[] {
  if (p.jobs?.length) {
    return p.jobs
      .map((j) => ({
        name: j.name,
        state: j.state || p.status,
        steps: j.children?.length ? j.children : [{ pid: j.pid || 0, name: j.name, state: j.state || p.status }],
      }))
      .sort((a, b) => jobRank(a.name) - jobRank(b.name) || a.name.localeCompare(b.name))
  }
  if (flat?.length) {
    return [{ name: p.event || "pipeline", state: p.status, steps: flat }]
  }
  return []
}

export function jobDotsOf(p: Pipeline, flat?: Step[]): Stage[] {
  return jobsOf(p, flat || p.steps).map((j) => ({ name: j.name, state: j.state }))
}

export async function withJobs(pipes: Pipeline[]): Promise<Pipeline[]> {
  return Promise.all(
    pipes.map(async (p) => {
      if (p.jobs?.length) return p
      const { owner, name } = splitRepo(p.repo)
      if (!owner || !name || !p.number) return p
      try {
        const d = await api.pipeline(owner, name, p.number)
        return { ...p, ...d.pipeline, jobs: d.pipeline.jobs, steps: d.steps }
      } catch {
        return p
      }
    }),
  )
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
