function pad2(n: number) {
  return String(n).padStart(2, "0")
}

function formatDate(d: Date) {
  return `${d.getFullYear()}-${pad2(d.getMonth() + 1)}-${pad2(d.getDate())} ${pad2(d.getHours())}:${pad2(d.getMinutes())}:${pad2(d.getSeconds())}`
}

export function formatStamp(value?: string) {
  if (!value) return "—"
  const d = new Date(value)
  if (Number.isNaN(d.getTime())) return value
  return formatDate(d)
}

export function formatUnix(sec?: number) {
  if (!sec) return "—"
  const d = new Date(sec * 1000)
  if (Number.isNaN(d.getTime())) return "—"
  return formatDate(d)
}

export function formatSize(bytes?: number) {
  if (!bytes) return "—"
  const kib = 1024
  if (bytes < kib * kib) return `${(bytes / kib).toFixed(2)} KiB`
  return `${(bytes / kib / kib).toFixed(3)} MiB`
}
