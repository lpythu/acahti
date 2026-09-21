import type { Activity } from "@/lib/api"
import { splitRepo } from "@/lib/api"

export function activityHref(a: Activity): string {
  const { owner, name } = splitRepo(a.repo?.full_name || "")
  if (!owner || !name) return "/repos"
  const base = `/repos/${owner}/${name}`
  if (a.op_type === "push" || a.op_type === "commit_repo") {
    const sha = (a.content || "")
      .split(/[\n,]+/)
      .map((s) => s.trim())
      .find((s) => /^[0-9a-f]{7,40}$/i.test(s))
    if (sha) return `${base}/commits/${sha}`
  }
  if (
    a.op_type === "create_pull_request" ||
    a.op_type === "merge_pull_request" ||
    a.op_type === "close_pull_request" ||
    a.op_type === "reopen_pull_request"
  ) {
    const n = Number((a.content || "").trim())
    if (n > 0) return `${base}/pulls/${n}`
  }
  return base
}

export function activityKind(op: string): "push" | "pr" | "issue" | "repo" | "other" {
  switch (op) {
    case "push":
    case "commit_repo":
      return "push"
    case "create_pull_request":
    case "merge_pull_request":
    case "close_pull_request":
    case "reopen_pull_request":
      return "pr"
    case "create_issue":
    case "close_issue":
    case "reopen_issue":
    case "comment_issue":
      return "issue"
    case "create_repo":
    case "fork_repo":
      return "repo"
    default:
      return "other"
  }
}
