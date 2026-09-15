export function displayName(p?: { author?: string; full_name?: string; name?: string; login?: string } | null) {
  return (p?.author || p?.full_name || p?.name || p?.login || "").trim()
}
