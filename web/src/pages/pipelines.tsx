import { useState } from "react"
import { useNavigate, useSearchParams } from "react-router-dom"

import { PagedList } from "@/components/paged-list"
import { PipelineRunRow } from "@/components/pipeline-run-row"
import { usePage } from "@/hooks/use-page"
import { useT } from "@/i18n/i18n"
import { api, splitRepo, type Pipeline } from "@/lib/api"
import { pipelineHref } from "@/lib/nav"

export function PipelinesPage() {
  const t = useT()
  const nav = useNavigate()
  const [sp] = useSearchParams()
  const team = sp.get("team") || ""
  const repo = sp.get("repo") || ""
  const list = usePage((q) => api.pipelines({ ...q, repo, team }), [repo, team])
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
      header={<p className="text-sm text-muted-foreground">{team || repo || t("pipelinesDesc")}</p>}
      skeleton="lines"
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
