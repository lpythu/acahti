import { useState } from "react"
import { useNavigate, useSearchParams } from "react-router-dom"
import { toast } from "sonner"

import { PagedList } from "@/components/paged-list"
import { PipelineRunRow } from "@/components/pipeline-run-row"
import { QueueStrip } from "@/components/queue-strip"
import { useEvents } from "@/hooks/use-events"
import { useLoad } from "@/hooks/use-load"
import { usePage } from "@/hooks/use-page"
import { useT } from "@/i18n/i18n"
import { api, repoName, splitRepo, type Pipeline } from "@/lib/api"
import { pipelineHref } from "@/lib/nav"
import { asPipeline, upsertHead } from "@/lib/pipeline"
import { useSession } from "@/lib/session"

export function PipelinesPage() {
  const t = useT()
  const nav = useNavigate()
  const { me } = useSession()
  const [sp] = useSearchParams()
  const team = sp.get("team") || ""
  const repo = sp.get("repo") || ""
  const list = usePage((q) => api.pipelines({ ...q, repo, team }), [repo, team])
  const queue = useLoad(() => api.queue(), [me?.admin], Boolean(me?.admin))
  useEvents((ev) => {
    if (ev.type !== "pipeline.updated") return
    const next = asPipeline(ev.data)
    if (!next) return
    if (repo && next.repo !== repo) return
    list.apply((page) => upsertHead(page, next, list.page))
    if (me?.admin) void queue.reload()
  })
  const [busy, setBusy] = useState("")

  async function runPipe(p: Pipeline) {
    const { owner, name } = splitRepo(p.repo)
    setBusy(p.repo)
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
        queue.data || team || repo ? (
          <div className="flex flex-col gap-1">
            {queue.data ? <QueueStrip queue={queue.data} /> : null}
            {team || repo ? (
              <p className="text-sm text-muted-foreground">{team || repoName(repo)}</p>
            ) : null}
          </div>
        ) : undefined
      }
    >
      {(items) => (
        <ul className="divide-y rounded-md border">
          {items.map((p) => (
            <PipelineRunRow
              key={`${p.repo}-${p.number}`}
              pipe={p}
              busy={busy === p.repo}
              onRun={() => void runPipe(p)}
            />
          ))}
        </ul>
      )}
    </PagedList>
  )
}
