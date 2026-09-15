import { useState } from "react"
import { useNavigate, useSearchParams } from "react-router-dom"

import { PagedList } from "@/components/paged-list"
import { PipelineRunRow } from "@/components/pipeline-run-row"
import { useEvents } from "@/hooks/use-events"
import { usePage } from "@/hooks/use-page"
import { useT } from "@/i18n/i18n"
import { api, repoName, splitRepo, type Pipeline } from "@/lib/api"
import { pipelineHref } from "@/lib/nav"
import { asPipeline, upsertRun } from "@/lib/pipeline"

export function PipelinesPage() {
  const t = useT()
  const nav = useNavigate()
  const [sp] = useSearchParams()
  const team = sp.get("team") || ""
  const repo = sp.get("repo") || ""
  const list = usePage((q) => api.pipelines({ ...q, repo, team }), [repo, team])
  useEvents((ev) => {
    if (ev.type !== "pipeline.updated") return
    const next = asPipeline(ev.data)
    if (!next) return
    if (repo && next.repo !== repo) return
    list.apply((page) => upsertRun(page, next, list.page))
  })
  const [busy, setBusy] = useState("")
  const [actionErr, setActionErr] = useState("")

  async function runPipe(p: Pipeline) {
    const { owner, name } = splitRepo(p.repo)
    setBusy(p.repo)
    setActionErr("")
    try {
      const next = await api.trigger(owner, name, p.branch || "dev")
      nav(pipelineHref(owner, name, next.number))
    } catch (err) {
      setActionErr(err instanceof Error ? err.message : t("loadError"))
      setBusy("")
    }
  }

  return (
    <PagedList
      list={{ ...list, error: list.error || actionErr }}
      emptyText={t("noPipelines")}
      skeleton="lines"
      header={
        team || repo ? (
          <p className="text-sm text-muted-foreground">{team || repoName(repo)}</p>
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
