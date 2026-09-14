import { Link, useSearchParams } from "react-router-dom"

import { PageFrame } from "@/components/page-frame"
import { Avatar, AvatarFallback } from "@/components/ui/avatar"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { useLoad } from "@/hooks/use-load"
import { useT } from "@/i18n/i18n"
import { api, codeGroupOf, groupRepos, splitRepo } from "@/lib/api"

export function ReposPage() {
  const t = useT()
  const [sp] = useSearchParams()
  const group = sp.get("group") || ""
  const { data, error, loading } = useLoad(async () => (await api.repos()).repos || [], [])
  const groups = groupRepos(data || [])
  const repos = (data || []).filter((r) => {
    if (!group) return true
    return codeGroupOf(splitRepo(r.full_name || r.name).name) === group
  })

  return (
    <PageFrame
      loading={loading && !data}
      error={error}
      empty={!!data && (group ? repos.length === 0 : groups.length === 0)}
      emptyText={t("noRepos")}
      header={<p className="text-sm text-muted-foreground">{group ? group : t("reposDesc")}</p>}
      skeleton="table"
    >
      {!group && groups.length ? (
        <ul className="divide-y rounded-md border">
          {groups.map((g) => (
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
                  <p className="text-xs text-muted-foreground">{t("reposInGroup", { n: g.repos.length })}</p>
                </div>
              </Link>
            </li>
          ))}
        </ul>
      ) : null}
      {group && repos.length ? (
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
                      {name}
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
