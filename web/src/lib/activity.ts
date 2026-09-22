import type { Activity } from "@/lib/api"
import { splitRepo } from "@/lib/api"

export type ActivityKind = "commit" | "pr" | "issue" | "release" | "repo" | "other"

const COMMIT = new Set(["push", "commit_repo", "mirror_sync_push"])
const PR = new Set([
  "create_pull_request",
  "merge_pull_request",
  "close_pull_request",
  "reopen_pull_request",
  "comment_pull",
  "approve_pull_request",
  "reject_pull_request",
  "pull_review_dismissed",
  "pull_request_ready_for_review",
  "auto_merge_pull_request",
])
const ISSUE = new Set(["create_issue", "close_issue", "reopen_issue", "comment_issue"])
const RELEASE = new Set(["publish_release", "push_tag", "delete_tag"])
const REPO = new Set([
  "create_repo",
  "rename_repo",
  "star_repo",
  "watch_repo",
  "transfer_repo",
  "fork_repo",
  "delete_branch",
  "mirror_sync_create",
  "mirror_sync_delete",
])

function isSha(s: string): boolean {
  return /^[0-9a-f]{7,40}$/i.test(s)
}

function shaOf(raw: unknown): string {
  if (!raw || typeof raw !== "object") return ""
  const o = raw as { Sha1?: string; sha1?: string }
  const s = o.Sha1 || o.sha1 || ""
  return isSha(s) ? s : ""
}

/** Forgejo push content is PushCommits JSON; older rows are `N\\nsha\\n…`. */
export function activitySha(content: string): string {
  const t = content.trim()
  if (!t) return ""
  if (t.startsWith("{")) {
    try {
      const j = JSON.parse(t) as {
        commits?: unknown[]
        Commits?: unknown[]
        head_commit?: unknown
        HeadCommit?: unknown
      }
      const commits = j.commits || j.Commits || []
      for (const c of [commits[0], j.head_commit, j.HeadCommit, ...commits]) {
        const sha = shaOf(c)
        if (sha) return sha
      }
    } catch {
      /* fall through */
    }
  }
  for (const line of t.split(/[\n,]+/)) {
    const s = line.trim()
    if (isSha(s)) return s
  }
  return t.match(/\b([0-9a-f]{40})\b/i)?.[1] || ""
}

function urlIndex(u: string): number {
  const m = u.match(/\/(?:pulls?|issues)\/(\d+)/i)
  return m ? Number(m[1]) : 0
}

export function activityIndex(a: Activity): number {
  const first = (a.content || "").trim().split(/\n/)[0]?.trim() || ""
  const n = Number(first)
  if (Number.isInteger(n) && n > 0) return n
  const c = a.comment
  for (const u of [c?.pull_request_url, c?.html_url, c?.issue_url]) {
    const i = urlIndex(u || "")
    if (i) return i
  }
  return 0
}

export function activityRef(ref: string): string {
  return ref.replace(/^refs\/(heads|tags)\//, "")
}

function commentIsPR(a: Activity): boolean {
  const c = a.comment
  return Boolean(c?.pull_request_url) || /\/pulls\/\d+/i.test(c?.html_url || "")
}

export function activityKind(a: Activity): ActivityKind {
  const op = a.op_type || ""
  if (COMMIT.has(op)) return "commit"
  if (PR.has(op) || (op === "comment_issue" && commentIsPR(a))) return "pr"
  if (ISSUE.has(op)) return "issue"
  if (RELEASE.has(op)) return "release"
  if (REPO.has(op)) return "repo"
  return "other"
}

export function activityHref(a: Activity): string {
  const { owner, name } = splitRepo(a.repo?.full_name || "")
  if (!owner || !name) return "/repos"
  const base = `/repos/${owner}/${name}`
  const kind = activityKind(a)
  if (kind === "commit") {
    const sha = activitySha(a.content || "")
    if (sha) return `${base}/commits/${sha}`
  }
  if (kind === "pr") {
    const n = activityIndex(a)
    if (n) return `${base}/pulls/${n}`
  }
  if (kind === "release") return `${base}/tags`
  if (a.op_type === "delete_branch") return `${base}/branches`
  return base
}

export function activityDetail(a: Activity): string {
  const kind = activityKind(a)
  if (kind === "commit") {
    const sha = activitySha(a.content || "").slice(0, 7)
    const ref = activityRef(a.ref_name || "")
    return [sha, ref].filter(Boolean).join(" · ")
  }
  if (kind === "pr" || kind === "issue") {
    const n = activityIndex(a)
    return n ? `#${n}` : activityRef(a.ref_name || "")
  }
  if (kind === "release") return activityRef(a.ref_name || a.content || "")
  return activityRef(a.ref_name || "")
}
