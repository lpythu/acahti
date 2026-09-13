import { Link } from "react-router-dom"

import { PageFrame } from "@/components/page-frame"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { useT } from "@/i18n/i18n"
import { useRepo } from "@/pages/repo-layout"

export function RepoPullsPage() {
  const t = useT()
  const { owner, name, data, error, loading } = useRepo()
  const pulls = data?.pulls || []

  return (
    <PageFrame
      loading={loading && !data}
      error={error}
      empty={!!data && pulls.length === 0}
      emptyText={t("noPulls")}
      skeleton="table"
    >
      {pulls.length ? (
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>PR</TableHead>
              <TableHead>{t("title")}</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {pulls.map((pr) => (
              <TableRow key={pr.number}>
                <TableCell>#{pr.number}</TableCell>
                <TableCell>
                  <Link className="hover:underline" to={`/repos/${owner}/${name}/pulls/${pr.number}`}>
                    {pr.title}
                  </Link>
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      ) : null}
    </PageFrame>
  )
}
