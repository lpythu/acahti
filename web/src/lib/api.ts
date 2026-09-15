import { pageQS, type Page, type PageQuery } from "@/lib/page"

export type Me = {
  user: string
  admin: boolean
  root_url: string
  org: string
  git_name?: string
  git_email?: string
}

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
  user?: { login: string; full_name?: string }
  head?: { ref: string; sha: string }
  base?: { ref: string }
}

export type Step = {
  id?: number
  pid: number
  name: string
  state: string
  error?: string
  type?: string
}

export type PipelineJob = {
  name: string
  state: string
  pid?: number
  children?: Step[]
}

export type Pipeline = {
  repo: string
  number: number
  status: string
  event: string
  title: string
  message?: string
  branch?: string
  ref?: string
  author?: string
  avatar?: string
  commit?: string
  error?: string
  created?: number
  started?: number
  finished?: number
  jobs?: PipelineJob[]
  steps?: Step[]
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
  ci: string
  postgres: string
  upgrade_hint: string
}

export type User = { login: string; email: string; is_admin: boolean; full_name: string; teams?: string[] }

export type Invite = { code: string }

export type Perm = { admin: boolean; push: boolean; pull: boolean }

export type Repo = {
  name: string
  full_name: string
  private: boolean
  default_branch: string
  clone_url?: string
  team?: string
  updated?: number
  archived?: boolean
  permissions?: Perm
}

export type AccessPerson = { login: string; author?: string; permission: string }

export type TeamAccess = {
  name: string
  can_manage: boolean
  members: AccessPerson[]
  repos: Repo[]
}

export type RepoAccess = {
  team: string
  can_manage: boolean
  inherited: AccessPerson[]
  direct: AccessPerson[]
}

export type Package = { id: number; name: string; version: string; type: string; created_at?: string }

export type PackageRow = {
  type: string
  name: string
  latest: string
  updated_at: string
}

export type Status = {
  status: string
  context: string
  description: string
  target_url: string
}

export type Comment = {
  id: number
  body: string
  user: { login: string; full_name?: string }
  created_at: string
}

export type RepoTeam = { team: string; count: number }

export type BranchInfo = {
  name: string
  sha: string
  default: boolean
  protected: boolean
}

export type TagInfo = {
  name: string
  sha: string
}

export type CommitPerson = { name: string; email?: string; date: string }

export type CommitUser = { login: string; full_name?: string; avatar_url?: string }

export type CommitFile = {
  filename: string
  status: string
  additions: number
  deletions: number
  changes?: number
  previous_filename?: string
  patch?: string
}

export type CommitStats = { total: number; additions: number; deletions: number }

