import { type FormEvent, useState } from "react"
import { Link } from "react-router-dom"

import { AddPersonMenu, PermSelect, permLabel } from "@/components/access-fields"
import { PageFrame } from "@/components/page-frame"
import { Button } from "@/components/ui/button"
import { Field, FieldGroup, FieldLabel } from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover"
import { Select, SelectContent, SelectItem, SelectSeparator, SelectTrigger, SelectValue } from "@/components/ui/select"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { useLoad } from "@/hooks/use-load"
import { useT } from "@/i18n/i18n"
import { api, type AccessPerson } from "@/lib/api"
import { useSession } from "@/lib/session"
import { useRepo } from "@/pages/repo-layout"

function MoveTeamMenu({
  current,
  owner,
  name,
  onMoved,
}: {
  current: string
  owner: string
  name: string
  onMoved: () => Promise<void>
}) {
  const t = useT()
  const teams = useLoad(() => api.repoTeams({ page_size: 200 }), [])
  const [open, setOpen] = useState(false)
  const [pick, setPick] = useState("")
  const [newName, setNewName] = useState("")
  const [err, setErr] = useState("")
  const target = (pick || newName).trim()

  async function submit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault()
    if (!target || target === current) return
    setErr("")
    try {
      await api.moveRepo(owner, name, target, current)
      setOpen(false)
      setPick("")
      setNewName("")
      await onMoved()
    } catch (e) {
      setErr(e instanceof Error ? e.message : t("loadError"))
    }
  }

  return (
    <Popover
      open={open}
      onOpenChange={(next) => {
        setOpen(next)
        if (next) {
          setPick("")
          setNewName("")
          setErr("")
          void teams.reload()
        }
      }}
    >
      <PopoverTrigger render={<Button type="button" size="sm" variant="outline" />}>{t("moveTeam")}</PopoverTrigger>
      <PopoverContent className="w-72">
        <form onSubmit={(e) => void submit(e)}>
          <FieldGroup>
            <Field>
              <FieldLabel>{t("selectTeam")}</FieldLabel>
              <Select
                value={pick || null}
                onValueChange={(v) => {
                  setPick(String(v ?? ""))
                  setNewName("")
                }}
              >
                <SelectTrigger className="w-full">
                  <SelectValue placeholder={t("selectTeam")}>{pick || newName || null}</SelectValue>
                </SelectTrigger>
                <SelectContent>
                  {(teams.data?.items ?? []).map((row) => (
                    <SelectItem key={row.team} value={row.team}>
                      {row.team}
                    </SelectItem>
                  ))}
                  {(teams.data?.items ?? []).length ? <SelectSeparator /> : null}
                  <div
                    className="p-1"
                    onPointerDown={(e) => e.preventDefault()}
                    onKeyDown={(e) => e.stopPropagation()}
                  >
                    <Input
                      value={newName}
                      onChange={(e) => {
                        setNewName(e.target.value)
                        setPick("")
                      }}
                      placeholder={t("newTeam")}
                      autoComplete="off"
                    />
                  </div>
                </SelectContent>
              </Select>
            </Field>
            {err ? <p className="text-sm text-destructive">{err}</p> : null}
            <Button type="submit" disabled={!target || target === current}>
              {t("move")}
            </Button>
          </FieldGroup>
        </form>
      </PopoverContent>
    </Popover>
  )
}

export function RepoAccessPage() {
  const t = useT()
  const { me } = useSession()
  const { owner, name } = useRepo()
  const load = useLoad(() => api.repoAccess(owner, name), [owner, name])
  const data = load.data
  const manage = Boolean(data?.can_manage)
  const admin = Boolean(me?.admin)

  return (
    <PageFrame loading={load.loading && !data} error={load.error}>
      {data ? (
        <div className="flex flex-col gap-8">
          <section className="flex flex-col gap-3">
            <div className="flex items-center justify-between gap-3">
              <h2 className="text-sm font-medium">{t("team")}</h2>
              {admin ? (
                <MoveTeamMenu current={data.team} owner={owner} name={name} onMoved={() => load.reload()} />
              ) : null}
            </div>
            <p className="text-sm">
              {data.team ? (
                <Link className="hover:underline" to={`/admin/teams/${encodeURIComponent(data.team)}`}>
                  {data.team}
                </Link>
              ) : (
                <span className="text-muted-foreground">{t("unassignedRepos")}</span>
              )}
            </p>
            {data.team ? <PeopleTable people={data.inherited} empty={t("noInherited")} /> : null}
          </section>
          <section className="flex flex-col gap-3">
            <div className="flex items-center justify-between gap-3">
              <h2 className="text-sm font-medium">{data.team ? t("viaDirect") : t("members")}</h2>
              {manage ? (
                <AddPersonMenu
                  title={t("addMember")}
                  exclude={[...data.inherited, ...data.direct].map((p) => p.login)}
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
