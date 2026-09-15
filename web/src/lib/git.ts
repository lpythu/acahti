import type { Commit } from "@/lib/api"

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
