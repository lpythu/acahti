import { useState } from "react"
import { Link, useNavigate } from "react-router-dom"

import { PagedList } from "@/components/paged-list"
import { StatusBadge } from "@/components/status-badge"
import { Button } from "@/components/ui/button"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { usePage } from "@/hooks/use-page"
import { useT } from "@/i18n/i18n"
import { api } from "@/lib/api"
import { useRepo } from "@/pages/repo-layout"

export function RepoPipelinesPage() {
  const t = useT()
  const nav = useNavigate()
  const { owner, name, data } = useRepo()
  const list = usePage(
    (q) => api.pipelines({ ...q, repo: `${owner}/${name}` }),
    [owner, name],
  )
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
      skeleton="table"
      header={
        <Button className="w-fit" disabled={busy || !data} onClick={() => void runPipe()}>
          {t("run")}
        </Button>
      }
    >
      {(items) => (
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>#</TableHead>
              <TableHead>{t("status")}</TableHead>
              <TableHead>{t("title")}</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {items.map((p) => (
              <TableRow key={p.number}>
                <TableCell>
                  <Link className="hover:underline" to={`/pipelines/${owner}/${name}/${p.number}`}>
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
      )}
    </PagedList>
  )
}
