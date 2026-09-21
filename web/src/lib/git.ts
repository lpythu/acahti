import type { Commit, Page } from "@/lib/api"

export function shortSha(sha: string) {
  return sha ? sha.slice(0, 8) : "—"
}

export function commitTitle(message?: string) {
  if (!message) return ""
  return message.split("\n")[0] || ""
}

export function commitAuthor(c?: Commit | null) {
  return c?.commit?.author?.name || c?.author?.full_name || c?.author?.login || "—"
}

export function isCommitSha(q: string) {
  return /^[0-9a-f]{7,40}$/i.test(q.trim())
}

export function commitMatches(c: Commit, q: string) {
  const needle = q.trim().toLowerCase()
  if (!needle) return true
  if (c.sha.toLowerCase().includes(needle)) return true
  if ((c.commit?.message || "").toLowerCase().includes(needle)) return true
  if ((c.commit?.author?.name || "").toLowerCase().includes(needle)) return true
  if ((c.commit?.author?.email || "").toLowerCase().includes(needle)) return true
  if ((c.author?.login || "").toLowerCase().includes(needle)) return true
  if ((c.author?.full_name || "").toLowerCase().includes(needle)) return true
  return false
}

export function filterCommitPage(res: Page<Commit>, q: string): Page<Commit> {
  const query = q.trim()
  if (!query) return res
  const items = res.items.filter((c) => commitMatches(c, query))
  return { ...res, items, has_more: res.has_more && items.length === res.items.length }
}