export type Commit = {
  sha: string
  html_url?: string
  commit?: {
    message: string
    author?: CommitPerson
    committer?: CommitPerson
  }
  author?: CommitUser
  committer?: CommitUser
  parents?: { sha: string }[]
  files?: CommitFile[]
  stats?: CommitStats
  check_status?: string
  check_url?: string
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

export type RepoHeader = {
  repo: Repo
  clone_https: string
  ref: string
}

export type RepoContents = Page<ContentEntry> & {
  ref: string
  path: string
  file?: FileBlob
  readme: string
}

export type CommitDetail = Page<CommitFile> & {
  commit: Commit
  stats: CommitStats
}

export type PRDetail = {
  pr: PR
  checks: Status[]
  green: boolean
}

export type PipelineDetail = {
  pipeline: Pipeline
  steps: Step[]
  team?: string
  files?: FileBlob[]
}

export type NavTeam = {
  team: string
  repos: Repo[]
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
  const ct = res.headers.get("content-type") || ""
  if (text && !ct.includes("application/json")) {
    throw Object.assign(new Error(res.ok ? `${path} returned ${ct || "html"}` : res.statusText), {
      status: res.status,
    })
  }
  let data: { error?: string; message?: string } = {}
  try {
    data = text ? JSON.parse(text) : {}
  } catch {
    throw Object.assign(new Error(`${path} returned invalid JSON`), { status: res.status })
  }
  if (!res.ok) {
    throw Object.assign(new Error(data.error || data.message || res.statusText), {
      status: res.status,
    })
  }
  return data as T
}

export const api = {
  public: () => req<{ root_url: string; org: string; skill: string; join: string; mcp: string }>("/ui/public"),
  me: () => req<Me>("/ui/me"),
  login: (username: string, password: string) =>
    req<Me>("/ui/login", { method: "POST", body: JSON.stringify({ username, password }) }),
  join: (username: string, password: string, code: string) =>
    req<Me>("/ui/join", { method: "POST", body: JSON.stringify({ username, password, code }) }),
  oauthApprove: (body: { client_id: string; redirect_uri: string; state: string; code_challenge: string }) =>
    req<{ redirect: string }>("/ui/oauth/approve", { method: "POST", body: JSON.stringify(body) }),
  invites: () => req<{ invites: Invite[]; join: string }>("/ui/invites"),
  createInvite: () => req<{ code: string; url: string }>("/ui/invites", { method: "POST" }),
  deleteInvite: (code: string) => req<{ ok: boolean }>(`/ui/invites/${code}`, { method: "DELETE" }),
  logout: () => req<{ ok: boolean }>("/ui/logout", { method: "POST" }),
  boardPRs: (q?: PageQuery) => req<Page<PR>>(`/ui/board${pageQS(q, { section: "prs" })}`),
  boardPipes: (q?: PageQuery) => req<Page<Pipeline>>(`/ui/board${pageQS(q, { section: "pipes" })}`),
  users: (q?: PageQuery) => req<Page<User>>(`/ui/users${pageQS(q)}`),
  createUser: (username: string, password: string, admin: boolean) =>
    req<{ user: User }>("/ui/users", {
      method: "POST",
      body: JSON.stringify({ username, password, admin }),
    }),
  stack: () => req<Stack>("/ui/stack"),
  agents: (q?: PageQuery) => req<Page<Agent>>(`/ui/agents${pageQS(q)}`),
  setPassword: (username: string, password: string) =>
    req<{ ok: boolean }>("/ui/password", {
      method: "POST",
      body: JSON.stringify({ username, password }),
    }),
  setGitName: (username: string, git_name: string) =>
    req<{ git_name: string }>(`/ui/users/${encodeURIComponent(username)}`, {
      method: "PATCH",
      body: JSON.stringify({ git_name }),
    }),
  navTree: () => req<NavTeam[]>("/ui/nav/tree"),
  repoTeams: (q?: PageQuery) => req<Page<RepoTeam>>(`/ui/repos${pageQS(q, { teams: "1" })}`),
  repos: (q?: PageQuery, team?: string) => req<Page<Repo>>(`/ui/repos${pageQS(q, { team })}`),
  team: (name: string) => req<TeamAccess>(`/ui/teams/${encodeURIComponent(name)}`),
  createTeam: (name: string) =>
    req<TeamAccess>("/ui/teams", { method: "POST", body: JSON.stringify({ name }) }),
  deleteTeam: (name: string) => req<{ ok: boolean }>(`/ui/teams/${encodeURIComponent(name)}`, { method: "DELETE" }),
  setTeamMember: (team: string, login: string, permission: string) =>
    req<{ ok: boolean }>(`/ui/teams/${encodeURIComponent(team)}/members/${encodeURIComponent(login)}`, {
      method: "PUT",
      body: JSON.stringify({ permission }),
    }),
  removeTeamMember: (team: string, login: string) =>
    req<{ ok: boolean }>(`/ui/teams/${encodeURIComponent(team)}/members/${encodeURIComponent(login)}`, {
      method: "DELETE",
    }),
  addTeamRepo: (team: string, repo: string) =>
    req<{ ok: boolean }>(`/ui/teams/${encodeURIComponent(team)}/repos/${encodeURIComponent(repo)}`, {
      method: "PUT",
    }),
  removeTeamRepo: (team: string, repo: string) =>
    req<{ ok: boolean }>(`/ui/teams/${encodeURIComponent(team)}/repos/${encodeURIComponent(repo)}`, {
      method: "DELETE",
    }),
  repoAccess: (owner: string, name: string) => req<RepoAccess>(`/ui/repos/${owner}/${name}/access`),
  moveRepo: async (owner: string, name: string, team: string, from = "") => {
    const listed = await req<Page<RepoTeam>>(`/ui/repos${pageQS({ page_size: 200 }, { teams: "1" })}`)
    if (!listed.items.some((row) => row.team === team)) {
      try {
        await req<TeamAccess>("/ui/teams", { method: "POST", body: JSON.stringify({ name: team }) })
      } catch {
        /* already exists */
      }
    }
    if (from && from !== team) {
      await req<{ ok: boolean }>(`/ui/teams/${encodeURIComponent(from)}/repos/${encodeURIComponent(name)}`, {
        method: "DELETE",
      })
    }
    await req<{ ok: boolean }>(`/ui/teams/${encodeURIComponent(team)}/repos/${encodeURIComponent(name)}`, {
      method: "PUT",
    })
    return req<RepoAccess>(`/ui/repos/${owner}/${name}/access`)
  },
  setCollaborator: (owner: string, name: string, login: string, permission: string) =>
    req<{ ok: boolean }>(`/ui/repos/${owner}/${name}/collaborators/${encodeURIComponent(login)}`, {
      method: "PUT",
      body: JSON.stringify({ permission }),
    }),
  removeCollaborator: (owner: string, name: string, login: string) =>
    req<{ ok: boolean }>(`/ui/repos/${owner}/${name}/collaborators/${encodeURIComponent(login)}`, {
      method: "DELETE",
    }),
  repo: (owner: string, name: string, ref?: string) =>
    req<RepoHeader>(`/ui/repos/${owner}/${name}${ref ? `?ref=${encodeURIComponent(ref)}` : ""}`),
  contents: (owner: string, name: string, opts?: PageQuery & { ref?: string; path?: string }) =>
    req<RepoContents>(
      `/ui/repos/${owner}/${name}/contents${pageQS(opts, { ref: opts?.ref, path: opts?.path })}`,
    ),
  commits: (owner: string, name: string, opts?: PageQuery & { ref?: string }) =>
    req<Page<Commit>>(`/ui/repos/${owner}/${name}/commits${pageQS(opts, { ref: opts?.ref })}`),
  branches: (owner: string, name: string, q?: PageQuery) =>
    req<Page<BranchInfo>>(`/ui/repos/${owner}/${name}/branches${pageQS(q)}`),
  patchBranch: (owner: string, name: string, branch: string, body: { default?: boolean; protected?: boolean }) =>
    req<{ ok: boolean }>(`/ui/repos/${owner}/${name}/branches/${encodeURIComponent(branch)}`, {
      method: "PATCH",
      body: JSON.stringify(body),
    }),
  tags: (owner: string, name: string, q?: PageQuery) =>
    req<Page<TagInfo>>(`/ui/repos/${owner}/${name}/tags${pageQS(q)}`),
  pulls: (owner: string, name: string, q?: PageQuery & { state?: string }) =>
    req<Page<PR>>(`/ui/repos/${owner}/${name}/pulls${pageQS(q, { state: q?.state })}`),
  commit: (owner: string, name: string, sha: string, q?: PageQuery) =>
    req<CommitDetail>(`/ui/repos/${owner}/${name}/commits/${encodeURIComponent(sha)}${pageQS(q)}`),
  pull: (owner: string, name: string, n: number) =>
    req<PRDetail>(`/ui/repos/${owner}/${name}/pulls/${n}`),
  pullComments: (owner: string, name: string, n: number, q?: PageQuery) =>
    req<Page<Comment>>(`/ui/repos/${owner}/${name}/pulls/${n}/comments${pageQS(q)}`),
  mergePull: (owner: string, name: string, n: number) =>
    req<{ merged: boolean }>(`/ui/repos/${owner}/${name}/pulls/${n}/merge`, { method: "POST" }),
  pipelines: (q?: PageQuery & { repo?: string; team?: string }) =>
    req<Page<Pipeline>>(`/ui/pipelines${pageQS(q, { repo: q?.repo, team: q?.team })}`),
  pipeline: (owner: string, name: string, n: number) =>
    req<PipelineDetail>(`/ui/pipelines/${owner}/${name}/${n}`),
  pipelineLog: (owner: string, name: string, n: number, step: number) =>
    req<{ log: string }>(`/ui/pipelines/${owner}/${name}/${n}/log?step=${step}`),
  rerun: (owner: string, name: string, n: number) =>
    req<Pipeline>(`/ui/pipelines/${owner}/${name}/${n}/rerun`, { method: "POST" }),
  cancel: (owner: string, name: string, n: number) =>
    req<{ ok: boolean }>(`/ui/pipelines/${owner}/${name}/${n}/cancel`, { method: "POST" }),
  deletePipeline: (owner: string, name: string, n: number) =>
    req<{ ok: boolean }>(`/ui/pipelines/${owner}/${name}/${n}`, { method: "DELETE" }),
  approve: (owner: string, name: string, n: number) =>
    req<{ ok: boolean }>(`/ui/pipelines/${owner}/${name}/${n}/approve`, { method: "POST" }),
  packages: (q?: PageQuery, kind?: string) =>
    req<Page<PackageRow>>(`/ui/packages${pageQS(q, { kind })}`),
  packageVersions: (kind: string, name: string, q?: PageQuery) =>
    req<Page<Package>>(`/ui/packages/${encodeURIComponent(kind)}/${encodeURIComponent(name)}${pageQS(q)}`),
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

/** Display name without org prefix (saidc/ops → ops). */
export function repoName(full: string): string {
  return splitRepo(full).name
}

