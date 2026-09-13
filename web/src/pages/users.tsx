import { type FormEvent, useState } from "react"

import { PageFrame } from "@/components/page-frame"
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert"
import { Button } from "@/components/ui/button"
import { Checkbox } from "@/components/ui/checkbox"
import { Field, FieldGroup, FieldLabel } from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { useLoad } from "@/hooks/use-load"
import { useT } from "@/i18n/i18n"
import { api } from "@/lib/api"

export function UsersPage() {
  const t = useT()
  const { data, error, loading, reload } = useLoad(async () => (await api.users()).users || [], [])
  const [notice, setNotice] = useState("")
  const [formErr, setFormErr] = useState("")
  const [admin, setAdmin] = useState(false)
  const [resets, setResets] = useState<Record<string, string>>({})
  const users = data || []

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
      await reload()
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
      loading={loading && !data}
      error={error || formErr}
      className="gap-6"
      skeleton="table"
      header={
        <>
          <p className="text-sm text-muted-foreground">{t("usersDesc")}</p>
          {notice ? (
            <Alert>
              <AlertTitle>{notice}</AlertTitle>
              <AlertDescription>{t("loginWithPassword")}</AlertDescription>
            </Alert>
          ) : null}
          <form className="max-w-lg" onSubmit={onCreate}>
            <FieldGroup>
              <Field>
                <FieldLabel htmlFor="username">{t("username")}</FieldLabel>
                <Input id="username" name="username" required autoComplete="off" />
              </Field>
              <Field>
                <FieldLabel htmlFor="password">{t("password")}</FieldLabel>
                <Input id="password" name="password" type="password" required autoComplete="new-password" />
              </Field>
              <Field className="flex-row items-center gap-2">
                <Checkbox id="admin" checked={admin} onCheckedChange={(v) => setAdmin(v === true)} />
                <FieldLabel htmlFor="admin">{t("admin")}</FieldLabel>
              </Field>
              <Button type="submit">{t("create")}</Button>
            </FieldGroup>
          </form>
        </>
      }
    >
      {users.length ? (
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
      ) : null}
    </PageFrame>
  )
}
