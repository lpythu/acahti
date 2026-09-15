import { CircleCheckIcon, CircleDashedIcon, CircleDotIcon, CircleXIcon, Loader2Icon } from "lucide-react"
import { cn } from "cn"

import { resolveTone } from "@/components/status-badge"

const ICON = {
  success: CircleCheckIcon,
  running: Loader2Icon,
  pending: CircleDashedIcon,
  warning: CircleDotIcon,
  danger: CircleXIcon,
}

const CLASS = {
  success:
    "text-white [&>circle]:fill-emerald-600 [&>circle]:stroke-emerald-600 dark:[&>circle]:fill-emerald-400 dark:[&>circle]:stroke-emerald-400",
  running: "text-sky-600 dark:text-sky-400 animate-spin",
  pending: "text-muted-foreground",
  warning:
    "text-amber-600 [&>circle:last-child]:fill-amber-600 [&>circle:last-child]:stroke-amber-600 [&>circle:first-child]:fill-white [&>circle:first-child]:stroke-none dark:text-amber-400 dark:[&>circle:last-child]:fill-amber-400 dark:[&>circle:last-child]:stroke-amber-400",
  danger:
    "text-white [&>circle]:fill-red-600 [&>circle]:stroke-red-600 dark:[&>circle]:fill-red-400 dark:[&>circle]:stroke-red-400",
}

export function RunStatusIcon({ status, className }: { status: string; className?: string }) {
  const tone = resolveTone(status)
  const Icon = ICON[tone]
  return <Icon className={cn("size-4", CLASS[tone], className)} />
}
