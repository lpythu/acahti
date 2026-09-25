import { Link, Navigate } from "react-router-dom"
import { toast } from "sonner"

import { BootScreen } from "@/components/boot"
import { AcahtiMark } from "@/components/logo"
import { Button } from "@/components/ui/button"
import { useLoad } from "@/hooks/use-load"
import { useT } from "@/i18n/i18n"
import { api } from "@/lib/api"
import { useSession } from "@/lib/session"

async function copyText(value: string, ok: string, fail: string) {
  try {
    await navigator.clipboard.writeText(value)
    toast.success(ok)
  } catch {
    toast.error(fail)
  }
}

function CopyBlock({
  label,
  value,
  rows = 1,
}: {
  label: string
  value: string
  rows?: number
}) {
  const t = useT()
  return (
    <div className="landing-copy">
      <div className="landing-copy-head">
        <span>{label}</span>
        <button type="button" className="landing-copy-btn" onClick={() => void copyText(value, t("copied"), t("copyFailed"))}>
          {t("copy")}
        </button>
      </div>
      <pre className={rows > 1 ? "landing-pre multiline" : "landing-pre"}>{value}</pre>
    </div>
  )
}

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
  const mcp = data?.mcp || "/mcp"
  const root = data?.root_url || window.location.origin
  const org = data?.org || "org"
  const agentPrompt =
    data?.agent_prompt ||
    `Install ${skill}\nInstall ${demo}\n\nRun the Acahti demo loop on this instance now. Connect MCP with OAuth. Do not paste tokens. Do not change global git config. Create or reuse a disposable demo-* repo under ${org}, push a trivial branch with a smoke pipeline, wait for checks, open a PR, merge when green (or stop with evidence if no Runner). Reply with repo URL, PR number, and pipeline number. I will only watch the Board.`

  return (
    <div className="landing">
      <div className="landing-glow" aria-hidden />
      <header className="landing-nav">
        <a href="/" className="landing-brand">
          <AcahtiMark className="size-7" />
          <span>{t("brand")}</span>
        </a>
        <div className="landing-nav-actions">
          <Button variant="ghost" render={<Link to="/login" />}>
            {t("login")}
          </Button>
          <Button render={<Link to="/join" />}>{t("join")}</Button>
        </div>
      </header>

      <main className="landing-main">
        <section className="landing-hero">
          <p className="landing-eyebrow">{t("landingEyebrow")}</p>
          <h1 className="landing-title">{t("landingTitle")}</h1>
          <p className="landing-sub">{t("landingSub")}</p>
          <div className="landing-cta">
            <Button
              size="lg"
              onClick={() => void copyText(agentPrompt, t("demoCopied"), t("copyFailed"))}
            >
              {t("landingCopyDemo")}
            </Button>
            <Button size="lg" variant="outline" render={<Link to="/join" />}>
              {t("landingJoinWatch")}
            </Button>
          </div>
          <p className="landing-hint">{t("landingHint")}</p>
        </section>

        <section className="landing-visual" aria-label={t("landingVisualAlt")}>
          <img src="/delivery.svg" alt={t("landingVisualAlt")} className="landing-diagram" />
        </section>

        <section className="landing-split">
          <div>
            <h2>{t("landingAgentTitle")}</h2>
            <p>{t("landingAgentBody")}</p>
            <ul className="landing-list">
              <li>{t("landingAgent1")}</li>
              <li>{t("landingAgent2")}</li>
              <li>{t("landingAgent3")}</li>
            </ul>
          </div>
          <div>
            <h2>{t("landingHumanTitle")}</h2>
            <p>{t("landingHumanBody")}</p>
            <ul className="landing-list human">
              <li>{t("landingHuman1")}</li>
              <li>{t("landingHuman2")}</li>
              <li>{t("landingHuman3")}</li>
            </ul>
          </div>
        </section>

        <section className="landing-actions">
          <h2>{t("landingStartTitle")}</h2>
          <p>{t("landingStartBody")}</p>
          <div className="landing-copies">
            <CopyBlock label={t("useInstall")} value={`Install ${skill}`} />
            <CopyBlock label={t("useDemo")} value={`Install ${demo}`} />
            <CopyBlock label={t("useAgentPrompt")} value={agentPrompt} rows={8} />
            <CopyBlock label={t("useJoin")} value={`Join    ${join}`} />
            <CopyBlock label={t("useMcp")} value={mcp} />
          </div>
          <p className="landing-meta">
            {root.replace(/\/$/, "")} · org <code>{org}</code>
          </p>
        </section>

        <section className="landing-diff">
          <h2>{t("landingDiffTitle")}</h2>
          <div className="landing-diff-grid">
            <div>
              <h3>GitHub / GitLab</h3>
              <p>{t("landingDiffGithub")}</p>
            </div>
            <div>
              <h3>Gitea / Forgejo</h3>
              <p>{t("landingDiffGitea")}</p>
            </div>
            <div className="accent">
              <h3>Acahti</h3>
              <p>{t("landingDiffAcahti")}</p>
            </div>
          </div>
        </section>

        <footer className="landing-foot">
          <p>{t("landingFoot")}</p>
          <a href="https://github.com/lpythu/acahti" target="_blank" rel="noreferrer">
            GitHub
          </a>
        </footer>
      </main>
    </div>
  )
}
