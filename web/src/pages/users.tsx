import { type FormEvent, useEffect, useState } from "react"
import { Link } from "react-router-dom"
import { ShieldIcon } from "lucide-react"
import { toast } from "sonner"

import { PermSelect, type AccessPerm } from "@/components/access-fields"
import { CopyField } from "@/components/copy-field"
import { MenuPanel } from "@/components/menu-panel"
import { PagedList } from "@/components/paged-list"
import { Button } from "@/components/ui/button"
import { Field, FieldGroup, FieldLabel } from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Switch } from "@/components/ui/switch"
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { useLoad } from "@/hooks/use-load"
import { usePage } from "@/hooks/use-page"
import { useT } from "@/i18n/i18n"
import { api, repoName, splitRepo, type Invite, type UserRepoPerm } from "@/lib/api"
import { onboardNote } from "@/lib/onboard"
import { useSession } from "@/lib/session"

async function copyText(text: string) {
  try {
    await navigator.clipboard.writeText(text)
  } catch {
    /* note stays in the menu for a manual copy */
  }
}

function AuthorInput({
  login,
  value,
  onSaved,
}: {
  login: string
  value: string
  onSaved: () => Promise<void>
}) {
  const t = useT()
  const [text, setText] = useState(value)

  useEffect(() => {
    setText(value)
  }, [value])

  async function save() {
    const next = text.trim() || login
    if (next === value) {
      setText(value)
      return
    }
    try {
      await api.setGitName(login, next)
      toast.success(t("gitNameSaved"))
      await onSaved()
    } catch (err) {
      setText(value)
      toast.error(err instanceof Error ? err.message : t("gitNameFailed"))
    }
  }

  return (
    <Input
      value={text}
      placeholder={login}
      autoComplete="off"
      aria-label={t("gitName")}
      onChange={(e) => setText(e.target.value)}
      onBlur={() => void save()}
      onKeyDown={(e) => {
        if (e.key === "Enter") (e.target as HTMLInputElement).blur()
      }}
    />
  )
}

function PasswordInput({ value, onChange }: { value: string; onChange: (v: string) => void }) {
  return (
    <Input
      type="password"
      autoComplete="new-password"
      value={value}
      onChange={(e) => onChange(e.target.value)}
    />
  )
}

function InviteMenu({
  invites,
  joinBase,
  error,
  onCreate,
  onRevoke,
}: {
  invites: Invite[]
  joinBase: string
  error?: string
  onCreate: () => Promise<void>
  onRevoke: (code: string) => Promise<void>
}) {
  const t = useT()
  return (
    <Popover>
      <PopoverTrigger render={<Button type="button" variant="outline" size="sm" />}>
        {t("inviteCreate")}
      </PopoverTrigger>
      <PopoverContent className="flex w-96 flex-col gap-3">
        <p className="text-sm text-muted-foreground">{t("invitesDesc")}</p>
        {error ? <p className="text-sm text-destructive">{error}</p> : null}
        <Button
          type="button"
          size="sm"
          onClick={() => {
            void onCreate()
          }}
        >
          {t("inviteCreate")}
        </Button>
        {invites.length ? (
          invites.map((inv) => (
            <div key={inv.code} className="flex flex-col gap-2">
              <CopyField label={t("invite")} value={`${joinBase}?code=${inv.code}`} />
              <Button type="button" size="sm" variant="outline" onClick={() => void onRevoke(inv.code)}>
                {t("revoke")}
              </Button>
            </div>
          ))
        ) : (
          <p className="text-sm text-muted-foreground">{t("noInvites")}</p>
        )}
      </PopoverContent>
    </Popover>
  )
}

function CreateUserMenu({
  admin,
  setAdmin,
  root,
  onCreate,
}: {
  admin: boolean
  setAdmin: (v: boolean) => void
  root: string
  onCreate: (login: string, password: string, admin: boolean) => Promise<string>
}) {
  const t = useT()
  const [note, setNote] = useState("")
  const [err, setErr] = useState("")

  return (
    <Popover
      onOpenChange={(open) => {
        if (!open) {
          setNote("")
          setErr("")
        }
      }}
    >
      <PopoverTrigger render={<Button type="button" size="sm" />}>{t("createUser")}</PopoverTrigger>
      <PopoverContent className="flex w-96 flex-col gap-3">
        <form
          onSubmit={(e: FormEvent<HTMLFormElement>) => {
            e.preventDefault()
            const fd = new FormData(e.currentTarget)
            const login = String(fd.get("username") || "").trim()
            const password = String(fd.get("password") || "")
            setErr("")
            const form = e.currentTarget
            void onCreate(login, password, admin)
              .then((created) => {
                const text = onboardNote(root, created, password)
                setNote(text)
                void copyText(text)
                form.reset()
                setAdmin(false)
              })
              .catch((e: unknown) => {
                setErr(e instanceof Error ? e.message : t("loadError"))
              })
          }}
        >
          <FieldGroup>
            <Field>
              <FieldLabel htmlFor="username">{t("username")}</FieldLabel>
              <Input id="username" name="username" required autoComplete="off" />
            </Field>
            <Field>
              <FieldLabel htmlFor="password">{t("password")}</FieldLabel>
              <Input id="password" name="password" type="password" required autoComplete="new-password" />
            </Field>
            <div className="flex w-fit items-center gap-2">
              <Switch id="admin" checked={admin} onCheckedChange={(v) => setAdmin(v === true)} />
              <Label htmlFor="admin">{t("admin")}</Label>
            </div>
            <Button type="submit">{t("create")}</Button>
          </FieldGroup>
        </form>
        {err ? <p className="text-sm text-destructive">{err}</p> : null}
        {note ? (
          <>
            <p className="text-sm text-muted-foreground">{t("onboardHint")}</p>
            <CopyField multiline value={note} />
          </>
        ) : null}
      </PopoverContent>
    </Popover>
  )
}

