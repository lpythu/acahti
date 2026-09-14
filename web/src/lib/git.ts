export function shortSha(sha: string) {
  return sha ? sha.slice(0, 8) : "—"
}

export function commitTitle(message?: string) {
  if (!message) return ""
  return message.split("\n")[0] || ""
}

export function commitDate(value?: string) {
  if (!value) return ""
  const d = new Date(value)
  if (Number.isNaN(d.getTime())) return value
  return d.toLocaleString()
}
