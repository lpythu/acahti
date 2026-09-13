import { Link } from "react-router-dom"

import { PageFrame } from "@/components/page-frame"
import { StatusBadge } from "@/components/status-badge"
import { Button } from "@/components/ui/button"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { useEvents } from "@/hooks/use-events"
import { useLoad } from "@/hooks/use-load"
import { useT } from "@/i18n/i18n"
import { api, splitRepo } from "@/lib/api"

export function BoardPage() {
  const t = useT()
  const { data, error, loading, reload } = useLoad(() => api.board(), [])
  useEvents(reload)

  const box = data
  const empty = !!box && box.prs.length === 0 && box.blocked.length === 0 && box.failed.length === 0

  return (
    <PageFrame
      loading={loading && !data}
      error={error}
      empty={empty}
      emptyText={t("inboxEmpty")}
      header={<p className="text-sm text-muted-foreground">{t("boardDesc")}</p>}
      className="gap-6"
      skeleton="table"
    >
      {box?.blocked.length ? (
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
              {box.blocked.map((p) => {
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
                    <TableCell>
                      <Button
                        size="sm"
                        onClick={async () => {
                          await api.approve(owner, name, p.number)
                          await reload()
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
        </section>
      ) : null}

      {box?.failed.length ? (
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
              {box.failed.map((p) => {
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
                  </TableRow>
                )
              })}
            </TableBody>
          </Table>
        </section>
      ) : null}

      {box?.prs.length ? (
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
              {box.prs.map((pr) => {
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
        </section>
      ) : null}
    </PageFrame>
  )
}
