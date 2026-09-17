import { useState } from "react"
import { useNavigate, useSearchParams } from "react-router-dom"
import { toast } from "sonner"

import { PagedList } from "@/components/paged-list"
import { PipelineRunRow } from "@/components/pipeline-run-row"
import { QueuePaused, QueueStrip } from "@/components/queue-strip"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { useEvents } from "@/hooks/use-events"
import { useLoad } from "@/hooks/use-load"
import { usePage } from "@/hooks/use-page"
import { useT } from "@/i18n/i18n"
import type { MessageKey } from "@/i18n/messages"
import { api, repoName, splitRepo, type Pipeline } from "@/lib/api"
import { pipelineHref } from "@/lib/nav"
import { asPipeline, failedStatus, pipeFilterMatch, upsertHead } from "@/lib/pipeline"
import { useSession } from "@/lib/session"

const STATUS_FILTERS = ["failed", "blocked", "running", "success"] as const

const STATUS_FILTER_LABEL: Record<string, MessageKey> = {
  all: "filterAll",
  failed: "failed",
  blocked: "blocked",
  running: "statusRunning",
  success: "statusSuccess",
}

function pipeStatus(raw: string | null) {
  return STATUS_FILTERS.includes(raw as (typeof STATUS_FILTERS)[number]) ? raw! : "all"
}

export function PipelinesPage() {
  const t = useT()
  const nav = useNavigate()
  const { me } = useSession()
  const [sp, setSp] = useSearchParams()
  const team = sp.get("team") || ""
  const repo = sp.get("repo") || ""
  const status = pipeStatus(sp.get("status"))
  const list = usePage(
    (q) => api.pipelines({ ...q, repo, team, status: status === "all" ? undefined : status }),
    [repo, team, status],
  )
  const queue = useLoad(() => api.queue(), [me?.admin], Boolean(me?.admin))
  useEvents((ev) => {
    if (ev.type !== "pipeline.updated") return
    const next = asPipeline(ev.data)
    if (!next) return
    if (repo && next.repo !== repo) return
    list.apply((page) => {
      if (!page) return page
      const items = page.items || []
      const i = items.findIndex((p) => p.repo === next.repo)
      if (!pipeFilterMatch(next.status, status)) {
        if (i < 0) return page
        return { ...page, items: items.filter((p) => p.repo !== next.repo) }
      }
      if (
        i >= 0 &&
        items[i].number === next.number &&
        items[i].status === next.status &&
        status !== "all" &&
        status !== "running"
      ) {
        return page
      }
      return upsertHead(page, next, list.page)
    })
    if (me?.admin) void queue.reload()
  })
  const [busy, setBusy] = useState("")

  function setStatus(next: string) {
    const q = new URLSearchParams(sp)
    if (next === "all") q.delete("status")
    else q.set("status", next)
    q.delete("page")
    setSp(q, { replace: true })
  }

  async function approve(p: Pipeline) {
    const { owner, name } = splitRepo(p.repo)
    setBusy(`${p.repo}-${p.number}`)
    try {
      await api.approve(owner, name, p.number)
      toast.success(t("approved"))
      await list.reload()
    } catch (err) {
      toast.error(err instanceof Error ? err.message : t("loadError"))
    } finally {
      setBusy("")
    }
  }

  async function rerun(p: Pipeline) {
    const { owner, name } = splitRepo(p.repo)
    setBusy(`${p.repo}-${p.number}`)
    try {
      const next = await api.rerun(owner, name, p.number)
      toast.success(t("rerunQueued"))
      nav(pipelineHref(owner, name, next.number))
    } catch (err) {
      toast.error(err instanceof Error ? err.message : t("loadError"))
      setBusy("")
    }
  }

  async function runPipe(p: Pipeline) {
    const { owner, name } = splitRepo(p.repo)
    setBusy(`${p.repo}-${p.number}`)
    try {
      const next = await api.trigger(owner, name, p.branch || "dev")
      toast.success(t("runStarted"))
      nav(pipelineHref(owner, name, next.number))
    } catch (err) {
      toast.error(err instanceof Error ? err.message : t("loadError"))
      setBusy("")
    }
  }

  return (
    <PagedList
      list={list}
      emptyText={t("noPipelines")}
      skeleton="lines"
      header={
        <div className="flex flex-col gap-2">
          <QueuePaused paused={queue.data?.paused} />
          <div className="flex items-center justify-between gap-2">
            <div className="flex min-w-0 flex-wrap items-center gap-x-3">
              {queue.data ? <QueueStrip queue={queue.data} /> : null}
              {team || repo ? (
                <p className="text-sm text-muted-foreground">{team || repoName(repo)}</p>
              ) : null}
            </div>
            <Select value={status} onValueChange={(v) => setStatus(String(v ?? "all"))}>
              <SelectTrigger size="sm" className="shrink-0" aria-label={t("status")}>
                <SelectValue>{t(STATUS_FILTER_LABEL[status] ?? "filterAll")}</SelectValue>
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">{t("filterAll")}</SelectItem>
                <SelectItem value="failed">{t("failed")}</SelectItem>
                <SelectItem value="blocked">{t("blocked")}</SelectItem>
                <SelectItem value="running">{t("statusRunning")}</SelectItem>
                <SelectItem value="success">{t("statusSuccess")}</SelectItem>
              </SelectContent>
            </Select>
          </div>
        </div>
      }
    >
      {(items) => (
        <ul className="divide-y rounded-md border">
          {items.map((p) => (
            <PipelineRunRow
              key={`${p.repo}-${p.number}`}
              pipe={p}
              busy={busy === `${p.repo}-${p.number}`}
              onApprove={p.status === "blocked" ? () => void approve(p) : undefined}
              onRun={
                p.status === "blocked"
                  ? undefined
                  : failedStatus(p.status)
                    ? () => void rerun(p)
                    : () => void runPipe(p)
              }
            />
          ))}
        </ul>
      )}
    </PagedList>
  )
}
