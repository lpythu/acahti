import { Link, useSearchParams } from "react-router-dom"

import { PagedList } from "@/components/paged-list"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { useEvents } from "@/hooks/use-events"
import { usePage } from "@/hooks/use-page"
import { useLocale, useT } from "@/i18n/i18n"
import { api, splitRepo } from "@/lib/api"
import { formatUnixWhen } from "@/lib/format"

export function ReposPage() {
  const t = useT()
  const locale = useLocale()
  const [sp] = useSearchParams()
  const team = sp.get("team") || ""
  const repos = usePage((q) => api.repos(q, team || undefined), [team])
  useEvents((ev) => {
    if (ev.type === "catalog.updated") void repos.reload()
  })

  return (
    <PagedList
      list={repos}
      emptyText={t("noRepos")}
      skeleton="table"
      header={team ? <p className="text-sm text-muted-foreground">{team}</p> : undefined}
    >
      {(items) => (
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>{t("repo")}</TableHead>
              <TableHead>{t("team")}</TableHead>
              <TableHead>{t("lastUpdated")}</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {items.map((r) => {
              const { owner, name } = splitRepo(r.full_name || r.name)
              return (
                <TableRow key={r.full_name || r.name}>
                  <TableCell>
                    <Link className="hover:underline" to={`/repos/${owner}/${name}`}>
                      {name}
                    </Link>
                  </TableCell>
                  <TableCell>
                    {r.team ? (
                      <Link className="hover:underline" to={`/repos?team=${encodeURIComponent(r.team)}`}>
                        {r.team}
                      </Link>
                    ) : (
                      "—"
                    )}
                  </TableCell>
                  <TableCell>{formatUnixWhen(r.updated, locale)}</TableCell>
                </TableRow>
              )
            })}
          </TableBody>
        </Table>
      )}
    </PagedList>
  )
}
