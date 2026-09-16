import { Badge } from "@/components/ui/badge"
import { useT } from "@/i18n/i18n"
import type { MessageKey } from "@/i18n/messages"

type Tone = "success" | "running" | "pending" | "queue" | "waiting" | "warning" | "danger"

const TONE: Record<string, Tone> = {
  success: "success",
  merged: "success",
  online: "success",
  running: "running",
  pending: "pending",
  open: "pending",
  blocked: "warning",
  skipped: "pending",
  stale: "warning",
  failure: "danger",
  error: "danger",
  failed: "danger",
  killed: "danger",
  declined: "danger",
  closed: "danger",
}

const TONE_CLASS: Record<Tone, string> = {
  success: "border-transparent bg-emerald-500/10 text-emerald-700 dark:text-emerald-400",
  running: "border-transparent bg-sky-500/10 text-sky-700 dark:text-sky-400",
  pending: "border-transparent bg-slate-500/10 text-slate-700 dark:text-slate-300",
  queue: "border-transparent bg-indigo-500/10 text-indigo-700 dark:text-indigo-400",
  waiting: "border-transparent bg-slate-500/10 text-slate-700 dark:text-slate-300",
  warning: "border-transparent bg-amber-500/10 text-amber-800 dark:text-amber-400",
  danger: "border-transparent bg-red-500/10 text-red-700 dark:text-red-400",
}

const LABEL: Record<string, MessageKey> = {
  success: "statusSuccess",
  merged: "statusMerged",
  online: "statusOnline",
  running: "statusRunning",
  pending: "statusPending",
  open: "statusOpen",
  blocked: "statusBlocked",
  skipped: "statusSkipped",
  stale: "statusStale",
  failure: "statusFailure",
  error: "statusError",
  failed: "statusFailure",
  killed: "statusKilled",
  declined: "statusDeclined",
  closed: "statusClosed",
}

const DANGER = new Set(["failure", "error", "failed", "killed", "declined"])

function waitTone(wait?: string): Tone | null {
  switch (wait) {
    case "queue":
      return "queue"
    case "concurrency":
      return "warning"
    case "deps":
      return "waiting"
    default:
      return null
  }
}

export function resolveTone(status: string, wait?: string): Tone {
  const s = status.toLowerCase()
  if (DANGER.has(s)) return "danger"
  const w = waitTone(wait)
  if (w) return w
  return TONE[s] ?? "pending"
}

export function statusText(status: string, t: (key: MessageKey) => string, wait?: string): string {
  const s = status.toLowerCase()
  if (DANGER.has(s)) {
    const key = LABEL[s]
    return key ? t(key) : status
  }
  if (wait === "queue") return t("statusQueued")
  if (wait === "deps") return t("statusWaiting")
  if (wait === "concurrency") return t("statusSlot")
  const key = LABEL[s]
  return key ? t(key) : status
}

export function StatusBadge({ status, wait }: { status: string; wait?: string }) {
  const t = useT()
  return (
    <Badge variant="outline" className={TONE_CLASS[resolveTone(status, wait)]}>
      {statusText(status, t, wait)}
    </Badge>
  )
}
