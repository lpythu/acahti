import { CircleCheckIcon, CircleDashedIcon, CircleIcon, CircleXIcon, Loader2Icon, MinusCircleIcon } from "lucide-react"
import { cn } from "cn"

const ICON = {
  success: CircleCheckIcon,
  running: Loader2Icon,
  pending: CircleIcon,
  blocked: CircleDashedIcon,
  skipped: MinusCircleIcon,
  killed: CircleXIcon,
  danger: CircleXIcon,
}

const CLASS = {
  success:
    "text-white [&>circle]:fill-emerald-600 [&>circle]:stroke-emerald-600 dark:[&>circle]:fill-emerald-400 dark:[&>circle]:stroke-emerald-400",
  running: "text-sky-600 dark:text-sky-400 animate-spin",
  pending: "text-amber-600 dark:text-amber-400",
  blocked: "text-slate-500 dark:text-slate-400",
  skipped: "text-slate-500 dark:text-slate-400",
  killed: "text-slate-500 dark:text-slate-400",
  danger:
    "text-white [&>circle]:fill-red-600 [&>circle]:stroke-red-600 dark:[&>circle]:fill-red-400 dark:[&>circle]:stroke-red-400",
}

function look(status: string): keyof typeof ICON {
  switch (status.toLowerCase()) {
    case "success":
    case "merged":
    case "online":
      return "success"
    case "running":
    case "started":
      return "running"
    case "pending":
    case "created":
    case "open":
      return "pending"
    case "blocked":
    case "stale":
      return "blocked"
    case "skipped":
    case "canceled":
      return "skipped"
    case "killed":
      return "killed"
    case "failure":
    case "error":
    case "failed":
    case "declined":
    case "closed":
      return "danger"
    default:
      return "pending"
  }
}

export function RunStatusIcon({ status, className }: { status: string; className?: string }) {
  const kind = look(status)
  const Icon = ICON[kind]
  return <Icon className={cn("size-4", CLASS[kind], className)} />
}
