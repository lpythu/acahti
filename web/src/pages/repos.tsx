import { Link, useSearchParams } from "react-router-dom"

import { PagedList } from "@/components/paged-list"
import { Avatar, AvatarFallback } from "@/components/ui/avatar"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { usePage } from "@/hooks/use-page"
import { useT } from "@/i18n/i18n"
import { api, splitRepo } from "@/lib/api"

export function ReposPage() {
  const t = useT()
  const [sp] = useSearchParams()
  const group = sp.get("group") || ""
  const groups = usePage((q) => api.repoGroups(q), [], { enabled: !group })
  const repos = usePage((q) => api.repos(q, group), [group], {
    enabled: Boolean(group),
  })

  if (group) {
    return (
      <PagedList
        list={repos}
        emptyText={t("noRepos")}
        header={<p className="text-sm text-muted-foreground">{group}</p>}
        skeleton="table"
      >
        {(items) => (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>{t("repo")}</TableHead>
                <TableHead>{t("defaultBranch")}</TableHead>
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
                    <TableCell>{r.default_branch || "dev"}</TableCell>
                  </TableRow>
                )
              })}
            </TableBody>
          </Table>
        )}
      </PagedList>
    )
  }

  return (
    <PagedList
      list={groups}
      emptyText={t("noRepos")}
      header={<p className="text-sm text-muted-foreground">{t("reposDesc")}</p>}
    >
      {(items) => (
        <ul className="divide-y rounded-md border">
          {items.map((g) => (
            <li key={g.group}>
              <Link
                className="flex items-center gap-3 px-3 py-3 hover:bg-muted/50"
                to={`/repos?group=${encodeURIComponent(g.group)}`}
              >
                <Avatar className="rounded-lg after:rounded-lg">
                  <AvatarFallback className="rounded-lg">{g.group.slice(0, 1).toUpperCase()}</AvatarFallback>
                </Avatar>
                <div className="min-w-0">
                  <p className="truncate text-sm font-medium">{g.group}</p>
                  <p className="text-xs text-muted-foreground">{t("reposInGroup", { n: g.count })}</p>
                </div>
              </Link>
            </li>
          ))}
        </ul>
      )}
    </PagedList>
  )
}
