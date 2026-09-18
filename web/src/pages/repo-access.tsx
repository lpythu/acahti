import { type FormEvent, useRef, useState } from "react"
import { Link } from "react-router-dom"
import { toast } from "sonner"

import { AddPersonMenu, PermSelect, permLabel } from "@/components/access-fields"
import { MenuButton } from "@/components/menu-button"
import { MenuPanel } from "@/components/menu-panel"
import { PageFrame } from "@/components/page-frame"
import { Button } from "@/components/ui/button"
import { Field, FieldGroup, FieldLabel } from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Switch } from "@/components/ui/switch"
import { Select, SelectContent, SelectItem, SelectSeparator, SelectTrigger, SelectValue } from "@/components/ui/select"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { useLoad } from "@/hooks/use-load"
import { useT } from "@/i18n/i18n"
import { api, type AccessPerson } from "@/lib/api"
import { useSession } from "@/lib/session"
import { displayName } from "@/lib/user"
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
  const target = (pick || newName).trim()

  async function submit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault()
    if (!target || target === current) return
    try {
      await api.moveRepo(owner, name, target, current)
      toast.success(t("repoMoved"))
      setOpen(false)
      setPick("")
      setNewName("")
      await onMoved()
    } catch (e) {
      toast.error(e instanceof Error ? e.message : t("loadError"))
    }
  }

  return (
    <MenuButton
      label={t("moveTeam")}
      variant="outline"
      open={open}
      onOpenChange={(next) => {
        setOpen(next)
        if (next) {
          setPick("")
          setNewName("")
          void teams.reload()
        }
      }}
      className="w-72"
    >
      <MenuPanel loading={open && teams.loading} error={teams.error}>
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
            <Button type="submit" disabled={!target || target === current}>
              {t("move")}
            </Button>
          </FieldGroup>
        </form>
      </MenuPanel>
    </MenuButton>
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
  const grantBusy = useRef(false)

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
            {data.team && admin ? (
              <div className="flex items-center gap-2 text-sm">
                <Switch
                  id="grant-team"
                  checked={Boolean(data.granted)}
                  onCheckedChange={(v) => {
                    if (grantBusy.current) return
                    grantBusy.current = true
                    void api
                      .setRepoTeamGrant(owner, name, v === true)
                      .then(async () => {
                        toast.success(t("accessUpdated"), { id: "access-updated" })
                        await load.reload()
                      })
                      .catch((e) => toast.error(e instanceof Error ? e.message : t("loadError")))
                      .finally(() => {
                        grantBusy.current = false
                      })
                  }}
                />
                <Label htmlFor="grant-team" className="font-normal">
                  {t("grantTeam")}
                </Label>
              </div>
            ) : null}
            {data.team && data.granted ? <PeopleTable people={data.inherited} empty={t("noInherited")} /> : null}
          </section>
          <section className="flex flex-col gap-3">
            <div className="flex items-center justify-between gap-3">
              <h2 className="text-sm font-medium">{data.team ? t("viaDirect") : t("members")}</h2>
              {manage ? (
                <AddPersonMenu
                  title={t("addMember")}
                  exclude={[...data.inherited, ...data.direct].map((p) => p.login)}
                  onAdd={async (logins, permission) => {
                    await Promise.all(
                      logins.map((login) => api.setCollaborator(owner, name, login, permission)),
                    )
                    toast.success(t("memberAdded"), { id: "member-added" })
                    await load.reload()
                  }}
                />
              ) : null}
            </div>
            <PeopleTable
              people={data.direct}
              empty={t("noDirect")}
              manage={manage}
              onPerm={(login, perm) =>
                void api
                  .setCollaborator(owner, name, login, perm)
                  .then(async () => {
                    toast.success(t("accessUpdated"), { id: "access-updated" })
                    await load.reload()
                  })
                  .catch((e) => toast.error(e instanceof Error ? e.message : t("loadError")))
              }
              onRemove={(login) =>
                void api
                  .removeCollaborator(owner, name, login)
                  .then(async () => {
                    toast.success(t("accessRemoved"), { id: "access-removed" })
                    await load.reload()
                  })
                  .catch((e) => toast.error(e instanceof Error ? e.message : t("loadError")))
              }
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
          <TableHead>{t("gitName")}</TableHead>
          <TableHead>{t("permission")}</TableHead>
          {manage ? <TableHead /> : null}
        </TableRow>
      </TableHeader>
      <TableBody>
        {people.map((p) => (
          <TableRow key={p.login}>
            <TableCell>{displayName(p)}</TableCell>
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
