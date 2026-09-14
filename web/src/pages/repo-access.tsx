import { Link } from "react-router-dom"

import { AddPersonMenu, PermSelect, permLabel } from "@/components/access-fields"
import { PageFrame } from "@/components/page-frame"
import { Button } from "@/components/ui/button"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { useLoad } from "@/hooks/use-load"
import { useT } from "@/i18n/i18n"
import { api, type AccessPerson } from "@/lib/api"
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
          {data.group ? (
            <section className="flex flex-col gap-3">
              <h2 className="text-sm font-medium">
                <Link className="hover:underline" to={`/repos?group=${encodeURIComponent(data.group)}`}>
                  {t("viaGroup", { group: data.group })}
                </Link>
              </h2>
              <PeopleTable people={data.inherited} empty={t("noInherited")} />
            </section>
          ) : null}
          <section className="flex flex-col gap-3">
            <div className="flex items-center justify-between gap-3">
              <h2 className="text-sm font-medium">{data.group ? t("viaDirect") : t("members")}</h2>
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
            <PeopleTable
              people={data.direct}
              empty={t("noDirect")}
              manage={manage}
              onPerm={(login, perm) => void api.setCollaborator(owner, name, login, perm).then(() => load.reload())}
              onRemove={(login) => void api.removeCollaborator(owner, name, login).then(() => load.reload())}
            />
          </section>
        </div>
      ) : null}
    </PageFrame>
  )
}

function PeopleTable({
  people,
  empty,
  manage,
  onPerm,
  onRemove,
}: {
  people: AccessPerson[]
  empty: string
  manage?: boolean
  onPerm?: (login: string, perm: "read" | "write" | "admin") => void
  onRemove?: (login: string) => void
}) {
  const t = useT()
  if (!people.length) {
    return <p className="text-sm text-muted-foreground">{empty}</p>
  }
  return (
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead>{t("username")}</TableHead>
          <TableHead>{t("permission")}</TableHead>
          {manage ? <TableHead /> : null}
        </TableRow>
      </TableHeader>
      <TableBody>
        {people.map((p) => (
          <TableRow key={p.login}>
            <TableCell>{p.login}</TableCell>
            <TableCell>
              {manage && onPerm ? (
                <PermSelect value={p.permission} onChange={(perm) => onPerm(p.login, perm)} />
              ) : (
                permLabel(t, p.permission)
              )}
            </TableCell>
            {manage && onRemove ? (
              <TableCell className="text-right">
                <Button type="button" size="sm" variant="outline" onClick={() => onRemove(p.login)}>
                  {t("remove")}
                </Button>
              </TableCell>
            ) : null}
          </TableRow>
        ))}
      </TableBody>
    </Table>
  )
}
