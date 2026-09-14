import { type FormEvent, useState } from "react"
import { Link, useNavigate, useSearchParams } from "react-router-dom"

import { AddPersonMenu, PermSelect, permLabel } from "@/components/access-fields"
import { PageFrame } from "@/components/page-frame"
import { PagedList } from "@/components/paged-list"
import { Avatar, AvatarFallback } from "@/components/ui/avatar"
import { Button } from "@/components/ui/button"
import { Field, FieldGroup, FieldLabel } from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { useLoad } from "@/hooks/use-load"
import { usePage } from "@/hooks/use-page"
import { useT } from "@/i18n/i18n"
import { api, splitRepo } from "@/lib/api"
import { useSession } from "@/lib/session"

function CreateGroupMenu({ onCreated }: { onCreated: (name: string) => void }) {
  const t = useT()
  const [err, setErr] = useState("")
  async function submit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault()
    const fd = new FormData(e.currentTarget)
    const name = String(fd.get("name") || "").trim()
    if (!name) return
    setErr("")
    try {
      await api.createGroup(name)
      e.currentTarget.reset()
      onCreated(name)
    } catch (e) {
      setErr(e instanceof Error ? e.message : t("loadError"))
    }
  }
  return (
    <Popover>
      <PopoverTrigger render={<Button type="button" size="sm" />}>{t("createGroup")}</PopoverTrigger>
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

function AddRepoMenu({ group, onAdd }: { group: string; onAdd: () => Promise<void> }) {
  const t = useT()
  async function submit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault()
    const fd = new FormData(e.currentTarget)
    const repo = String(fd.get("repo") || "").trim()
    if (!repo) return
    await api.addGroupRepo(group, repo)
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

function GroupPage({ group }: { group: string }) {
  const t = useT()
  const nav = useNavigate()
  const load = useLoad(() => api.group(group), [group])
  const data = load.data
  const manage = Boolean(data?.can_manage)

  return (
    <PageFrame
      loading={load.loading && !data}
      error={load.error}
      header={
        <div className="flex flex-wrap items-start justify-between gap-3">
          <div>
            <h1 className="text-base font-medium">{group}</h1>
            <p className="text-sm text-muted-foreground">{t("groupAccessDesc")}</p>
          </div>
          {manage ? (
            <div className="flex flex-wrap items-center gap-2">
              <AddPersonMenu
                title={t("addMember")}
                onAdd={async (login, permission) => {
                  await api.setGroupMember(group, login, permission)
                  await load.reload()
                }}
              />
              <AddRepoMenu group={group} onAdd={() => load.reload()} />
              <Button
                type="button"
                size="sm"
                variant="outline"
                onClick={() => {
                  void api.deleteGroup(group).then(() => nav("/repos"))
                }}
              >
                {t("deleteGroup")}
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
                              onClick={() => void api.removeGroupRepo(group, name).then(() => load.reload())}
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
                              void api.setGroupMember(group, m.login, perm).then(() => load.reload())
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
                            onClick={() => void api.removeGroupMember(group, m.login).then(() => load.reload())}
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

export function ReposPage() {
  const t = useT()
  const nav = useNavigate()
  const { me } = useSession()
  const [sp] = useSearchParams()
  const group = sp.get("group") || ""
  const groups = usePage((q) => api.repoGroups(q), [], { enabled: !group })

  if (group) {
    return <GroupPage group={group} />
  }

  return (
    <PagedList
      list={groups}
      emptyText={t("noRepos")}
      header={
        <div className="flex flex-wrap items-start justify-between gap-3">
          <p className="text-sm text-muted-foreground">{t("reposDesc")}</p>
          {me?.admin ? <CreateGroupMenu onCreated={(name) => nav(`/repos?group=${encodeURIComponent(name)}`)} /> : null}
        </div>
      }
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
