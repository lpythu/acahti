import { useNavigate } from "react-router-dom"

import { PagedList } from "@/components/paged-list"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { usePage } from "@/hooks/use-page"
import { useT } from "@/i18n/i18n"
import { api } from "@/lib/api"
import { shortSha } from "@/lib/git"
import { useRepo } from "@/pages/repo-layout"

export function RepoTagsPage() {
  const t = useT()
  const nav = useNavigate()
  const { owner, name } = useRepo()
  const list = usePage((q) => api.tags(owner, name, q), [owner, name])

  return (
    <PagedList list={list} emptyText={t("noTags")} skeleton="table">
      {(items) => (
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>{t("tags")}</TableHead>
              <TableHead>{t("sha")}</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {items.map((tag) => (
              <TableRow key={tag.name}>
                <TableCell>
                  <button
                    className="hover:underline"
                    type="button"
                    onClick={() => nav(`/repos/${owner}/${name}?ref=${encodeURIComponent(tag.name)}`)}
                  >
                    {tag.name}
                  </button>
                </TableCell>
                <TableCell className="font-mono text-xs">{shortSha(tag.sha)}</TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      )}
    </PagedList>
  )
}
