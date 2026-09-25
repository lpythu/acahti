import { Link, Navigate } from "react-router-dom"

import { BootScreen } from "@/components/boot"
import { AcahtiMark } from "@/components/logo"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { CopyField } from "@/components/copy-field"
import { useLoad } from "@/hooks/use-load"
import { useT } from "@/i18n/i18n"
import { api } from "@/lib/api"
import { useSession } from "@/lib/session"

/** Public product site (GitHub Pages). */
export const PRODUCT_SITE = "https://lpythu.github.io/acahti/"

export function UsePage() {
  const t = useT()
  const { me, ready } = useSession()
  const { data } = useLoad(() => api.public(), [])

  if (!ready) {
    return <BootScreen label={t("loading")} />
  }
  if (me) {
    return <Navigate to="/board" replace />
  }

  const skill = data?.skill || "/skill.md"
  const demo = data?.demo || "/demo.md"
  const join = data?.join || "/join"
  const org = data?.org || "org"
  const agentPrompt =
    data?.agent_prompt ||
    `Install ${skill}\nInstall ${demo}\n\nRun the Acahti demo loop on this instance now. Connect MCP with OAuth. Do not paste tokens. Do not change global git config. Create or reuse a disposable demo-* repo under ${org}, push a trivial branch with a smoke pipeline, wait for checks, open a PR, merge when green (or stop with evidence if no Runner). Reply with repo URL, PR number, and pipeline number. I will only watch the Board.`

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
          <p className="text-sm text-muted-foreground">{t("usePositioning")}</p>
          <CopyField label={t("useInstall")} value={`Install ${skill}`} />
          <CopyField label={t("useDemo")} value={`Install ${demo}`} />
          <CopyField label={t("useAgentPrompt")} value={agentPrompt} multiline />
          <CopyField label={t("useJoin")} value={`Join    ${join}`} />
          <div className="flex flex-wrap gap-2">
            <Button render={<Link to="/join" />}>{t("join")}</Button>
            <Button variant="outline" render={<Link to="/login" />}>
              {t("login")}
            </Button>
            <Button variant="ghost" render={<a href={PRODUCT_SITE} target="_blank" rel="noreferrer" />}>
              {t("useProductSite")}
            </Button>
            <Button variant="ghost" render={<a href={`${PRODUCT_SITE}cloud.html`} target="_blank" rel="noreferrer" />}>
              {t("useCloudWaitlist")}
            </Button>
          </div>
        </CardContent>
      </Card>
    </div>
  )
}