function UserTeamsMenu({ teams }: { teams: string[] }) {
  const t = useT()
  if (!teams.length) {
    return <span className="text-sm text-muted-foreground">—</span>
  }
  return (
    <Popover>
      <PopoverTrigger render={<Button type="button" size="sm" variant="outline" />}>
        {t("teamsCount", { n: teams.length })}
      </PopoverTrigger>
      <PopoverContent align="start" className="flex max-h-80 w-56 flex-col gap-1 overflow-y-auto">
        {teams.map((name) => (
          <Link
            key={name}
            to={`/admin/teams/${encodeURIComponent(name)}`}
            className="rounded-md px-2 py-1.5 text-sm hover:bg-muted"
          >
            {name}
          </Link>
        ))}
      </PopoverContent>
    </Popover>
  )
}

function repoViaLabel(t: ReturnType<typeof useT>, row: UserRepoPerm) {
  if (row.direct && row.team) return t("viaDirectOverTeam", { team: row.team })
  if (row.direct) return t("viaDirectGrant")
  if (row.team) return t("viaTeam", { team: row.team })
  return t("viaDirectGrant")
}

function UserReposMenu({
  login,
  isAdmin,
  repos: initial,
  onChanged,
}: {
  login: string
  isAdmin: boolean
  repos: UserRepoPerm[]
  onChanged: () => Promise<void>
}) {
  const t = useT()
  const [open, setOpen] = useState(false)
  const [rows, setRows] = useState(initial)
  const [err, setErr] = useState("")

  useEffect(() => {
    setRows(initial)
  }, [initial])

  async function refresh() {
    const r = await api.userRepos(login)
    setRows(r.repos || [])
    await onChanged()
  }

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger render={<Button type="button" size="sm" variant="outline" />}>
        {t("userRepos")}
      </PopoverTrigger>
      <PopoverContent
        align="start"
        className="flex max-h-80 w-max min-w-48 max-w-[min(22rem,calc(100vw-2rem))] flex-col gap-2 overflow-y-auto"
      >
        {isAdmin ? <p className="text-sm text-muted-foreground">{t("orgAdminRepos")}</p> : null}
        <MenuPanel error={err} empty={!rows.length && !isAdmin ? t("noUserRepos") : undefined}>
          {rows.length ? (
            <ul className="flex flex-col gap-1.5">
              {rows.map((row) => {
                const { owner, name } = splitRepo(row.repo)
                const direct = Boolean(row.direct)
                return (
                  <li key={row.repo} className="flex items-center gap-2 text-sm">
                    <Link className="min-w-0 truncate hover:underline" to={`/repos/${owner}/${name}`}>
                      {repoName(row.repo)}
                    </Link>
                    <span className="shrink-0 text-muted-foreground">{repoViaLabel(t, row)}</span>
                    <PermSelect
                      value={row.permission}
                      className="w-24"
                      onChange={(perm: AccessPerm) => {
                        setErr("")
                        void api
                          .setCollaborator(owner, name, login, perm)
                          .then(() => refresh())
                          .catch((e) => setErr(e instanceof Error ? e.message : t("loadError")))
                      }}
                    />
                    <Button
                      type="button"
                      size="sm"
                      variant="outline"
                      disabled={!direct}
                      title={!direct ? t("inheritRemoveHint") : undefined}
                      onClick={() => {
                        if (!direct) return
                        setErr("")
                        void api
                          .removeCollaborator(owner, name, login)
                          .then(() => refresh())
                          .catch((e) => setErr(e instanceof Error ? e.message : t("loadError")))
                      }}
                    >
                      {t("remove")}
                    </Button>
                  </li>
                )
              })}
            </ul>
          ) : null}
        </MenuPanel>
      </PopoverContent>
    </Popover>
  )
}

