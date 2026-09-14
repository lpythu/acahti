import { api, splitRepo, type FileBlob, type Pipeline, type Step } from "@/lib/api"
import type { MessageKey } from "@/i18n/messages"

export type Stage = { name: string; state: string }

export type Job = {
  name: string
  state: string
  steps: Step[]
}

export function latestByRepo(pipes: Pipeline[]): Pipeline[] {
  const m = new Map<string, Pipeline>()
  for (const p of pipes) {
    const cur = m.get(p.repo)
    if (!cur || p.number > cur.number) m.set(p.repo, p)
  }
  return [...m.values()].sort((a, b) => (b.started || b.created || 0) - (a.started || a.created || 0))
}

function jobRank(name: string) {
  const n = name.toLowerCase()
  if (/^(ci|build|test|lint|check)/.test(n)) return 0
  if (/^(cd|deploy|release)/.test(n)) return 2
  return 1
}

export function jobsOf(p: Pipeline, flat?: Step[]): Job[] {
  if (p.workflows?.length) {
    return p.workflows
      .map((w) => ({
        name: w.name,
        state: w.state || p.status,
        steps: w.children?.length ? w.children : [{ pid: w.pid || 0, name: w.name, state: w.state || p.status }],
      }))
      .sort((a, b) => jobRank(a.name) - jobRank(b.name) || a.name.localeCompare(b.name))
  }
  if (flat?.length) {
    return [{ name: p.event || "run", state: p.status, steps: flat }]
  }
  return []
}

export function stagesOf(p: Pipeline, flat?: Step[]): Stage[] {
  return jobsOf(p, flat || p.steps).map((j) => ({ name: j.name, state: j.state }))
}

export async function withWorkflows(pipes: Pipeline[]): Promise<Pipeline[]> {
  return Promise.all(
    pipes.map(async (p) => {
      if (p.workflows?.length) return p
      const { owner, name } = splitRepo(p.repo)
      if (!owner || !name || !p.number) return p
      try {
        const d = await api.pipeline(owner, name, p.number)
        return { ...p, ...d.pipeline, workflows: d.pipeline.workflows, steps: d.steps }
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
      const ov = await api.repo(owner, name, ref, path)
      if (ov.file) out.push(ov.file)
    } catch {
      /* missing */
    }
  }

  let dirEntries: { path?: string; name: string; type: string }[] = []
  try {
    const dir = await api.repo(owner, name, ref, ".woodpecker")
    dirEntries = dir.entries || []
  } catch {
    dirEntries = []
  }

  const paths = [
    ".woodpecker.yml",
    ".woodpecker.yaml",
    ...dirEntries
      .filter((e) => e.type === "file" || e.type === "blob")
      .map((e) => e.path || `.woodpecker/${e.name}`),
  ]
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
