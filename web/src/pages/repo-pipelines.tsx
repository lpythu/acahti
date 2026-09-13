import { useState } from "react"
import { Link, useNavigate } from "react-router-dom"

import { PageFrame } from "@/components/page-frame"
import { StatusBadge } from "@/components/status-badge"
import { Button } from "@/components/ui/button"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { useT } from "@/i18n/i18n"
import { api } from "@/lib/api"
import { useRepo } from "@/pages/repo-layout"

export function RepoPipelinesPage() {
  const t = useT()
  const nav = useNavigate()
  const { owner, name, data, error, loading, reload } = useRepo()
  const [busy, setBusy] = useState(false)
  const [actionErr, setActionErr] = useState("")
  const pipes = data?.pipes || []

  async function runPipe() {
    setBusy(true)
    setActionErr("")
    try {
      const p = await api.trigger(owner, name, data?.ref || "dev")
      nav(`/repos/${owner}/${name}/pipelines/${p.number}`)
    } catch (err) {
      setActionErr(err instanceof Error ? err.message : t("loadError"))
      setBusy(false)
      await reload()
    }
  }

  return (
    <PageFrame
      loading={loading && !data}
      error={error || actionErr}
      empty={!!data && pipes.length === 0}
      emptyText={t("noRuns")}
      skeleton="table"
      header={
        <Button className="w-fit" disabled={busy || !data} onClick={() => void runPipe()}>
          {t("run")}
        </Button>
      }
    >
      {pipes.length ? (
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>#</TableHead>
              <TableHead>{t("status")}</TableHead>
              <TableHead>{t("title")}</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {pipes.map((p) => (
              <TableRow key={p.number}>
                <TableCell>
                  <Link className="hover:underline" to={`/repos/${owner}/${name}/pipelines/${p.number}`}>
                    {p.number}
                  </Link>
                </TableCell>
                <TableCell>
                  <StatusBadge status={p.status} />
                </TableCell>
                <TableCell>{p.title || p.event}</TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      ) : null}
    </PageFrame>
  )
}