function OnboardMenu({
  login,
  root,
  password,
}: {
  login: string
  root: string
  password: string
}) {
  const t = useT()
  const [pw, setPw] = useState("")
  const [err, setErr] = useState("")

  async function prepare() {
    setErr("")
    const next = password.trim()
    if (!next) {
      setErr(t("passwordRequired"))
      return
    }
    setPw(next)
    await copyText(onboardNote(root, login, next))
  }

  return (
    <Popover
      onOpenChange={(open) => {
        if (open) void prepare()
        else {
          setPw("")
          setErr("")
        }
      }}
    >
      <PopoverTrigger render={<Button type="button" size="sm" disabled={!root} />}>
        {t("join")}
      </PopoverTrigger>
      <PopoverContent className="flex w-96 flex-col gap-3">
        {err ? <p className="text-sm text-destructive">{err}</p> : null}
        {pw ? (
          <>
            <p className="text-sm text-muted-foreground">{t("onboardHint")}</p>
            <CopyField multiline value={onboardNote(root, login, pw)} />
          </>
        ) : null}
      </PopoverContent>
    </Popover>
  )
}

export function UsersPage() {
  const t = useT()
  const { me } = useSession()
  const root = (me?.root_url || "").replace(/\/$/, "")
  const usersLoad = usePage((q) => api.users(q), [])
  const invitesLoad = useLoad(async () => {
    const r = await api.invites()
    return { invites: r.invites || [], join: r.join }
  }, [])
  const [formErr, setFormErr] = useState("")
  const [admin, setAdmin] = useState(false)
  const [resets, setResets] = useState<Record<string, string>>({})
  const users = usersLoad.items
  const invites = invitesLoad.data?.invites || []
  const joinBase = invitesLoad.data?.join || "/join"

  async function onCreate(login: string, password: string, isAdmin: boolean) {
    const r = await api.createUser(login, password, isAdmin)
    setResets((m) => ({ ...m, [r.user.login]: password }))
    await usersLoad.reload()
    return r.user.login
  }

  async function resetRow(login: string) {
    const pw = resets[login]?.trim() || ""
    if (!pw) {
      toast.error(t("passwordRequired"))
      return
    }
    try {
      await api.setPassword(login, pw)
      toast.success(t("resetPasswordSuccess"))
    } catch (err) {
      toast.error(err instanceof Error ? err.message : t("resetPasswordFailed"))
    }
  }

  return (
    <PagedList
      list={{ ...usersLoad, error: usersLoad.error || formErr }}
      className="gap-6"
      skeleton="table"
      header={
        <div className="flex flex-wrap items-start justify-end gap-3">
          <div className="flex flex-wrap items-center gap-2">
            <InviteMenu
              invites={invites}
              joinBase={joinBase}
              error={invitesLoad.error ? t("invitesUnavailable") : undefined}
              onCreate={async () => {
                setFormErr("")
                try {
                  await api.createInvite()
                  await invitesLoad.reload()
                } catch (err) {
                  setFormErr(err instanceof Error ? err.message : t("loadError"))
                }
              }}
              onRevoke={async (code) => {
                await api.deleteInvite(code)
                await invitesLoad.reload()
              }}
            />
            <CreateUserMenu admin={admin} setAdmin={setAdmin} root={root} onCreate={onCreate} />
          </div>
        </div>
      }
    >
      {() => (
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>{t("username")}</TableHead>
              <TableHead>{t("gitName")}</TableHead>
              <TableHead>{t("team")}</TableHead>
              <TableHead>{t("repos")}</TableHead>
              <TableHead>{t("admin")}</TableHead>
              <TableHead>{t("password")}</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {users.map((u) => {
              const teams = u.teams || []
              return (
                <TableRow key={u.login}>
                  <TableCell>{u.login}</TableCell>
                  <TableCell>
                    <AuthorInput
                      login={u.login}
                      value={u.full_name || u.login}
                      onSaved={async () => {
                        usersLoad.reload()
                      }}
                    />
                  </TableCell>
                  <TableCell>
                    <UserTeamsMenu teams={teams} />
                  </TableCell>
                  <TableCell>
                    <UserReposMenu
                      login={u.login}
                      isAdmin={u.is_admin}
                      repos={u.repos || []}
                      onChanged={async () => {
                        usersLoad.reload()
                      }}
                    />
                  </TableCell>
                  <TableCell>
                    {u.is_admin ? (
                      <ShieldIcon className="size-4 text-muted-foreground" aria-label={t("admin")} />
                    ) : null}
                  </TableCell>
                  <TableCell>
                    <div className="flex items-center gap-2">
                      <PasswordInput
                        value={resets[u.login] || ""}
                        onChange={(v) => setResets((m) => ({ ...m, [u.login]: v }))}
                      />
                      <Button type="button" size="sm" variant="outline" onClick={() => void resetRow(u.login)}>
                        {t("resetPassword")}
                      </Button>
                      <OnboardMenu
                        login={u.login}
                        root={root}
                        password={resets[u.login] || ""}
                      />
                    </div>
                  </TableCell>
                </TableRow>
              )
            })}
          </TableBody>
        </Table>
      )}
    </PagedList>
  )
}
