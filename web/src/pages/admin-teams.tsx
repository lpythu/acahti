import { type FormEvent, useState } from "react"
import { Link, useNavigate, useParams } from "react-router-dom"

import { AddPersonMenu, PermSelect, permLabel } from "@/components/access-fields"
import { PageFrame } from "@/components/page-frame"
import { PagedList } from "@/components/paged-list"
import { Button } from "@/components/ui/button"
import { Field, FieldGroup, FieldLabel } from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { useEvents } from "@/hooks/use-events"
import { useLoad } from "@/hooks/use-load"
import { usePage } from "@/hooks/use-page"
import { useT } from "@/i18n/i18n"
import { api, splitRepo } from "@/lib/api"

function CreateTeamMenu({ onCreated }: { onCreated: (name: string) => void }) {
  const t = useT()
  const [err, setErr] = useState("")
  async function submit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault()
    const fd = new FormData(e.currentTarget)
    const name = String(fd.get("name") || "").trim()
    if (!name) return
    setErr("")
    try {
      await api.createTeam(name)
      e.currentTarget.reset()
      onCreated(name)
    } catch (e) {
      setErr(e instanceof Error ? e.message : t("loadError"))
    }
  }
  return (
    <Popover>
      <PopoverTrigger render={<Button type="button" size="sm" />}>{t("createTeam")}</PopoverTrigger>
      <PopoverContent className="w-72">
        <form onSubmit={(e) => void submit(e)}>
          <FieldGroup>
            <Field>
              <FieldLabel htmlFor="name">{t("name")}</FieldLabel>
              <Input id="name" name="name" required autoComplete="off" />
            </Field>
            {err ? <p className="text-sm text-destructive">{err}</p> : null}
            <Button type="submit">{t("create")}</Button>
          </FieldGroup>
        </form>
      </PopoverContent>
    </Popover>
  )
}

function AddRepoMenu({ team, onAdd }: { team: string; onAdd: () => Promise<void> }) {
  const t = useT()
  async function submit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault()
    const fd = new FormData(e.currentTarget)
    const repo = String(fd.get("repo") || "").trim()
    if (!repo) return
    await api.addTeamRepo(team, repo)
    e.currentTarget.reset()
    await onAdd()
  }
  return (
    <Popover>
      <PopoverTrigger render={<Button type="button" size="sm" variant="outline" />}>{t("addRepo")}</PopoverTrigger>
      <PopoverContent className="w-72">
        <form onSubmit={(e) => void submit(e)}>
          <FieldGroup>
            <Field>
              <FieldLabel htmlFor="repo">{t("repo")}</FieldLabel>
              <Input id="repo" name="repo" required autoComplete="off" />
            </Field>
            <Button type="submit">{t("create")}</Button>
          </FieldGroup>
        </form>
      </PopoverContent>
    </Popover>
  )
}

export function AdminTeamsPage() {
  const t = useT()
  const nav = useNavigate()
  const list = usePage((q) => api.repoTeams(q), [])
  useEvents((ev) => {
    if (ev.type === "catalog.updated") void list.reload()
  })

  return (
    <PagedList
      list={list}
      emptyText={t("noTeams")}
      skeleton="table"
      header={
        <div className="flex flex-wrap items-start justify-end gap-3">
          <CreateTeamMenu onCreated={(name) => nav(`/admin/teams/${encodeURIComponent(name)}`)} />
        </div>
      }
    >
      {(items) => (
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>{t("team")}</TableHead>
              <TableHead>{t("repos")}</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {items.map((row) => (
              <TableRow key={row.team}>
                <TableCell>
                  <Link className="hover:underline" to={`/admin/teams/${encodeURIComponent(row.team)}`}>
                    {row.team}
                  </Link>
                </TableCell>
                <TableCell>{row.count}</TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      )}
    </PagedList>
  )
}

export function AdminTeamPage() {
  const t = useT()
  const nav = useNavigate()
  const { name: teamParam = "" } = useParams()
  const team = decodeURIComponent(teamParam)
  const load = useLoad(() => api.team(team), [team])
  useEvents((ev) => {
    if (ev.type === "catalog.updated") void load.reload()
  })
  const data = load.data
  const manage = Boolean(data?.can_manage)

  return (
    <PageFrame
      loading={load.loading && !data}
      error={load.error}
      header={
        <div className="flex flex-wrap items-start justify-between gap-3">
          <div className="min-w-0">
            <Link className="text-sm text-muted-foreground hover:underline" to="/admin/teams">
              {t("teams")}
            </Link>
            <h1 className="text-base font-medium">{team}</h1>
          </div>
          {manage ? (
            <div className="flex flex-wrap items-center gap-2">
              <AddPersonMenu
                title={t("addMember")}
                exclude={(data?.members ?? []).map((m) => m.login)}
                onAdd={async (login, permission) => {
                  await api.setTeamMember(team, login, permission)
                  await load.reload()
                }}
              />
              <AddRepoMenu team={team} onAdd={() => load.reload()} />
              <Button
                type="button"
                size="sm"
                variant="outline"
                onClick={() => {
                  void api.deleteTeam(team).then(() => nav("/admin/teams"))
                }}
              >
                {t("deleteTeam")}
              </Button>
            </div>
          ) : null}
        </div>
      }
    >
      {data ? (
        <div className="flex flex-col gap-8">
          <section className="flex flex-col gap-3">
            <h2 className="text-sm font-medium">{t("repos")}</h2>
            {data.repos.length ? (
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>{t("repo")}</TableHead>
                    <TableHead>{t("defaultBranch")}</TableHead>
                    {manage ? <TableHead /> : null}
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {data.repos.map((r) => {
                    const { owner, name } = splitRepo(r.full_name || r.name)
                    return (
                      <TableRow key={r.full_name || r.name}>
                        <TableCell>
                          <Link className="hover:underline" to={`/repos/${owner}/${name}`}>
                            {name}
                          </Link>
                        </TableCell>
                        <TableCell>{r.default_branch || "dev"}</TableCell>
                        {manage ? (
                          <TableCell className="text-right">
                            <Button
                              type="button"
                              size="sm"
                              variant="outline"
                              onClick={() => void api.removeTeamRepo(team, name).then(() => load.reload())}
                            >
                              {t("remove")}
                            </Button>
                          </TableCell>
                        ) : null}
                      </TableRow>
                    )
                  })}
                </TableBody>
              </Table>
            ) : (
              <p className="text-sm text-muted-foreground">{t("noRepos")}</p>
            )}
          </section>
          <section className="flex flex-col gap-3">
            <h2 className="text-sm font-medium">{t("members")}</h2>
            {data.members.length ? (
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>{t("username")}</TableHead>
                    <TableHead>{t("permission")}</TableHead>
                    {manage ? <TableHead /> : null}
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {data.members.map((m) => (
                    <TableRow key={m.login}>
                      <TableCell>{m.login}</TableCell>
                      <TableCell>
                        {manage ? (
                          <PermSelect
                            value={m.permission}
                            onChange={(perm) => {
                              void api.setTeamMember(team, m.login, perm).then(() => load.reload())
                            }}
                          />
                        ) : (
                          permLabel(t, m.permission)
                        )}
                      </TableCell>
                      {manage ? (
                        <TableCell className="text-right">
                          <Button
                            type="button"
                            size="sm"
                            variant="outline"
                            onClick={() => void api.removeTeamMember(team, m.login).then(() => load.reload())}
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
