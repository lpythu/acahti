import {
  GitCommitVerticalIcon,
  GitPullRequestIcon,
  TagIcon,
  TimerIcon,
  ZapIcon,
} from "lucide-react"
import { Link } from "react-router-dom"

import { RunStatusIcon } from "@/components/run-status-icon"
import { Badge } from "@/components/ui/badge"
import { useT, useTr } from "@/i18n/i18n"
import { splitRepo, type Pipeline } from "@/lib/api"
import { shortSha } from "@/lib/git"
import { argosLinksOf, runEventKey, runRef, runTitle, triggerKind, triggerVars, type TriggerKind } from "@/lib/pipeline"

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

export function PipelineRunHead({
  pipe,
  hideRepo,
  titleTo,
}: {
  pipe: Pipeline
  hideRepo?: boolean
  titleTo?: string
}) {
  const t = useT()
  const tr = useTr()
  const { owner, name } = splitRepo(pipe.repo)
  const title = runTitle(pipe)
  const ref = runRef(pipe)
  const kind = triggerKind(pipe.event, pipe.ref)
  const kindLabel = triggerRefLabel(kind)
  const vars = triggerVars(pipe)
  const sha = shortSha(pipe.commit || "")
  const shaText = sha === "—" ? vars.ref : sha
  const author = <span className="font-medium text-foreground">{vars.author}</span>
  const shaNode =
    pipe.commit && sha !== "—" ? (
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
  const argos = argosLinksOf(pipe)
  const titleInner = (
    <>
      <RunStatusIcon status={pipe.status} className="size-5 shrink-0" />
      <span className="min-w-0 truncate text-sm font-medium" title={title}>
        {title}
      </span>
    </>
  )

  return (
    <div className="min-w-0">
      <div className="flex min-w-0 items-center gap-2">
        {titleTo ? (
          <Link to={titleTo} className="flex min-w-0 items-center gap-3 hover:underline">
            {titleInner}
          </Link>
        ) : (
          <span className="flex min-w-0 items-center gap-3">{titleInner}</span>
        )}
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
  )
}
