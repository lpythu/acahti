import { Link } from "react-router-dom"

import { PagedList } from "@/components/paged-list"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { usePage } from "@/hooks/use-page"
import { useT } from "@/i18n/i18n"
import { api } from "@/lib/api"
import { useRepo } from "@/pages/repo-layout"

export function RepoPullsPage() {
  const t = useT()
  const { owner, name } = useRepo()
  const list = usePage((q) => api.pulls(owner, name, q), [owner, name])

  return (
    <PagedList list={list} emptyText={t("noPulls")} skeleton="table">
      {(items) => (
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>PR</TableHead>
              <TableHead>{t("title")}</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {items.map((pr) => (
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
      )}
    </PagedList>
  )
}
