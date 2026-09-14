export function shortSha(sha: string) {
  return sha ? sha.slice(0, 8) : "—"
}

export function commitTitle(message?: string) {
  if (!message) return ""
  return message.split("\n")[0] || ""
}
