import { useState } from "react"
import { Link, useSearchParams } from "react-router-dom"
import { toast } from "sonner"

import { BoardHeatmap } from "@/components/board-heatmap"
import { PagedList } from "@/components/paged-list"
import { StatusBadge } from "@/components/status-badge"
import { Button } from "@/components/ui/button"
import { useEvents } from "@/hooks/use-events"
import { usePage } from "@/hooks/use-page"
import { useLocale, useT } from "@/i18n/i18n"
import type { MessageKey } from "@/i18n/messages"
import { activityHref, activityKind } from "@/lib/activity"
import { api, repoName, splitRepo, type Activity, type PR } from "@/lib/api"
import { displayName } from "@/lib/user"

function heatDateLabel(date: string, locale: string) {
  return new Intl.DateTimeFormat(locale.startsWith("zh") ? "zh-CN" : "en-US", {
    dateStyle: "long",
    timeZone: "Asia/Shanghai",
  }).format(new Date(`${date}T00:00:00+08:00`))
}

function activityLabel(a: Activity, t: (key: MessageKey, vars?: Record<string, string | number>) => string) {
  const repo = repoName(a.repo?.full_name || a.repo?.name || "")
  switch (activityKind(a.op_type)) {
    case "push":
      return t("activityPush", { repo })
    case "pr":
      return t("activityPR", { repo })
    case "issue":
      return t("activityIssue", { repo })
    case "repo":
      return t("activityRepo", { repo })
    default:
      return t("activityOther", { repo, op: a.op_type || "—" })
  }
}

function ActivityRows({ items }: { items: Activity[] }) {
  const t = useT()
  return (
    <ul className="divide-y rounded-md border">
      {items.map((a) => (
        <li key={a.id}>
          <Link className="block px-3 py-3 text-sm hover:bg-muted/50" to={activityHref(a)}>
            <span className="font-medium">{activityLabel(a, t)}</span>
            {a.ref_name ? <span className="mt-0.5 block truncate text-xs text-muted-foreground">{a.ref_name}</span> : null}
          </Link>
        </li>
      ))}
    </ul>
  )
}

export function BoardPage() {
  const t = useT()
  const locale = useLocale()
  const [sp, setSp] = useSearchParams()
  const date = sp.get("date") || ""
  const prs = usePage((q) => api.boardPRs(q), [], { enabled: !date })
  const acts = usePage((q) => api.boardActivities({ ...q, date }), [date], { enabled: Boolean(date) })
  const [busy, setBusy] = useState("")

  useEvents((ev) => {
    if (ev.type !== "forgejo") return
    if (date) void acts.reload()
    else void prs.reload()
  })

  function setDate(next: string) {
    const q = new URLSearchParams(sp)
    if (!next || next === date) q.delete("date")
    else q.set("date", next)
    q.delete("page")
    setSp(q, { replace: true })
  }

  async function merge(pr: PR) {
    const { owner, name } = splitRepo(pr.repo)
    setBusy(`${pr.repo}-${pr.number}`)
    try {
      await api.mergePull(owner, name, pr.number)
      toast.success(t("mergeSuccess"))
      await prs.reload()
    } catch {
      toast.error(t("checksNotGreen"))
    } finally {
      setBusy("")
    }
  }

  const header = (
    <>
      <BoardHeatmap selected={date} onSelect={setDate} />
      {date ? <h2 className="text-sm font-medium">{t("heatDayTitle", { date: heatDateLabel(date, locale) })}</h2> : null}
    </>
  )

  if (date) {
    return (
      <PagedList list={acts} emptyText={t("noActivities")} skeleton="lines" header={header}>
        {(items) => <ActivityRows items={items} />}
      </PagedList>
    )
  }

  return (
    <PagedList list={prs} emptyText={t("boardEmpty")} skeleton="lines" header={header}>
      {(items) => (
        <ul className="divide-y rounded-md border">
          {items.map((pr) => {
            const { owner, name } = splitRepo(pr.repo)
            const author = displayName(pr.user)
            const head = pr.head?.ref || ""
            const base = pr.base?.ref || ""
            return (
              <li key={`${pr.repo}-${pr.number}`} className="flex items-start gap-3 px-3 py-3">
                <StatusBadge status={pr.merged ? "merged" : pr.state || "open"} />
                <div className="min-w-0 flex-1">
                  <div className="flex flex-wrap items-baseline gap-x-2 gap-y-0.5">
                    <Link
                      className="truncate text-sm font-medium hover:underline"
                      to={`/repos/${owner}/${name}/pulls/${pr.number}`}
                    >
                      {pr.title}
                    </Link>
                    <span className="text-xs text-muted-foreground">
                      {repoName(pr.repo)} #{pr.number}
                    </span>
                  </div>
                  {(author || (head && base)) && (
                    <p className="mt-0.5 truncate text-xs text-muted-foreground">
                      {author}
                      {author && head && base ? " · " : ""}
                      {head && base ? `${head} → ${base}` : ""}
                    </p>
                  )}
                </div>
                <Button
                  type="button"
                  size="sm"
                  disabled={busy === `${pr.repo}-${pr.number}` || Boolean(pr.merged)}
                  onClick={() => void merge(pr)}
                >
                  {t("merge")}
                </Button>
              </li>
            )
          })}
        </ul>
      )}
    </PagedList>
  )
}
