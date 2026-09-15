import { type FormEvent, useEffect, useState } from "react"
import { Link } from "react-router-dom"
import { ShieldIcon } from "lucide-react"
import { toast } from "sonner"

import { permLabel } from "@/components/access-fields"
import { CopyField } from "@/components/copy-field"
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
import { api, repoName, splitRepo, type Invite } from "@/lib/api"
import { onboardNote, randomPassword } from "@/lib/onboard"
import { useSession } from "@/lib/session"

async function copyText(text: string) {
  try {
    await navigator.clipboard.writeText(text)
  } catch {
    /* note stays in the menu for a manual copy */
  }
}

const PASSWORD_MASK = "••••••••"

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
  const [focused, setFocused] = useState(false)
  return (
    <Input
      type="password"
      autoComplete="new-password"
      value={value || (focused ? "" : PASSWORD_MASK)}
      onFocus={() => setFocused(true)}
      onBlur={() => setFocused(false)}
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

function UserReposMenu({
  login,
  isAdmin,
}: {
  login: string
  isAdmin: boolean
}) {
  const t = useT()
  const [open, setOpen] = useState(false)
  const load = useLoad(() => api.userRepos(login).then((r) => r.repos || []), [login], open)
  const repos = load.data || []

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger render={<Button type="button" size="sm" variant="outline" />}>
        {t("userRepos")}
      </PopoverTrigger>
      <PopoverContent align="start" className="flex max-h-80 w-80 flex-col gap-2 overflow-y-auto">
        {load.error ? <p className="text-sm text-destructive">{load.error}</p> : null}
        {isAdmin ? <p className="text-sm text-muted-foreground">{t("orgAdminRepos")}</p> : null}
        {!load.loading && !load.error && !isAdmin && !repos.length ? (
          <p className="text-sm text-muted-foreground">{t("noUserRepos")}</p>
        ) : null}
        {repos.length ? (
          <ul className="flex flex-col gap-1.5">
            {repos.map((row) => {
              const { owner, name } = splitRepo(row.repo)
              return (
                <li key={row.repo} className="flex items-center justify-between gap-2 text-sm">
                  <Link className="min-w-0 truncate hover:underline" to={`/repos/${owner}/${name}`}>
                    {repoName(row.repo)}
                  </Link>
                  <span className="shrink-0 text-muted-foreground">{permLabel(t, row.permission)}</span>
                </li>
              )
            })}
          </ul>
        ) : null}
      </PopoverContent>
    </Popover>
  )
}

function OnboardMenu({
  login,
  root,
  password,
  onPassword,
}: {
  login: string
  root: string
  password: string
  onPassword: (pw: string) => void
}) {
  const t = useT()
  const [note, setNote] = useState("")
  const [err, setErr] = useState("")

  async function prepare() {
    setErr("")
    try {
      let pw = password.trim()
      if (!pw) {
        pw = randomPassword()
        onPassword(pw)
      }
      await api.setPassword(login, pw)
      const text = onboardNote(root, login, pw)
      setNote(text)
      await copyText(text)
    } catch (e) {
      setErr(e instanceof Error ? e.message : t("loadError"))
    }
  }

  return (
    <Popover
      onOpenChange={(open) => {
        if (open) void prepare()
        else {
          setNote("")
          setErr("")
        }
      }}
    >
      <PopoverTrigger render={<Button type="button" size="sm" disabled={!root} />}>
        {t("join")}
      </PopoverTrigger>
      <PopoverContent className="flex w-96 flex-col gap-3">
        <p className="text-sm text-muted-foreground">{t("onboardHint")}</p>
        {err ? <p className="text-sm text-destructive">{err}</p> : null}
        {note ? <CopyField multiline value={note} /> : null}
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
                    <UserReposMenu login={u.login} isAdmin={u.is_admin} />
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
