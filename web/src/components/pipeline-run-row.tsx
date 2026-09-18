import { CalendarIcon, ClockIcon, PlayIcon } from "lucide-react"

import { PipelineRunHead } from "@/components/pipeline-run-head"
import { PipelineJobDots } from "@/components/pipeline-stages"
import { Button } from "@/components/ui/button"
import { useLocale, useT } from "@/i18n/i18n"
import { splitRepo, type Pipeline } from "@/lib/api"
import { formatDuration, formatUnix, formatUnixWhen } from "@/lib/format"
import { pipelineHref } from "@/lib/nav"
import { jobDotsOf, waitLine } from "@/lib/pipeline"

export function PipelineRunRow({
  pipe,
  hideRepo,
  busy,
  onRun,
  onApprove,
}: {
  pipe: Pipeline
  hideRepo?: boolean
  busy?: boolean
  onRun?: () => void
  onApprove?: () => void
}) {
  const t = useT()
  const locale = useLocale()
  const { owner, name } = splitRepo(pipe.repo)
  const href = pipelineHref(owner, name, pipe.number)
  const when = formatUnixWhen(pipe.started || pipe.created, locale)
  const exact = formatUnix(pipe.started || pipe.created)
  const duration = formatDuration(pipe.started, pipe.finished, pipe.status)
  const wait = waitLine(pipe)
  const jobs = jobDotsOf(pipe)
  const blocked = pipe.status === "blocked"

  return (
    <li className="flex items-center gap-3 px-3 py-3 hover:bg-muted/50">
      <div className="min-w-0 flex-1 basis-32 @2xl/main:basis-40 @4xl/main:basis-48">
        <PipelineRunHead pipe={pipe} hideRepo={hideRepo} titleTo={href} />
      </div>
      <div className="flex min-w-0 max-w-48 flex-1 items-center @xl/main:max-w-56 @4xl/main:max-w-xs @5xl/main:max-w-sm">
        <PipelineJobDots jobs={jobs} />
      </div>
      <div className="flex w-28 shrink-0 flex-col items-end justify-center gap-0.5 text-xs text-muted-foreground @xl/main:w-36 @3xl/main:w-40">
        <span className="inline-flex items-center gap-1.5" title={exact}>
          <CalendarIcon className="size-3.5" />
          {when}
        </span>
        {wait ? (
          <span className="inline-flex items-center gap-1.5">
            <ClockIcon className="size-3.5" />
            {t(wait.key, wait.vars)}
          </span>
        ) : duration ? (
          <span className="inline-flex items-center gap-1.5">
            <ClockIcon className="size-3.5" />
            {duration}
          </span>
        ) : null}
      </div>
      {blocked && onApprove ? (
        <Button type="button" size="sm" disabled={busy} onClick={onApprove}>
          {t("approve")}
        </Button>
      ) : onRun ? (
        <Button
          type="button"
          size="icon-xs"
          variant="ghost"
          disabled={busy}
          aria-label={t("run")}
          onClick={onRun}
        >
          <PlayIcon />
        </Button>
      ) : null}
    </li>
  )
}
