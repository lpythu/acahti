import { useState, type FormEvent } from "react"
import { Link, useSearchParams } from "react-router-dom"
import { cn } from "cn"

import { Button } from "@/components/ui/button"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import { Field, FieldDescription, FieldGroup, FieldLabel } from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { useT } from "@/i18n/i18n"
import { api } from "@/lib/api"
import { useSession } from "@/lib/session"

export function LoginForm({ className, ...props }: React.ComponentProps<"div">) {
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
      await api.login(String(fd.get("username") || ""), String(fd.get("password") || ""))
      await refresh()
      const next = params.get("next") || "/board"
      window.location.replace(next.startsWith("/") ? next : "/board")
    } catch {
      setError(t("loginError"))
    } finally {
      setPending(false)
    }
  }

  return (
    <div className={cn("flex flex-col gap-6", className)} {...props}>
      <Card>
        <CardHeader className="text-center">
          <CardTitle className="text-xl">{t("loginTitle")}</CardTitle>
          <CardDescription>{t("loginDesc")}</CardDescription>
        </CardHeader>
        <CardContent>
          <form onSubmit={onSubmit}>
            <FieldGroup>
              <Field>
                <FieldLabel htmlFor="username">{t("username")}</FieldLabel>
                <Input
                  id="username"
                  name="username"
                  autoComplete="username"
                  required
                />
              </Field>
              <Field>
                <FieldLabel htmlFor="password">{t("password")}</FieldLabel>
                <Input
                  id="password"
                  name="password"
                  type="password"
                  autoComplete="current-password"
                  required
                />
              </Field>
              {error ? (
                <FieldDescription className="text-destructive">{error}</FieldDescription>
              ) : null}
              <Field>
                <Button type="submit" disabled={pending}>
                  {t("login")}
                </Button>
                <FieldDescription>
                  <Link to={params.get("next") ? `/join?next=${encodeURIComponent(params.get("next") || "")}` : "/join"}>
                    {t("join")}
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
