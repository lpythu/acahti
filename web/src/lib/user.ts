export function displayName(
  p?: { author?: string; full_name?: string; git_name?: string; name?: string; login?: string } | null,
) {
  return (p?.author || p?.full_name || p?.git_name || p?.name || p?.login || "").trim()
}
