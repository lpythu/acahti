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

function clock(d: Date) {
  return `${pad2(d.getHours())}:${pad2(d.getMinutes())}`
}

function monthDay(d: Date, locale: "en" | "zh") {
  if (locale === "zh") return `${d.getMonth() + 1}月${d.getDate()}日`
  return d.toLocaleString("en", { month: "short", day: "numeric" })
}

export function formatUnixWhen(sec?: number, locale: "en" | "zh" = "en") {
  if (!sec) return "—"
  const d = new Date(sec * 1000)
  if (Number.isNaN(d.getTime())) return "—"
  const now = new Date()
  const diff = Math.round((now.getTime() - d.getTime()) / 1000)
  if (diff < 60) return locale === "zh" ? "刚刚" : "just now"
  if (diff < 3600) {
    const m = Math.max(1, Math.floor(diff / 60))
    if (locale === "zh") return `${m} 分钟前`
    return m === 1 ? "1 minute ago" : `${m} minutes ago`
  }
  const time = clock(d)
  if (d.toDateString() === now.toDateString()) {
    return locale === "zh" ? `今天 ${time}` : `Today at ${time}`
  }
  const yest = new Date(now)
  yest.setDate(now.getDate() - 1)
  if (d.toDateString() === yest.toDateString()) {
    return locale === "zh" ? `昨天 ${time}` : `Yesterday at ${time}`
  }
  const day =
    d.getFullYear() === now.getFullYear()
      ? monthDay(d, locale)
      : locale === "zh"
        ? `${d.getFullYear()}年${monthDay(d, locale)}`
        : `${monthDay(d, locale)}, ${d.getFullYear()}`
  return locale === "zh" ? `${day} ${time}` : `${day} at ${time}`
}

export function formatDuration(started?: number, finished?: number, status?: string) {
  if (!started) return ""
  const running = (status || "").toLowerCase() === "running"
  const end = finished || (running ? Math.floor(Date.now() / 1000) : 0)
  if (!end || end < started) return ""
  const sec = end - started
  if (sec < 60) return `${sec}s`
  const m = Math.floor(sec / 60)
  const s = sec % 60
  if (m < 60) return s ? `${m}m ${s}s` : `${m}m`
  return `${Math.floor(m / 60)}h ${m % 60}m`
}

export function formatRelative(value?: string) {
  if (!value) return ""
  const d = new Date(value)
  if (Number.isNaN(d.getTime())) return value
  const sec = Math.round((Date.now() - d.getTime()) / 1000)
  const abs = Math.abs(sec)
  if (abs < 60) return sec >= 0 ? `${abs}s` : `in ${abs}s`
  if (abs < 3600) return sec >= 0 ? `${Math.floor(abs / 60)}m` : `in ${Math.floor(abs / 60)}m`
  if (abs < 86400) return sec >= 0 ? `${Math.floor(abs / 3600)}h` : `in ${Math.floor(abs / 3600)}h`
  if (abs < 86400 * 30) return sec >= 0 ? `${Math.floor(abs / 86400)}d` : `in ${Math.floor(abs / 86400)}d`
  return formatStamp(value)
}

export function dayKey(value?: string) {
  if (!value) return ""
  const d = new Date(value)
  if (Number.isNaN(d.getTime())) return ""
  return `${d.getFullYear()}-${pad2(d.getMonth() + 1)}-${pad2(d.getDate())}`
}
