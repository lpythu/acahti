import { useNavigate } from "react-router-dom"

import { PagedList } from "@/components/paged-list"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { usePage } from "@/hooks/use-page"
import { useT } from "@/i18n/i18n"
import { api } from "@/lib/api"
import { shortSha } from "@/lib/git"
import { useRepo } from "@/pages/repo-layout"

export function RepoBranchesPage() {
  const t = useT()
  const nav = useNavigate()
  const { owner, name } = useRepo()
  const list = usePage((q) => api.branches(owner, name, q), [owner, name])

  return (
    <PagedList list={list} emptyText={t("noRepos")} skeleton="table">
      {(items) => (
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>{t("branches")}</TableHead>
              <TableHead>{t("sha")}</TableHead>
              <TableHead>{t("defaultBranch")}</TableHead>
              <TableHead>{t("protected")}</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {items.map((b) => (
              <TableRow key={b.name}>
                <TableCell>
                  <button
                    className="hover:underline"
                    type="button"
                    onClick={() => nav(`/repos/${owner}/${name}?ref=${encodeURIComponent(b.name)}`)}
                  >
                    {b.name}
                  </button>
                </TableCell>
                <TableCell className="font-mono text-xs">{shortSha(b.sha)}</TableCell>
                <TableCell>{b.default ? t("defaultBranch") : ""}</TableCell>
                <TableCell>{b.protected ? t("protected") : ""}</TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      )}
    </PagedList>
  )
}
