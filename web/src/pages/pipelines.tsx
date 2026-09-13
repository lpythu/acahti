import { Link } from "react-router-dom"

import { PageFrame } from "@/components/page-frame"
import { StatusBadge } from "@/components/status-badge"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { useLoad } from "@/hooks/use-load"
import { useT } from "@/i18n/i18n"
import { api, splitRepo } from "@/lib/api"

export function PipelinesPage() {
  const t = useT()
  const { data, error, loading } = useLoad(async () => (await api.pipelines()).pipes || [], [])
  const pipes = data || []

  return (
    <PageFrame
      loading={loading && !data}
      error={error}
      empty={!!data && pipes.length === 0}
      emptyText={t("noRuns")}
      header={<p className="text-sm text-muted-foreground">{t("pipelinesDesc")}</p>}
      skeleton="table"
    >
      {pipes.length ? (
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>{t("repo")}</TableHead>
              <TableHead>#</TableHead>
              <TableHead>{t("status")}</TableHead>
              <TableHead>{t("title")}</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {pipes.map((p) => {
              const { owner, name } = splitRepo(p.repo)
              return (
                <TableRow key={`${p.repo}-${p.number}`}>
                  <TableCell>
                    <Link className="hover:underline" to={`/repos/${owner}/${name}/pipelines/${p.number}`}>
                      {p.repo}
                    </Link>
                  </TableCell>
                  <TableCell>{p.number}</TableCell>
                  <TableCell>
                    <StatusBadge status={p.status} />
                  </TableCell>
                  <TableCell>{p.title || p.event}</TableCell>
                </TableRow>
              )
            })}
          </TableBody>
        </Table>
      ) : null}
    </PageFrame>
  )
}
