import { Badge } from "@/components/ui/badge"
import { useT } from "@/i18n/i18n"
import type { MessageKey } from "@/i18n/messages"

// Colors match Woodpecker 3.18 web/src/components/repo/pipeline/pipeline-status.ts
type Tone = "success" | "running" | "pending" | "neutral" | "danger"

const TONE: Record<string, Tone> = {
  success: "success",
  merged: "success",
  online: "success",
  running: "running",
  started: "running",
  pending: "pending",
  created: "pending",
  open: "pending",
  blocked: "neutral",
  skipped: "neutral",
  canceled: "neutral",
  killed: "neutral",
  stale: "neutral",
  failure: "danger",
  error: "danger",
  failed: "danger",
  declined: "danger",
  closed: "danger",
}

const TONE_CLASS: Record<Tone, string> = {
  success: "border-transparent bg-emerald-500/10 text-emerald-700 dark:text-emerald-400",
  running: "border-transparent bg-sky-500/10 text-sky-700 dark:text-sky-400",
  pending: "border-transparent bg-amber-500/10 text-amber-800 dark:text-amber-400",
  neutral: "border-transparent bg-slate-500/10 text-slate-700 dark:text-slate-300",
  danger: "border-transparent bg-red-500/10 text-red-700 dark:text-red-400",
}

const LABEL: Record<string, MessageKey> = {
  success: "statusSuccess",
  merged: "statusMerged",
  online: "statusOnline",
  running: "statusRunning",
  started: "statusStarted",
  pending: "statusPending",
  created: "statusPending",
  open: "statusOpen",
  blocked: "statusBlocked",
  skipped: "statusSkipped",
  canceled: "statusCanceled",
  killed: "statusKilled",
  stale: "statusStale",
  failure: "statusFailure",
  error: "statusError",
  failed: "statusFailure",
  declined: "statusDeclined",
  closed: "statusClosed",
}

export function resolveTone(status: string): Tone {
  return TONE[status.toLowerCase()] ?? "pending"
}

export function statusText(status: string, t: (key: MessageKey) => string): string {
  const key = LABEL[status.toLowerCase()]
  return key ? t(key) : status
}

export function StatusBadge({ status }: { status: string }) {
  const t = useT()
  return (
    <Badge variant="outline" className={TONE_CLASS[resolveTone(status)]}>
      {statusText(status, t)}
    </Badge>
  )
}
