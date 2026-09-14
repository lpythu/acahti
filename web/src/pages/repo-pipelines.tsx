import { useState } from "react"
import { useNavigate } from "react-router-dom"

import { PagedList } from "@/components/paged-list"
import { PipelineRunRow } from "@/components/pipeline-run-row"
import { Button } from "@/components/ui/button"
import { usePage } from "@/hooks/use-page"
import { useT } from "@/i18n/i18n"
import { api } from "@/lib/api"
import { useRepo } from "@/pages/repo-layout"

export function RepoPipelinesPage() {
  const t = useT()
  const nav = useNavigate()
  const { owner, name, data } = useRepo()
  const list = usePage((q) => api.pipelines({ ...q, repo: `${owner}/${name}` }), [owner, name])
  const [busy, setBusy] = useState(false)
  const [actionErr, setActionErr] = useState("")

  async function runPipe() {
    setBusy(true)
    setActionErr("")
    try {
      const p = await api.trigger(owner, name, data?.repo.default_branch || "dev")
      nav(`/pipelines/${owner}/${name}/${p.number}`)
    } catch (err) {
      setActionErr(err instanceof Error ? err.message : t("loadError"))
      setBusy(false)
      await list.reload()
    }
  }

  return (
    <PagedList
      list={{ ...list, error: list.error || actionErr }}
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
