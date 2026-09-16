import { useState } from "react"
import { Navigate, useLocation, useSearchParams } from "react-router-dom"
import { toast } from "sonner"

import { BootScreen } from "@/components/boot"
import { AcahtiMark } from "@/components/logo"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { useT } from "@/i18n/i18n"
import { api } from "@/lib/api"
import { useSession } from "@/lib/session"

export function ConsentPage() {
  const t = useT()
  const loc = useLocation()
  const [params] = useSearchParams()
  const { me, ready } = useSession()
  const [pending, setPending] = useState(false)

  if (!ready) {
    return <BootScreen label={t("loading")} />
  }
  if (!me) {
    return <Navigate to={`/login?next=${encodeURIComponent(loc.pathname + loc.search)}`} replace />
  }

  async function approve() {
    setPending(true)
    try {
      const r = await api.oauthApprove({
        client_id: params.get("client_id") || "",
        redirect_uri: params.get("redirect_uri") || "",
        state: params.get("state") || "",
        code_challenge: params.get("code_challenge") || "",
      })
      window.location.href = r.redirect
    } catch (err) {
      toast.error(err instanceof Error ? err.message : t("loadError"))
      setPending(false)
    }
  }

  return (
    <div className="flex min-h-svh flex-col items-center justify-center gap-6 bg-muted p-6">
      <div className="flex items-center gap-2 font-medium">
        <AcahtiMark className="size-6" />
        {t("brand")}
      </div>
      <Card className="w-full max-w-sm">
        <CardHeader>
          <CardTitle>{t("consentTitle")}</CardTitle>
          <CardDescription>{t("consentDesc")}</CardDescription>
        </CardHeader>
        <CardContent className="flex flex-col gap-3">
          <Button disabled={pending} onClick={() => void approve()}>
            {t("consentAllow")}
          </Button>
        </CardContent>
      </Card>
    </div>
  )
}
