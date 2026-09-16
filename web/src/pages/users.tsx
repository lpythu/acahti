import { type FormEvent, useEffect, useState } from "react"
import { Link } from "react-router-dom"
import { CheckIcon, CopyIcon, ShieldIcon, Trash2Icon } from "lucide-react"
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

async function copyText(text: string, ok: string, fail: string) {
  try {
    await navigator.clipboard.writeText(text)
    toast.success(ok)
    return true
  } catch {
    toast.error(fail)
    return false
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

function PasswordInput({
  value,
  hasPassword,
  onChange,
}: {
  value: string
  hasPassword: boolean
  onChange: (v: string) => void
}) {
  const t = useT()
  const [copied, setCopied] = useState(false)
  const known = value.trim() !== ""
  const showMask = hasPassword && !known

  async function copy() {
    if (!known) return
    if (await copyText(value, t("copied"), t("copyFailed"))) {
      setCopied(true)
      window.setTimeout(() => setCopied(false), 1500)
    }
  }

  return (
    <div className="relative w-40 shrink-0">
      <Input
        type="password"
        autoComplete="new-password"
        aria-label={t("password")}
        value={value}
        className="w-40 pr-7"
        onChange={(e) => onChange(e.target.value)}
      />
      {showMask ? (
        <span
          aria-hidden
          className="pointer-events-none absolute inset-y-0 left-2.5 right-7 flex items-center text-sm tracking-[0.35em] text-foreground"
        >
          ••••••••
        </span>
      ) : null}
      {known ? (
        <button
          type="button"
          aria-label={t("copy")}
          className="absolute top-1/2 right-1.5 z-10 -translate-y-1/2 text-muted-foreground hover:text-foreground"
          onClick={() => void copy()}
        >
          {copied ? <CheckIcon className="size-3.5" /> : <CopyIcon className="size-3.5" />}
        </button>
      ) : null}
    </div>
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

  return (
    <Popover
      onOpenChange={(open) => {
        if (!open) setNote("")
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
            const form = e.currentTarget
            void onCreate(login, password, admin)
              .then((created) => {
                toast.success(t("userCreated"))
                const text = onboardNote(root, created, password)
                setNote(text)
                void copyText(text, t("copied"), t("copyFailed"))
                form.reset()
                setAdmin(false)
              })
              .catch((e: unknown) => {
                toast.error(e instanceof Error ? e.message : t("loadError"))
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
      <PopoverContent align="start" className="flex max-h-80 w-[min(24rem,calc(100vw-2rem))] flex-col gap-2 overflow-y-auto">
        {isAdmin ? <p className="text-sm text-muted-foreground">{t("orgAdminRepos")}</p> : null}
        <MenuPanel empty={!rows.length && !isAdmin ? t("noUserRepos") : undefined}>
          {rows.length ? (
            <ul className="flex flex-col gap-1.5">
              {rows.map((row) => {
                const { owner, name } = splitRepo(row.repo)
                const direct = Boolean(row.direct)
                const via = repoViaLabel(t, row)
                return (
                  <li
                    key={row.repo}
                    className="grid grid-cols-[minmax(0,1fr)_minmax(0,8rem)_6rem_1.5rem] items-center gap-x-2 text-sm"
                  >
                    <Link
                      className="min-w-0 truncate hover:underline"
                      title={repoName(row.repo)}
                      to={`/repos/${owner}/${name}`}
                    >
                      {repoName(row.repo)}
                    </Link>
                    <span className="min-w-0 truncate text-muted-foreground" title={via}>
                      {via}
                    </span>
                    <PermSelect
                      value={row.permission}
                      className="w-full"
                      onChange={(perm: AccessPerm) => {
                        void api
                          .setCollaborator(owner, name, login, perm)
                          .then(async () => {
                            toast.success(t("accessUpdated"))
                            await refresh()
                          })
                          .catch((e) => toast.error(e instanceof Error ? e.message : t("loadError")))
                      }}
                    />
                    <Button
                      type="button"
                      size="icon-xs"
                      variant="ghost"
                      disabled={!direct}
                      aria-label={t("remove")}
                      title={!direct ? t("inheritRemoveHint") : t("remove")}
                      className="text-destructive hover:bg-destructive/10 hover:text-destructive disabled:text-muted-foreground"
                      onClick={() => {
                        if (!direct) return
                        void api
                          .removeCollaborator(owner, name, login)
                          .then(async () => {
                            toast.success(t("accessRemoved"))
                            await refresh()
                          })
                          .catch((e) => toast.error(e instanceof Error ? e.message : t("loadError")))
                      }}
                    >
                      <Trash2Icon />
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
  onPassword,
}: {
  login: string
  root: string
  onPassword: (pw: string) => void
}) {
  const t = useT()
  const [note, setNote] = useState("")

  async function prepare() {
    try {
      const r = await api.ensurePassword(login)
      const next = r.password || ""
      if (next) onPassword(next)
      const text = onboardNote(root, login, next)
      setNote(text)
      await copyText(text, t("copied"), t("copyFailed"))
    } catch (err) {
      toast.error(err instanceof Error ? err.message : t("loadError"))
    }
  }

  return (
    <Popover
      onOpenChange={(open) => {
        if (open) void prepare()
        else setNote("")
      }}
    >
      <PopoverTrigger render={<Button type="button" size="sm" disabled={!root} />}>
        {t("join")}
      </PopoverTrigger>
      <PopoverContent className="flex w-96 flex-col gap-3">
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

export function UsersPage() {
  const t = useT()
  const { me } = useSession()
  const root = (me?.root_url || "").replace(/\/$/, "")
  const usersLoad = usePage((q) => api.users(q), [])
  const invitesLoad = useLoad(async () => {
    const r = await api.invites()
    return { invites: r.invites || [], join: r.join }
  }, [])
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

  async function resetRow(login: string, current: string) {
    const pw = (resets[login] ?? current).trim()
    if (!pw) {
      toast.error(t("passwordRequired"))
      return
    }
    try {
      const r = await api.setPassword(login, pw)
      setResets((m) => ({ ...m, [login]: r.password }))
      toast.success(t("resetPasswordSuccess"))
    } catch (err) {
      toast.error(err instanceof Error ? err.message : t("resetPasswordFailed"))
    }
  }

  return (
    <PagedList
      list={usersLoad}
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
                try {
                  await api.createInvite()
                  toast.success(t("inviteCreated"))
                  await invitesLoad.reload()
                } catch (err) {
                  toast.error(err instanceof Error ? err.message : t("loadError"))
                }
              }}
              onRevoke={async (code) => {
                try {
                  await api.deleteInvite(code)
                  toast.success(t("inviteRevoked"))
                  await invitesLoad.reload()
                } catch (err) {
                  toast.error(err instanceof Error ? err.message : t("loadError"))
                }
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
              const password = resets[u.login] ?? u.password ?? ""
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
                        value={password}
                        hasPassword={Boolean(u.has_password) || password.trim() !== ""}
                        onChange={(v) => setResets((m) => ({ ...m, [u.login]: v }))}
                      />
                      <Button type="button" size="sm" variant="outline" onClick={() => void resetRow(u.login, u.password || "")}>
                        {t("resetPassword")}
                      </Button>
                      <OnboardMenu
                        login={u.login}
                        root={root}
                        onPassword={(pw) => setResets((m) => ({ ...m, [u.login]: pw }))}
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
