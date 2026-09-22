import { ChevronDownIcon } from "lucide-react"
import { Link } from "react-router-dom"

import { RunStatusIcon } from "@/components/run-status-icon"
import { Button } from "@/components/ui/button"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { useT } from "@/i18n/i18n"
import type { MessageKey } from "@/i18n/messages"
import type { QueueInfo, QueueTask } from "@/lib/api"
import { repoName } from "@/lib/api"
import { pipelineHref } from "@/lib/nav"
import { waitLine } from "@/lib/pipeline"

type QueuePipe = {
  key: string
  repo: string
  number: number
  tasks: QueueTask[]
}

function pipeKey(task: QueueTask) {
  if (task.repo && task.pipeline_number) return `${task.repo}#${task.pipeline_number}`
  return ""
}

function groupPipes(tasks: QueueTask[], skip: Set<string>) {
  const out: QueuePipe[] = []
  const by = new Map<string, QueuePipe>()
  for (const task of tasks) {
    const key = pipeKey(task)
    if (!key || skip.has(key)) continue
    let pipe = by.get(key)
    if (!pipe) {
      pipe = { key, repo: task.repo || "", number: task.pipeline_number || 0, tasks: [] }
      by.set(key, pipe)
      out.push(pipe)
    }
    pipe.tasks.push(task)
  }
  return out
}

// Same rule as the strip counts: a run with a running job and a queued job is running.
function queuePipes(queue: QueueInfo) {
  const running = groupPipes(queue.running || [], new Set())
  const pending = groupPipes(queue.pending || [], new Set(running.map((pipe) => pipe.key)))
  return { running, pending }
}

function pipeTitle(pipe: QueuePipe) {
  const name = repoName(pipe.repo) || pipe.repo
  return pipe.number ? `${name} #${pipe.number}` : name
}

function pipeDetail(
  pipe: QueuePipe,
  t: (key: MessageKey, vars?: Record<string, string | number>) => string,
) {
  const jobs = [...new Set(pipe.tasks.map((task) => task.name).filter(Boolean))]
  const agents = [...new Set(pipe.tasks.map((task) => task.agent).filter((agent): agent is string => Boolean(agent)))]
  const position = pipe.tasks.map((task) => task.queue_position || 0).find((n) => n > 0)
  const wait = pipe.tasks.find((task) => task.wait)?.wait
  const bits: string[] = []
  if (jobs.length) bits.push(jobs.join(", "))
  if (wait && wait !== "queue") {
    const line = waitLine({ status: "pending", wait, queue_position: position })
    if (line) bits.push(t(line.key, line.vars))
  } else if (position) {
    bits.push(t("waitQueuedN", { n: position }))
  } else if (agents.length) {
    bits.push(agents.map((agent) => t("waitAgent", { agent })).join(", "))
  }
  return bits.join(" · ")
}

function QueuePipeItem({ pipe, status }: { pipe: QueuePipe; status: "running" | "pending" }) {
  const t = useT()
  const [owner, name] = pipe.repo.split("/")
  const title = pipeTitle(pipe)
  const detail = pipeDetail(pipe, t)
  const href = owner && name && pipe.number ? pipelineHref(owner, name, pipe.number) : ""
  const body = (
    <>
      <RunStatusIcon status={status} className="mt-0.5" />
      <span className="min-w-0 flex-1">
        <span className="block truncate">{title}</span>
        {detail ? <span className="block truncate text-xs text-muted-foreground">{detail}</span> : null}
      </span>
    </>
  )
  if (!href) {
    return (
      <DropdownMenuItem className="items-start gap-2" disabled label={title}>
        {body}
      </DropdownMenuItem>
    )
  }
  return (
    <DropdownMenuItem className="items-start gap-2" label={title} render={<Link to={href} />}>
      {body}
    </DropdownMenuItem>
  )
}

function QueueMenu({
  label,
  empty,
  pipes,
  status,
}: {
  label: string
  empty: string
  pipes: QueuePipe[]
  status: "running" | "pending"
}) {
  return (
    <DropdownMenu>
      <DropdownMenuTrigger
        render={<Button type="button" variant="ghost" size="sm" className="px-1.5 font-normal text-muted-foreground" />}
      >
        {label}
        <ChevronDownIcon />
      </DropdownMenuTrigger>
      <DropdownMenuContent align="start" className="w-80">
        {pipes.length ? (
          pipes.map((pipe) => <QueuePipeItem key={pipe.key} pipe={pipe} status={status} />)
        ) : (
          <DropdownMenuItem disabled className="text-muted-foreground">
            {empty}
          </DropdownMenuItem>
        )}
      </DropdownMenuContent>
    </DropdownMenu>
  )
}

export function QueuePaused({ paused }: { paused?: boolean }) {
  const t = useT()
  if (!paused) return null
  return <p className="text-sm text-amber-700 dark:text-amber-400">{t("queuePaused")}</p>
}

export function QueueStrip({ queue }: { queue: QueueInfo }) {
  const t = useT()
  const { running, pending } = queuePipes(queue)
  return (
    <div className="flex items-center">
      <QueueMenu
        label={t("queueRunningN", { n: running.length })}
        empty={t("queueEmptyRunning")}
        pipes={running}
        status="running"
      />
      <span className="text-sm text-muted-foreground" aria-hidden>
        ·
      </span>
      <QueueMenu
        label={t("queueQueuedN", { n: pending.length })}
        empty={t("queueEmptyQueued")}
        pipes={pending}
        status="pending"
      />
    </div>
  )
}
