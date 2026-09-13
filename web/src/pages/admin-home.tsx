import { Link } from "react-router-dom"

import { CopyField } from "@/components/copy-field"
import { PageFrame } from "@/components/page-frame"
import { StatusBadge } from "@/components/status-badge"
import { Card, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { useLoad } from "@/hooks/use-load"
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
  const { data, error, loading } = useLoad(async () => {
    const [u, s] = await Promise.all([api.users(), api.stack()])
    return { users: u.users || [], stack: s }
  }, [])

  const users = data?.users || []
  const stack = data?.stack
  const bots = users.filter((u) => u.login.startsWith("agent-")).length
  const people = users.length - bots
  const agents = stack?.agents || []

  return (
    <PageFrame
      loading={loading && !data}
      error={error}
      className="gap-6"
      header={<p className="text-sm text-muted-foreground">{t("adminHomeDesc")}</p>}
    >
      {data ? (
        <>
          <div className="grid gap-4 @xl/main:grid-cols-3">
            <Card>
              <CardHeader>
                <CardDescription>{t("usersCount")}</CardDescription>
                <CardTitle className="text-2xl tabular-nums">{users.length}</CardTitle>
              </CardHeader>
            </Card>
            <Card>
              <CardHeader>
                <CardDescription>{t("peopleCount")}</CardDescription>
                <CardTitle className="text-2xl tabular-nums">{people}</CardTitle>
              </CardHeader>
            </Card>
            <Card>
              <CardHeader>
                <CardDescription>{t("botsCount")}</CardDescription>
                <CardTitle className="text-2xl tabular-nums">{bots}</CardTitle>
              </CardHeader>
            </Card>
          </div>

          {stack ? (
            <section className="flex flex-col gap-2">
              <h2 className="text-sm font-medium">{t("stack")}</h2>
              <p className="text-sm">
                Acahti {stack.version} · Forgejo {stack.forgejo} · Woodpecker {stack.woodpecker} · Caddy{" "}
                {stack.caddy} · Postgres {stack.postgres}
              </p>
              <CopyField label={t("upgradeHint")} value={stack.upgrade_hint} />
            </section>
          ) : null}

          <section className="flex flex-col gap-2">
            <h2 className="text-sm font-medium">{t("agents")}</h2>
            <p className="text-sm text-muted-foreground">{t("runnersNote")}</p>
            {agents.length === 0 ? (
              <p className="text-sm text-muted-foreground">{t("noAgents")}</p>
            ) : (
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
                  {agents.map((a) => (
                    <TableRow key={a.name}>
                      <TableCell>{a.name}</TableCell>
                      <TableCell>{fmtWhen(a.last_contact)}</TableCell>
                      <TableCell>
                        <StatusBadge status={agentStale(a) ? "stale" : "online"} />
                      </TableCell>
                      <TableCell className="font-mono text-xs">
                        {a.version || "—"}
                        {stack && a.version && !pinMatch(a.version, stack.woodpecker) ? (
                          <span className="ml-2 text-destructive">{t("runnerBehind")}</span>
                        ) : null}
                      </TableCell>
                      <TableCell className="font-mono text-xs">{labelText(a.labels)}</TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            )}
          </section>

          <Link className="text-sm underline-offset-4 hover:underline" to="/admin/users">
            {t("users")}
          </Link>
        </>
      ) : null}
    </PageFrame>
  )
}
