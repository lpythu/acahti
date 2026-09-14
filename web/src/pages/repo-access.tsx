import { Link } from "react-router-dom"

import { AddPersonMenu, PermSelect, permLabel } from "@/components/access-fields"
import { PageFrame } from "@/components/page-frame"
import { Button } from "@/components/ui/button"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { useLoad } from "@/hooks/use-load"
import { useT } from "@/i18n/i18n"
import { api } from "@/lib/api"
import { useRepo } from "@/pages/repo-layout"

export function RepoAccessPage() {
  const t = useT()
  const { owner, name } = useRepo()
  const load = useLoad(() => api.repoAccess(owner, name), [owner, name])
  const data = load.data
  const manage = Boolean(data?.can_manage)

  return (
    <PageFrame loading={load.loading && !data} error={load.error} header={<p className="text-sm text-muted-foreground">{t("accessDesc")}</p>}>
      {data ? (
        <div className="flex flex-col gap-8">
          <section className="flex flex-col gap-3">
            <h2 className="text-sm font-medium">{t("viaGroup")}</h2>
            {data.groups.length ? (
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>{t("codeGroup")}</TableHead>
                    <TableHead>{t("permission")}</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {data.groups.map((g) => (
                    <TableRow key={g.name}>
                      <TableCell>
                        <Link className="hover:underline" to={`/repos?group=${encodeURIComponent(g.name)}`}>
                          {g.name}
                        </Link>
                      </TableCell>
                      <TableCell>{permLabel(t, g.permission)}</TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            ) : (
              <p className="text-sm text-muted-foreground">{t("noRepos")}</p>
            )}
          </section>
          <section className="flex flex-col gap-3">
            <div className="flex items-center justify-between gap-3">
              <h2 className="text-sm font-medium">{t("viaDirect")}</h2>
              {manage ? (
                <AddPersonMenu
                  title={t("addMember")}
                  onAdd={async (login, permission) => {
                    await api.setCollaborator(owner, name, login, permission)
                    await load.reload()
                  }}
                />
              ) : null}
            </div>
            {data.collaborators.length ? (
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>{t("username")}</TableHead>
                    <TableHead>{t("permission")}</TableHead>
                    {manage ? <TableHead /> : null}
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {data.collaborators.map((p) => (
                    <TableRow key={p.login}>
                      <TableCell>{p.login}</TableCell>
                      <TableCell>
                        {manage ? (
                          <PermSelect
                            value={p.permission}
                            onChange={(perm) => {
                              void api.setCollaborator(owner, name, p.login, perm).then(() => load.reload())
                            }}
                          />
                        ) : (
                          permLabel(t, p.permission)
                        )}
                      </TableCell>
                      {manage ? (
                        <TableCell className="text-right">
                          <Button
                            type="button"
                            size="sm"
                            variant="outline"
                            onClick={() => void api.removeCollaborator(owner, name, p.login).then(() => load.reload())}
                          >
                            {t("remove")}
                          </Button>
                        </TableCell>
                      ) : null}
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            ) : (
              <p className="text-sm text-muted-foreground">{t("noMembers")}</p>
            )}
          </section>
        </div>
      ) : null}
    </PageFrame>
  )
}
