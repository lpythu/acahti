import { Link } from "react-router-dom"

import { CopyField } from "@/components/copy-field"
import { Pager } from "@/components/paged-list"
import { PageFrame } from "@/components/page-frame"
import { StatusBadge } from "@/components/status-badge"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { useLoad } from "@/hooks/use-load"
import { usePage } from "@/hooks/use-page"
import { useT } from "@/i18n/i18n"
import { api, type Agent } from "@/lib/api"

function fmtWhen(unix: number) {
  if (!unix) return "—"
  return new Date(unix * 1000).toLocaleString()
}

function labelText(labels: unknown) {
  if (!labels) return ""
  if (typeof labels === "string") return labels
  try {
    return JSON.stringify(labels)
  } catch {
    return ""
  }
}

function agentStale(a: Agent) {
  if (!a.last_contact) return true
  return Date.now() / 1000 - a.last_contact > 5 * 60
}

function pinMatch(have: string | undefined, pin: string) {
  if (!have || !pin) return true
  return have.replace(/^v/, "") === pin.replace(/^v/, "")
}

export function AdminHomePage() {
  const t = useT()
  const stackLoad = useLoad(() => api.stack(), [])
  const agents = usePage((q) => api.agents(q), [])
  const stack = stackLoad.data
  const error = stackLoad.error || agents.error
  const loading = (stackLoad.loading && !stackLoad.data) && (agents.loading && !agents.data)

  return (
    <PageFrame
      loading={loading}
      error={error}
      className="gap-6"
    >
      {stack ? (
        <section className="flex flex-col gap-2">
          <h2 className="text-sm font-medium">{t("stack")}</h2>
          <p className="text-sm">
            Acahti {stack.version} · {t("git")} {stack.forgejo} · {t("ci")} {stack.ci} · Postgres {stack.postgres}
          </p>
          <CopyField label={t("upgradeHint")} value={stack.upgrade_hint} />
        </section>
      ) : null}

      <section className="flex flex-col gap-2">
        <h2 className="text-sm font-medium">{t("agents")}</h2>
        <p className="text-sm text-muted-foreground">{t("runnersNote")}</p>
        {agents.empty ? (
          <p className="text-sm text-muted-foreground">{t("noAgents")}</p>
        ) : (
          <>
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>{t("name")}</TableHead>
                  <TableHead>{t("lastSeen")}</TableHead>
                  <TableHead>{t("status")}</TableHead>
                  <TableHead>{t("runnerVersion")}</TableHead>
                  <TableHead>{t("labels")}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {agents.items.map((a) => (
                  <TableRow key={a.name}>
                    <TableCell>{a.name}</TableCell>
                    <TableCell>{fmtWhen(a.last_contact)}</TableCell>
                    <TableCell>
                      <StatusBadge status={agentStale(a) ? "stale" : "online"} />
                    </TableCell>
                    <TableCell className="font-mono text-xs">
                      {a.version || "—"}
                      {stack && a.version && !pinMatch(a.version, stack.ci) ? (
                        <span className="ml-2 text-destructive">{t("runnerBehind")}</span>
                      ) : null}
                    </TableCell>
                    <TableCell className="font-mono text-xs">{labelText(a.labels)}</TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
            <Pager page={agents.page} hasMore={agents.hasMore} onPage={agents.setPage} />
          </>
        )}
      </section>

      <div className="flex flex-wrap gap-4">
        <Link className="text-sm underline-offset-4 hover:underline" to="/admin/secrets">
          {t("adminSecrets")}
        </Link>
        <Link className="text-sm underline-offset-4 hover:underline" to="/admin/teams">
          {t("teams")}
        </Link>
        <Link className="text-sm underline-offset-4 hover:underline" to="/admin/users">
          {t("users")}
        </Link>
      </div>
    </PageFrame>
  )
}
