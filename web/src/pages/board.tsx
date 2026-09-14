import { useCallback } from "react"
import { Link } from "react-router-dom"

import { Pager } from "@/components/paged-list"
import { PageFrame } from "@/components/page-frame"
import { StatusBadge } from "@/components/status-badge"
import { Button } from "@/components/ui/button"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { useEvents } from "@/hooks/use-events"
import { usePage } from "@/hooks/use-page"
import { useT } from "@/i18n/i18n"
import { api, splitRepo } from "@/lib/api"

export function BoardPage() {
  const t = useT()
  const blocked = usePage((q) => api.boardBlocked(q), [], { param: "blocked" })
  const failed = usePage((q) => api.boardFailed(q), [], { param: "failed" })
  const prs = usePage((q) => api.boardPRs(q), [], { param: "prs" })
  const reload = useCallback(() => {
    void blocked.reload()
    void failed.reload()
    void prs.reload()
  }, [blocked.reload, failed.reload, prs.reload])
  useEvents(reload)

  const loading = [blocked, failed, prs].every((x) => x.loading && !x.data)
  const empty = blocked.empty && failed.empty && prs.empty
  const error = blocked.error || failed.error || prs.error

  return (
    <PageFrame
      loading={loading}
      error={error}
      empty={empty}
      emptyText={t("inboxEmpty")}
      header={<p className="text-sm text-muted-foreground">{t("boardDesc")}</p>}
      className="gap-6"
      skeleton="table"
    >
      {blocked.items.length ? (
        <section className="flex flex-col gap-2">
          <h2 className="text-sm font-medium">{t("blocked")}</h2>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>{t("repo")}</TableHead>
                <TableHead>#</TableHead>
                <TableHead>{t("status")}</TableHead>
                <TableHead />
              </TableRow>
            </TableHeader>
            <TableBody>
              {blocked.items.map((p) => {
                const { owner, name } = splitRepo(p.repo)
                return (
                  <TableRow key={`${p.repo}-${p.number}`}>
                    <TableCell>
                      <Link className="hover:underline" to={`/pipelines/${owner}/${name}/${p.number}`}>
                        {p.repo}
                      </Link>
                    </TableCell>
                    <TableCell>{p.number}</TableCell>
                    <TableCell>
                      <StatusBadge status={p.status} />
                    </TableCell>
                    <TableCell>
                      <Button
                        size="sm"
                        onClick={async () => {
                          await api.approve(owner, name, p.number)
                          await blocked.reload()
                        }}
                      >
                        {t("approve")}
                      </Button>
                    </TableCell>
                  </TableRow>
                )
              })}
            </TableBody>
          </Table>
          <Pager page={blocked.page} hasMore={blocked.hasMore} onPage={blocked.setPage} />
        </section>
      ) : null}

      {failed.items.length ? (
        <section className="flex flex-col gap-2">
          <h2 className="text-sm font-medium">{t("failed")}</h2>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>{t("repo")}</TableHead>
                <TableHead>#</TableHead>
                <TableHead>{t("status")}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {failed.items.map((p) => {
                const { owner, name } = splitRepo(p.repo)
                return (
                  <TableRow key={`${p.repo}-${p.number}`}>
                    <TableCell>
                      <Link className="hover:underline" to={`/pipelines/${owner}/${name}/${p.number}`}>
                        {p.repo}
                      </Link>
                    </TableCell>
                    <TableCell>{p.number}</TableCell>
                    <TableCell>
                      <StatusBadge status={p.status} />
                    </TableCell>
                  </TableRow>
                )
              })}
            </TableBody>
          </Table>
          <Pager page={failed.page} hasMore={failed.hasMore} onPage={failed.setPage} />
        </section>
      ) : null}

      {prs.items.length ? (
        <section className="flex flex-col gap-2">
          <h2 className="text-sm font-medium">{t("prs")}</h2>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>PR</TableHead>
                <TableHead>{t("title")}</TableHead>
                <TableHead>{t("repo")}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {prs.items.map((pr) => {
                const { owner, name } = splitRepo(pr.repo)
                return (
                  <TableRow key={`${pr.repo}-${pr.number}`}>
                    <TableCell>#{pr.number}</TableCell>
                    <TableCell>
                      <Link className="hover:underline" to={`/repos/${owner}/${name}/pulls/${pr.number}`}>
                        {pr.title}
                      </Link>
                    </TableCell>
                    <TableCell>{pr.repo}</TableCell>
                  </TableRow>
                )
              })}
            </TableBody>
          </Table>
          <Pager page={prs.page} hasMore={prs.hasMore} onPage={prs.setPage} />
        </section>
      ) : null}
    </PageFrame>
  )
}
