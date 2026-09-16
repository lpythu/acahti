import { useState } from "react"
import { Link } from "react-router-dom"
import { toast } from "sonner"

import { PagedList } from "@/components/paged-list"
import { StatusBadge } from "@/components/status-badge"
import { Button } from "@/components/ui/button"
import { useEvents } from "@/hooks/use-events"
import { usePage } from "@/hooks/use-page"
import { useT } from "@/i18n/i18n"
import { api, repoName, splitRepo, type PR } from "@/lib/api"
import { displayName } from "@/lib/user"

export function BoardPage() {
  const t = useT()
  const prs = usePage((q) => api.boardPRs(q), [])
  const [busy, setBusy] = useState("")

  useEvents((ev) => {
    if (ev.type === "forgejo") void prs.reload()
  })

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

  return (
    <PagedList list={prs} emptyText={t("boardEmpty")} skeleton="lines">
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
