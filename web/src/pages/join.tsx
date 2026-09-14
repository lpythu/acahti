import { type FormEvent, useState } from "react"
import { Link, useSearchParams } from "react-router-dom"

import { AcahtiMark } from "@/components/logo"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Field, FieldDescription, FieldGroup, FieldLabel } from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { useT } from "@/i18n/i18n"
import { api } from "@/lib/api"
import { useSession } from "@/lib/session"

export function JoinPage() {
  const t = useT()
  const [params] = useSearchParams()
  const { refresh } = useSession()
  const [error, setError] = useState("")
  const [pending, setPending] = useState(false)

  async function onSubmit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault()
    const fd = new FormData(e.currentTarget)
    setPending(true)
    setError("")
    try {
      await api.join(String(fd.get("username") || ""), String(fd.get("password") || ""), String(fd.get("code") || ""))
      await refresh()
      const next = params.get("next") || "/board"
      window.location.replace(next.startsWith("/") ? next : "/board")
    } catch (err) {
      setError(err instanceof Error ? err.message : t("joinError"))
    } finally {
      setPending(false)
    }
  }

  return (
    <div className="flex min-h-svh flex-col items-center justify-center gap-6 bg-muted p-6">
      <Link to="/" className="flex items-center gap-2 font-medium">
        <AcahtiMark className="size-6" />
        {t("brand")}
      </Link>
      <Card className="w-full max-w-sm">
        <CardHeader className="text-center">
          <CardTitle>{t("joinTitle")}</CardTitle>
          <CardDescription>{t("joinDesc")}</CardDescription>
        </CardHeader>
        <CardContent>
          <form onSubmit={onSubmit}>
            <FieldGroup>
              <Field>
                <FieldLabel htmlFor="code">{t("invite")}</FieldLabel>
                <Input id="code" name="code" required defaultValue={params.get("code") || ""} autoComplete="off" />
              </Field>
              <Field>
                <FieldLabel htmlFor="username">{t("username")}</FieldLabel>
                <Input id="username" name="username" required autoComplete="username" />
              </Field>
              <Field>
                <FieldLabel htmlFor="password">{t("password")}</FieldLabel>
                <Input id="password" name="password" type="password" required autoComplete="new-password" />
              </Field>
              {error ? <FieldDescription className="text-destructive">{error}</FieldDescription> : null}
              <Field>
                <Button type="submit" disabled={pending}>
                  {t("join")}
                </Button>
                <FieldDescription>
                  <Link to={params.get("next") ? `/login?next=${encodeURIComponent(params.get("next") || "")}` : "/login"}>
                    {t("login")}
                  </Link>
                </FieldDescription>
              </Field>
            </FieldGroup>
          </form>
        </CardContent>
      </Card>
    </div>
  )
}
