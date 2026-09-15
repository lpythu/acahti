import { Link, useSearchParams } from "react-router-dom"
import { cn } from "cn"
import { GitPullRequestClosedIcon, GitPullRequestIcon } from "lucide-react"

import { PagedList } from "@/components/paged-list"
import { StatusBadge } from "@/components/status-badge"
import { Button } from "@/components/ui/button"
import { usePage } from "@/hooks/use-page"
import { useT } from "@/i18n/i18n"
import { api, type PR } from "@/lib/api"
import { displayName } from "@/lib/user"
import { useRepo } from "@/pages/repo-layout"

type PullState = "open" | "closed"

function prStatus(pr: PR) {
  if (pr.merged) return "merged"
  return pr.state || "open"
}

export function RepoPullsPage() {
  const t = useT()
  const { owner, name } = useRepo()
  const [sp, setSp] = useSearchParams()
  const state: PullState = sp.get("state") === "closed" ? "closed" : "open"
  const list = usePage((q) => api.pulls(owner, name, { ...q, state }), [owner, name, state])

  function setState(next: PullState) {
    const q = new URLSearchParams(sp)
    if (next === "open") q.delete("state")
    else q.set("state", next)
    q.delete("page")
    setSp(q, { replace: true })
  }

  return (
    <PagedList
      list={list}
      emptyText={state === "open" ? t("noPulls") : t("noClosedPulls")}
      skeleton="lines"
      header={
        <div className="flex items-center gap-1 border-b pb-2">
          <Button
            type="button"
            variant="ghost"
            size="sm"
            className={cn(state === "open" && "bg-muted font-medium text-foreground")}
            aria-pressed={state === "open"}
            onClick={() => setState("open")}
          >
            <GitPullRequestIcon />
            {t("pullsOpen")}
          </Button>
          <Button
            type="button"
            variant="ghost"
            size="sm"
            className={cn(state === "closed" && "bg-muted font-medium text-foreground")}
            aria-pressed={state === "closed"}
            onClick={() => setState("closed")}
          >
            <GitPullRequestClosedIcon />
            {t("pullsClosed")}
          </Button>
        </div>
      }
    >
      {(items) => (
        <ul className="divide-y rounded-md border">
          {items.map((pr) => {
            const status = prStatus(pr)
            const author = displayName(pr.user)
            const head = pr.head?.ref || ""
            const base = pr.base?.ref || ""
            return (
              <li key={pr.number} className="flex items-start gap-3 px-3 py-3">
                <StatusBadge status={status} />
                <div className="min-w-0 flex-1">
                  <div className="flex flex-wrap items-baseline gap-x-2 gap-y-0.5">
                    <Link
                      className="truncate text-sm font-medium hover:underline"
                      to={`/repos/${owner}/${name}/pulls/${pr.number}`}
                    >
                      {pr.title}
                    </Link>
                    <span className="text-xs text-muted-foreground">#{pr.number}</span>
                  </div>
                  {(author || (head && base)) && (
                    <p className="mt-0.5 truncate text-xs text-muted-foreground">
                      {author}
                      {author && head && base ? " · " : ""}
                      {head && base ? `${head} → ${base}` : ""}
                    </p>
                  )}
                </div>
              </li>
            )
          })}
        </ul>
      )}
    </PagedList>
  )
}
