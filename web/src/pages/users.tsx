import { type FormEvent, useState } from "react"

import { CopyField } from "@/components/copy-field"
import { PageFrame } from "@/components/page-frame"
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert"
import { Button } from "@/components/ui/button"
import { Field, FieldGroup, FieldLabel } from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Switch } from "@/components/ui/switch"
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { useLoad } from "@/hooks/use-load"
import { useT } from "@/i18n/i18n"
import { api, type Invite } from "@/lib/api"

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
  onCreate,
}: {
  admin: boolean
  setAdmin: (v: boolean) => void
  onCreate: (e: FormEvent<HTMLFormElement>) => Promise<void>
}) {
  const t = useT()
  return (
    <Popover>
      <PopoverTrigger render={<Button type="button" size="sm" />}>
        {t("createUser")}
      </PopoverTrigger>
      <PopoverContent className="w-80">
        <form onSubmit={(e) => void onCreate(e)}>
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
      </PopoverContent>
    </Popover>
  )
}

export function UsersPage() {
  const t = useT()
  const usersLoad = useLoad(async () => (await api.users()).users || [], [])
  const invitesLoad = useLoad(async () => {
    const r = await api.invites()
    return { invites: r.invites || [], join: r.join }
  }, [])
  const [notice, setNotice] = useState("")
  const [formErr, setFormErr] = useState("")
  const [admin, setAdmin] = useState(false)
  const [resets, setResets] = useState<Record<string, string>>({})
  const users = usersLoad.data || []
  const invites = invitesLoad.data?.invites || []
  const joinBase = invitesLoad.data?.join || "/join"

  async function onCreate(e: FormEvent<HTMLFormElement>) {
    e.preventDefault()
    const fd = new FormData(e.currentTarget)
    setFormErr("")
    setNotice("")
    try {
      const r = await api.createUser(String(fd.get("username") || ""), String(fd.get("password") || ""), admin)
      setNotice(t("userCreated", { name: r.user.login }))
      e.currentTarget.reset()
      setAdmin(false)
      await usersLoad.reload()
    } catch (err) {
      setFormErr(err instanceof Error ? err.message : t("loadError"))
    }
  }

  async function resetRow(login: string) {
    const pw = resets[login] || ""
    if (!pw) return
    setFormErr("")
    setNotice("")
    try {
      await api.setPassword(login, pw)
      setNotice(t("passwordSet", { name: login }))
      setResets((m) => ({ ...m, [login]: "" }))
    } catch (err) {
      setFormErr(err instanceof Error ? err.message : t("loadError"))
    }
  }

  return (
    <PageFrame
      loading={usersLoad.loading && !usersLoad.data}
      error={usersLoad.error || formErr}
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
            <CreateUserMenu admin={admin} setAdmin={setAdmin} onCreate={onCreate} />
          </div>
        </div>
      }
    >
      {notice ? (
        <Alert>
          <AlertTitle>{notice}</AlertTitle>
          <AlertDescription>{t("loginWithPassword")}</AlertDescription>
        </Alert>
      ) : null}
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>{t("username")}</TableHead>
            <TableHead>{t("admin")}</TableHead>
            <TableHead>{t("password")}</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {users.map((u) => (
            <TableRow key={u.login}>
              <TableCell>{u.login}</TableCell>
              <TableCell>{u.is_admin ? t("admin") : ""}</TableCell>
              <TableCell>
                <div className="flex items-center gap-2">
                  <Input
                    type="password"
                    autoComplete="new-password"
                    value={resets[u.login] || ""}
                    onChange={(e) => setResets((m) => ({ ...m, [u.login]: e.target.value }))}
                  />
                  <Button type="button" size="sm" variant="outline" onClick={() => void resetRow(u.login)}>
                    {t("resetPassword")}
                  </Button>
                </div>
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </PageFrame>
  )
}
