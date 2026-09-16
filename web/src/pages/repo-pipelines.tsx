import { useState } from "react"
import { useNavigate } from "react-router-dom"
import { toast } from "sonner"

import { PagedList } from "@/components/paged-list"
import { PipelineRunRow } from "@/components/pipeline-run-row"
import { Button } from "@/components/ui/button"
import { useEvents } from "@/hooks/use-events"
import { usePage } from "@/hooks/use-page"
import { useT } from "@/i18n/i18n"
import { api } from "@/lib/api"
import { asPipeline, upsertRun } from "@/lib/pipeline"
import { pipelineHref } from "@/lib/nav"
import { useRepo } from "@/pages/repo-layout"

export function RepoPipelinesPage() {
  const t = useT()
  const nav = useNavigate()
  const { owner, name, data } = useRepo()
  const list = usePage((q) => api.repoPipelines(owner, name, q), [owner, name])
  useEvents((ev) => {
    if (ev.type !== "pipeline.updated") return
    const next = asPipeline(ev.data)
    if (!next || next.repo !== `${owner}/${name}`) return
    list.apply((page) => upsertRun(page, next, list.page))
  })
  const [busy, setBusy] = useState(false)

  async function runPipe() {
    setBusy(true)
    try {
      const p = await api.trigger(owner, name, data?.repo.default_branch || "dev")
      toast.success(t("runStarted"))
      nav(pipelineHref(owner, name, p.number))
    } catch (err) {
      toast.error(err instanceof Error ? err.message : t("loadError"))
      setBusy(false)
      await list.reload()
    }
  }

  return (
    <PagedList
      list={list}
      emptyText={t("noPipelines")}
      skeleton="lines"
      header={
        <Button className="w-fit" disabled={busy || !data} onClick={() => void runPipe()}>
          {t("run")}
        </Button>
      }
    >
      {(items) => (
        <ul className="divide-y rounded-md border">
          {items.map((p) => (
            <PipelineRunRow key={p.number} pipe={p} hideRepo />
          ))}
        </ul>
      )}
    </PagedList>
  )
}
