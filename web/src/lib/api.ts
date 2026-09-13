export type Me = { user: string; admin: boolean; root_url: string; org: string }

export type PR = {
  number: number
  title: string
  state?: string
  body?: string
  repo: string
  url?: string
  html_url?: string
  mergeable?: boolean
  merged?: boolean
  user?: { login: string }
  head?: { ref: string; sha: string }
  base?: { ref: string }
}

export type Pipeline = {
  repo: string
  number: number
  status: string
  event: string
  title: string
  branch?: string
  author?: string
  error?: string
}

export type Step = {
  id?: number
  pid: number
  name: string
  state: string
  error?: string
}

export type Agent = {
  name: string
  last_contact: number
  labels: unknown
  version?: string
  platform?: string
}

export type Stack = {
  version: string
  forgejo: string
  woodpecker: string
  caddy: string
  postgres: string
  agents: Agent[]
  upgrade_hint: string
}

export type User = { login: string; email: string; is_admin: boolean; full_name: string }

export type PublicKey = { id: number; title: string; key: string }

export type AccessToken = { id: number; name: string; token_last_eight: string }

export type Repo = {
  name: string
  full_name: string
  private: boolean
  default_branch: string
  clone_url?: string
}

export type Package = { id: number; name: string; version: string; type: string }

export type Status = {
  status: string
  context: string
  description: string
  target_url: string
}

export type Comment = {
  id: number
  body: string
  user: { login: string }
  created_at: string
}

export type Inbox = {
  prs: PR[]
  blocked: Pipeline[]
  failed: Pipeline[]
}

export type BranchInfo = {
  name: string
  sha: string
  default: boolean
  protected: boolean
}

export type Commit = {
  sha: string
  commit?: { message: string; author?: { name: string; date: string } }
}

export type ContentEntry = {
  name: string
  path: string
  type: string
  sha?: string
}

export type FileBlob = {
  name: string
  path: string
  content: string
}

export type RepoOverview = {
  repo: Repo
  clone_https: string
  clone_ssh: string
  ref: string
  path: string
  branches: BranchInfo[]
  protections: string[]
  commits: Commit[]
  entries: ContentEntry[]
  file?: FileBlob
  readme: string
  pulls: PR[]
  pipes: Pipeline[]
}

export type PackageGroup = {
  type: string
  name: string
  latest: string
  versions: Package[]
}

export type PRDetail = {
  pr: PR
  checks: Status[]
  green: boolean
  comments: Comment[]
}

export type PipelineDetail = {
  pipeline: Pipeline
  steps: Step[]
}

async function req<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(path, {
    credentials: "same-origin",
    headers: { "Content-Type": "application/json", ...(init?.headers || {}) },
    ...init,
  })
  if (res.status === 401) {
    throw Object.assign(new Error("unauthorized"), { status: 401 })
  }
  const text = await res.text()
  const data = text ? JSON.parse(text) : {}
  if (!res.ok) {
    throw Object.assign(new Error(data.error || data.message || res.statusText), {
      status: res.status,
    })
  }
  return data as T
}

export const api = {
  me: () => req<Me>("/ui/me"),
  login: (username: string, password: string) =>
    req<Me>("/ui/login", { method: "POST", body: JSON.stringify({ username, password }) }),
  logout: () => req<{ ok: boolean }>("/ui/logout", { method: "POST" }),
  board: () => req<Inbox>("/ui/board"),
  users: () => req<{ users: User[] }>("/ui/users"),
  createUser: (username: string, password: string, admin: boolean) =>
    req<{ user: User }>("/ui/users", {
      method: "POST",
      body: JSON.stringify({ username, password, admin }),
    }),
  stack: () => req<Stack>("/ui/stack"),
  setPassword: (username: string, password: string) =>
    req<{ ok: boolean }>("/ui/password", {
      method: "POST",
      body: JSON.stringify({ username, password }),
    }),
  keys: () => req<{ keys: PublicKey[] }>("/ui/keys"),
  addKey: (title: string, key: string) =>
    req<{ ok: boolean }>("/ui/keys", { method: "POST", body: JSON.stringify({ title, key }) }),
  tokens: () => req<{ tokens: AccessToken[] }>("/ui/tokens"),
  revokeToken: (id: number) => req<{ ok: boolean }>(`/ui/tokens/${id}`, { method: "DELETE" }),
  issueToken: () => req<{ token: string }>("/ui/token", { method: "POST" }),
  repos: () => req<{ repos: Repo[] }>("/ui/repos"),
  repo: (owner: string, name: string, ref?: string, path?: string) => {
    const q = new URLSearchParams()
    if (ref) q.set("ref", ref)
    if (path) q.set("path", path)
    const qs = q.toString()
    return req<RepoOverview>(`/ui/repos/${owner}/${name}${qs ? `?${qs}` : ""}`)
  },
  pull: (owner: string, name: string, n: number) =>
    req<PRDetail>(`/ui/repos/${owner}/${name}/pulls/${n}`),
  mergePull: (owner: string, name: string, n: number) =>
    req<{ merged: boolean }>(`/ui/repos/${owner}/${name}/pulls/${n}/merge`, { method: "POST" }),
  pipelines: (repo?: string) =>
    req<{ pipes: Pipeline[] }>(repo ? `/ui/pipelines?repo=${encodeURIComponent(repo)}` : "/ui/pipelines"),
  pipeline: (owner: string, name: string, n: number) =>
    req<PipelineDetail>(`/ui/pipelines/${owner}/${name}/${n}`),
  pipelineLog: (owner: string, name: string, n: number, step: number) =>
    req<{ log: string }>(`/ui/pipelines/${owner}/${name}/${n}/log?step=${step}`),
  rerun: (owner: string, name: string, n: number) =>
    req<Pipeline>(`/ui/pipelines/${owner}/${name}/${n}/rerun`, { method: "POST" }),
  approve: (owner: string, name: string, n: number) =>
    req<{ ok: boolean }>(`/ui/pipelines/${owner}/${name}/${n}/approve`, { method: "POST" }),
  packages: (kind?: string) =>
    req<{ packages: Package[] }>(kind ? `/ui/packages?kind=${encodeURIComponent(kind)}` : "/ui/packages"),
  packageGroup: (kind: string, name: string) =>
    req<PackageGroup>(`/ui/packages/${encodeURIComponent(kind)}/${encodeURIComponent(name)}`),
  trigger: (owner: string, name: string, ref?: string) =>
    req<Pipeline>(`/ui/pipelines/${owner}/${name}/trigger`, {
      method: "POST",
      body: JSON.stringify({ ref: ref || "dev" }),
    }),
}

export function splitRepo(full: string): { owner: string; name: string } {
  const [owner, name] = full.split("/")
  return { owner: owner || "", name: name || full }
}
