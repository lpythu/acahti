import { Link } from "react-router-dom"

import { StatusBadge } from "@/components/status-badge"
import { useT } from "@/i18n/i18n"
import type { QueueInfo, QueueTask } from "@/lib/api"
import { repoName } from "@/lib/api"
import { pipelineHref } from "@/lib/nav"

function TaskLine({ task }: { task: QueueTask }) {
  const repo = task.repo || ""
  const [owner, name] = repo.split("/")
  const label = `${repoName(repo) || repo}${task.pipeline_number ? ` #${task.pipeline_number}` : ""} ${task.name}`
  const inner = (
    <span className="flex min-w-0 items-center gap-2 text-sm">
      <StatusBadge status={task.agent ? "running" : "pending"} wait={task.wait} />
      <span className="truncate">{label}</span>
      {task.agent ? <span className="shrink-0 text-muted-foreground">{task.agent}</span> : null}
    </span>
  )
  if (owner && name && task.pipeline_number) {
    return (
      <Link className="block hover:underline" to={pipelineHref(owner, name, task.pipeline_number)}>
        {inner}
      </Link>
    )
  }
  return inner
}

export function QueuePaused({ paused }: { paused?: boolean }) {
  const t = useT()
  if (!paused) return null
  return <p className="text-sm text-amber-700 dark:text-amber-400">{t("queuePaused")}</p>
}

export function QueueStrip({ queue }: { queue: QueueInfo }) {
  const t = useT()
  const stats = queue.stats || { running_count: 0, pending_count: 0, worker_count: 0 }
  return (
    <p className="text-sm text-muted-foreground">
      {t("queueCounts", {
        running: stats.running_count,
        pending: stats.pending_count,
      })}
    </p>
  )
}

export function QueueLists({ queue }: { queue: QueueInfo }) {
  const t = useT()
  const running = queue.running || []
  const pending = queue.pending || []
  if (!running.length && !pending.length) {
    return <p className="text-sm text-muted-foreground">{t("noQueue")}</p>
  }
  return (
    <div className="grid gap-4 md:grid-cols-2">
      <QueueColumn title={t("statusRunning")} tasks={running} />
      <QueueColumn title={t("statusQueued")} tasks={pending} />
    </div>
  )
}

function QueueColumn({ title, tasks }: { title: string; tasks: QueueTask[] }) {
  return (
    <div className="flex flex-col gap-2">
      <h3 className="text-sm font-medium">
        {title} ({tasks.length})
      </h3>
      {tasks.length ? (
        <ul className="flex flex-col gap-1">
          {tasks.map((task, i) => (
            <li key={`${task.repo}-${task.pipeline_number}-${task.name}-${i}`}>
              <TaskLine task={task} />
            </li>
          ))}
        </ul>
      ) : (
        <p className="text-sm text-muted-foreground">—</p>
      )}
    </div>
  )
}
