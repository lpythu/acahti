import { CheckCircle2Icon, CircleDashedIcon, CircleDotIcon, Loader2Icon, XCircleIcon } from "lucide-react"
import { cn } from "cn"

import { resolveTone } from "@/components/status-badge"

const ICON = {
  success: CheckCircle2Icon,
  running: Loader2Icon,
  pending: CircleDashedIcon,
  warning: CircleDotIcon,
  danger: XCircleIcon,
}

const CLASS = {
  success: "text-emerald-600 dark:text-emerald-400",
  running: "text-sky-600 dark:text-sky-400 animate-spin",
  pending: "text-muted-foreground",
  warning: "text-amber-600 dark:text-amber-400",
  danger: "text-red-600 dark:text-red-400",
}

export function RunStatusIcon({ status, className }: { status: string; className?: string }) {
  const tone = resolveTone(status)
  const Icon = ICON[tone]
  return <Icon className={cn("size-4", CLASS[tone], className)} />
}
