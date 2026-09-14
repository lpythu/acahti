import { CalendarIcon, ClockIcon, GitBranchIcon, PlayIcon } from "lucide-react"
import { Link } from "react-router-dom"

import { PipelineStages } from "@/components/pipeline-stages"
import { RunStatusIcon } from "@/components/run-status-icon"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { useLocale, useT } from "@/i18n/i18n"
import { splitRepo, type Pipeline } from "@/lib/api"
import { formatDuration, formatUnix, formatUnixWhen } from "@/lib/format"
import { shortSha } from "@/lib/git"
import { pipelineHref } from "@/lib/nav"
import { jobDotsOf, runEventKey, runRef, runTitle, triggerVars } from "@/lib/pipeline"

export function PipelineRunRow({
  pipe,
  hideRepo,
  busy,
  onRun,
}: {
  pipe: Pipeline
  hideRepo?: boolean
  busy?: boolean
  onRun?: () => void
}) {
  const t = useT()
  const locale = useLocale()
  const { owner, name } = splitRepo(pipe.repo)
  const href = pipelineHref(owner, name, pipe.number)
  const title = runTitle(pipe)
  const ref = runRef(pipe)
  const vars = triggerVars(pipe)
  const sha = shortSha(pipe.commit || "")
  const detail = t(runEventKey(pipe.event), { ...vars, sha: sha === "—" ? vars.ref : sha })
  const meta = t(hideRepo ? "runHeadlineRepo" : "runHeadline", {
    repo: name,
    number: pipe.number,
    detail,
  })
  const when = formatUnixWhen(pipe.started || pipe.created, locale)
  const exact = formatUnix(pipe.started || pipe.created)
  const duration = formatDuration(pipe.started, pipe.finished, pipe.status)
  const jobs = jobDotsOf(pipe)

  return (
    <li className="flex items-center gap-3 px-3 py-3 hover:bg-muted/50">
      <Link to={href} className="flex min-w-0 flex-1 items-start gap-3">
        <RunStatusIcon status={pipe.status} className="mt-0.5 size-5" />
        <span className="min-w-0 flex-1">
          <span className="flex min-w-0 items-center gap-2">
            <span className="min-w-0 truncate text-sm font-medium hover:underline">{title}</span>
            {ref ? (
              <Badge variant="outline" className="max-w-28 shrink-0 font-normal text-sky-700 dark:text-sky-400">
                <GitBranchIcon />
                <span className="truncate">{ref}</span>
              </Badge>
            ) : null}
          </span>
          <span className="mt-0.5 block truncate text-xs text-muted-foreground">{meta}</span>
        </span>
      </Link>
      <PipelineStages stages={jobs} />
      <div className="flex w-40 shrink-0 flex-col items-end gap-0.5 text-xs text-muted-foreground">
        <span className="inline-flex items-center gap-1.5" title={exact}>
          <CalendarIcon className="size-3.5" />
          {when}
        </span>
        {duration ? (
          <span className="inline-flex items-center gap-1.5">
            <ClockIcon className="size-3.5" />
            {duration}
          </span>
        ) : null}
      </div>
      {onRun ? (
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
