import { Link, Navigate } from "react-router-dom"

import { AcahtiMark } from "@/components/logo"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { CopyField } from "@/components/copy-field"
import { useLoad } from "@/hooks/use-load"
import { useT } from "@/i18n/i18n"
import { api } from "@/lib/api"
import { useSession } from "@/lib/session"

export function UsePage() {
  const t = useT()
  const { me, ready } = useSession()
  const { data } = useLoad(() => api.public(), [])

  if (!ready) {
    return <div className="bg-background min-h-svh" />
  }
  if (me) {
    return <Navigate to="/board" replace />
  }

  const skill = data?.skill || "/skill.md"
  const join = data?.join || "/join"

  return (
    <div className="flex min-h-svh flex-col items-center justify-center gap-6 bg-muted p-6">
      <a href="/" className="flex items-center gap-2 font-medium">
        <AcahtiMark className="size-6" />
        {t("brand")}
      </a>
      <Card className="w-full max-w-lg">
        <CardHeader>
          <CardTitle>{t("useTitle")}</CardTitle>
          <CardDescription>{t("useDesc")}</CardDescription>
        </CardHeader>
        <CardContent className="flex flex-col gap-4">
          <CopyField label={t("useInstall")} value={`Install ${skill}`} />
          <CopyField label={t("useJoin")} value={`Join    ${join}`} />
          <div className="flex gap-2">
            <Button render={<Link to="/join" />}>{t("join")}</Button>
            <Button variant="outline" render={<Link to="/login" />}>
              {t("login")}
            </Button>
          </div>
        </CardContent>
      </Card>
    </div>
  )
}
