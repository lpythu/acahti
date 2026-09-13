export function shortSha(sha: string) {
  return sha ? sha.slice(0, 8) : "—"
}
