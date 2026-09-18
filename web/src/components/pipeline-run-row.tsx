import {
  CalendarIcon,
  ClockIcon,
  GitCommitVerticalIcon,
  GitPullRequestIcon,
  PlayIcon,
  TagIcon,
  TimerIcon,
  ZapIcon,
} from "lucide-react"
import { Link } from "react-router-dom"

import { PipelineJobDots } from "@/components/pipeline-stages"
import { RunStatusIcon } from "@/components/run-status-icon"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { useLocale, useT, useTr } from "@/i18n/i18n"
import { splitRepo, type Pipeline } from "@/lib/api"
import { formatDuration, formatUnix, formatUnixWhen } from "@/lib/format"
import { shortSha } from "@/lib/git"
import { pipelineHref } from "@/lib/nav"
import { argosLinksOf, jobDotsOf, runEventKey, runRef, runTitle, triggerKind, triggerVars, waitLine, type TriggerKind } from "@/lib/pipeline"

function TriggerRefIcon({ kind }: { kind: TriggerKind }) {
  switch (kind) {
    case "tag":
      return <TagIcon />
    case "pr":
      return <GitPullRequestIcon />
    case "cron":
      return <TimerIcon />
    case "manual":
      return <ZapIcon />
    default:
      return <GitCommitVerticalIcon />
  }
}

function triggerRefLabel(kind: TriggerKind) {
  switch (kind) {
    case "tag":
      return "tags" as const
    case "pr":
      return "prs" as const
    case "push":
      return "commits" as const
    default:
      return null
  }
}

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
  const tr = useTr()
  const locale = useLocale()
  const { owner, name } = splitRepo(pipe.repo)
  const href = pipelineHref(owner, name, pipe.number)
  const title = runTitle(pipe)
  const ref = runRef(pipe)
  const kind = triggerKind(pipe.event, pipe.ref)
  const kindLabel = triggerRefLabel(kind)
  const vars = triggerVars(pipe)
  const sha = shortSha(pipe.commit || "")
  const shaText = sha === "—" ? vars.ref : sha
  const author = (
    <span className="font-medium text-foreground">{vars.author}</span>
  )
  const shaNode = pipe.commit && sha !== "—" ? (
    <Link
      to={`/repos/${owner}/${name}/commits/${pipe.commit}`}
      className="font-mono text-sky-700 hover:underline dark:text-sky-400"
    >
      {shaText}
    </Link>
  ) : (
    <span className="font-mono text-sky-700 dark:text-sky-400">{shaText}</span>
  )
  const refMark = <span className="font-medium text-sky-700 dark:text-sky-400">{vars.ref}</span>
  const detail = tr(runEventKey(pipe.event, pipe.ref), { ...vars, sha: shaNode, author, ref: refMark })
  const meta = tr(hideRepo ? "runHeadlineRepo" : "runHeadline", {
    repo: <span className="font-medium text-foreground">{name}</span>,
    number: <span className="tabular-nums text-foreground">{pipe.number}</span>,
    detail,
  })
  const metaTitle = t(hideRepo ? "runHeadlineRepo" : "runHeadline", {
    repo: name,
    number: pipe.number,
    detail: t(runEventKey(pipe.event, pipe.ref), { ...vars, sha: shaText }),
  })
  const when = formatUnixWhen(pipe.started || pipe.created, locale)
  const exact = formatUnix(pipe.started || pipe.created)
  const duration = formatDuration(pipe.started, pipe.finished, pipe.status)
  const wait = waitLine(pipe)
  const jobs = jobDotsOf(pipe)
  const argos = argosLinksOf(pipe)
  const blocked = pipe.status === "blocked"

  return (
    <li className="flex items-center gap-3 px-3 py-3 hover:bg-muted/50">
      <div className="min-w-0 flex-1 basis-32 @2xl/main:basis-40 @4xl/main:basis-48">
        <div className="flex min-w-0 items-center gap-2">
          <Link to={href} className="flex min-w-0 items-center gap-3">
            <RunStatusIcon status={pipe.status} className="size-5 shrink-0" />
            <span className="min-w-0 truncate text-sm font-medium hover:underline" title={title}>
              {title}
            </span>
          </Link>
          {ref ? (
            <Badge
              variant="outline"
              className="hidden max-w-28 shrink-0 font-normal text-sky-700 @xs/main:inline-flex dark:text-sky-400"
              title={kindLabel ? t(kindLabel) : undefined}
            >
              <TriggerRefIcon kind={kind} />
              <span className="truncate">{ref}</span>
            </Badge>
          ) : null}
          {argos.map((item) => (
            <Badge
              key={item.url}
              variant="outline"
              render={<a href={item.url} target="_blank" rel="noreferrer" />}
              className="max-w-28 shrink-0 font-normal text-sky-700 dark:text-sky-400"
            >
              {t("argosDash")}
              {argos.length > 1 ? ` ${item.name.replace(/^e2e\./, "")}` : ""}
            </Badge>
          ))}
        </div>
        <p className="mt-0.5 truncate pl-8 text-xs text-muted-foreground" title={metaTitle}>
          {meta}
        </p>
        {pipe.error ? (
          <p className="mt-0.5 truncate pl-8 text-xs text-muted-foreground" title={pipe.error}>
            {pipe.error}
          </p>
        ) : null}
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
