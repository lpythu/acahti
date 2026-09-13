import { Link } from "react-router-dom"

import { PageFrame } from "@/components/page-frame"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { useLoad } from "@/hooks/use-load"
import { useT } from "@/i18n/i18n"
import { api, splitRepo } from "@/lib/api"

export function ReposPage() {
  const t = useT()
  const { data, error, loading } = useLoad(async () => (await api.repos()).repos || [], [])
  const repos = data || []

  return (
    <PageFrame
      loading={loading && !data}
      error={error}
      empty={!!data && repos.length === 0}
      emptyText={t("noRepos")}
      header={<p className="text-sm text-muted-foreground">{t("reposDesc")}</p>}
      skeleton="table"
    >
      {repos.length ? (
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>{t("repo")}</TableHead>
              <TableHead>{t("defaultBranch")}</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {repos.map((r) => {
              const { owner, name } = splitRepo(r.full_name || r.name)
              return (
                <TableRow key={r.full_name || r.name}>
                  <TableCell>
                    <Link className="hover:underline" to={`/repos/${owner}/${name}`}>
                      {r.full_name || r.name}
                    </Link>
                  </TableCell>
                  <TableCell>{r.default_branch || "dev"}</TableCell>
                </TableRow>
              )
            })}
          </TableBody>
        </Table>
      ) : null}
    </PageFrame>
  )
}
