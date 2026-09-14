import { type FormEvent, useState } from "react"
import { Link } from "react-router-dom"
import { toast } from "sonner"

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
import { api, type Invite, type User } from "@/lib/api"
import { onboardNote, randomPassword } from "@/lib/onboard"
import { useSession } from "@/lib/session"

async function teamsByLogin() {
  const map: Record<string, string[]> = {}
  let pageNum = 1
  for (;;) {
    const listed = await api.repoTeams({ page: pageNum, page_size: 50 })
    await Promise.all(
      (listed.items || []).map(async (row) => {
        const team = await api.team(row.team)
        for (const m of team.members || []) {
          const cur = map[m.login] || []
          if (!cur.includes(row.team)) cur.push(row.team)
          map[m.login] = cur
        }
      }),
    )
    if (!listed.has_more) break
    pageNum++
  }
  for (const login of Object.keys(map)) {
    map[login].sort((a, b) => a.localeCompare(b))
  }
  return map
}

function userTeams(u: User, fallback: Record<string, string[]> | null) {
  if (u.teams) return u.teams
  return fallback?.[u.login] || []
}

async function copyText(text: string) {
  try {
    await navigator.clipboard.writeText(text)
  } catch {
    /* note stays in the menu for a manual copy */
  }
}

const PASSWORD_MASK = "••••••••"

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
  const apiHasTeams = Boolean(usersLoad.data) && users.every((u) => u.teams !== undefined)
  const membership = useLoad(teamsByLogin, [], Boolean(usersLoad.data) && !apiHasTeams)
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
        <div className="flex flex-wrap items-start justify-between gap-3">
          <p className="text-sm text-muted-foreground">{t("usersDesc")}</p>
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
              <TableHead>{t("team")}</TableHead>
              <TableHead>{t("admin")}</TableHead>
              <TableHead>{t("password")}</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {users.map((u) => (
              <TableRow key={u.login}>
                <TableCell>{u.login}</TableCell>
                <TableCell>
                  <div className="flex flex-wrap gap-1.5">
                    {userTeams(u, membership.data).map((name) => (
                      <Link
                        key={name}
                        to={`/repos?team=${encodeURIComponent(name)}`}
                        className="inline-flex rounded-md bg-background px-2 py-0.5 text-xs shadow-sm ring-1 ring-foreground/10 hover:bg-muted"
                      >
                        {name}
                      </Link>
                    ))}
                  </div>
                </TableCell>
                <TableCell>{u.is_admin ? t("admin") : ""}</TableCell>
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
            ))}
          </TableBody>
        </Table>
      )}
    </PagedList>
  )
}
