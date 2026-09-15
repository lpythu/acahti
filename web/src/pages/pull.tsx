import { useState } from "react"
import { Link, useParams } from "react-router-dom"

import { EmptyState } from "@/components/empty-state"
import { Pager } from "@/components/paged-list"
import { PageFrame } from "@/components/page-frame"
import { StatusBadge } from "@/components/status-badge"
import { Button } from "@/components/ui/button"
import { useLoad } from "@/hooks/use-load"
import { usePage } from "@/hooks/use-page"
import { useT } from "@/i18n/i18n"
import { api } from "@/lib/api"

export function PullPage() {
  const t = useT()
  const { owner = "", name = "", number = "" } = useParams()
  const n = Number(number)
  const { data, error, loading, reload } = useLoad(() => api.pull(owner, name, n), [owner, name, n])
  const comments = usePage((q) => api.pullComments(owner, name, n, q), [
    owner,
    name,
    n,
  ])
  const [busy, setBusy] = useState(false)
  const [actionErr, setActionErr] = useState("")

  async function merge() {
    setBusy(true)
    setActionErr("")
    try {
      await api.mergePull(owner, name, n)
      await reload()
    } catch {
      setActionErr(t("checksNotGreen"))
    } finally {
      setBusy(false)
    }
  }

  const pr = data?.pr
  return (
    <PageFrame loading={loading && !data} error={error || actionErr} className="gap-6">
      {pr ? (
        <>
          <div>
            <Link className="text-sm text-muted-foreground hover:underline" to={`/repos/${owner}/${name}/pulls`}>
              {name}
            </Link>
            <h2 className="text-lg font-medium">
              #{pr.number} {pr.title}
            </h2>
            <p className="text-sm text-muted-foreground">
              {pr.user?.login} · {pr.head?.ref} → {pr.base?.ref}
            </p>
          </div>
          {pr.body ? <p className="whitespace-pre-wrap text-sm">{pr.body}</p> : null}
          <div className="flex items-center gap-2">
            <StatusBadge status={pr.merged ? "merged" : pr.state || "open"} />
            {pr.merged ? (
              <span className="text-sm text-muted-foreground">{t("merged")}</span>
            ) : (
              <Button size="sm" disabled={busy || !data?.green} onClick={() => void merge()}>
                {t("merge")}
              </Button>
            )}
          </div>
          <p className="text-sm text-muted-foreground">{t("mergeHint")}</p>

          <section className="flex flex-col gap-2">
            <h3 className="text-sm font-medium">{t("checks")}</h3>
            {(data?.checks || []).length === 0 ? (
              <EmptyState>{t("noChecks")}</EmptyState>
            ) : (
              <ul className="flex flex-col gap-1 text-sm">
                {data?.checks.map((c) => (
                  <li key={c.context} className="flex items-center gap-2">
                    <StatusBadge status={c.status} />
                    <span>{c.context}</span>
                    {c.description ? <span className="text-muted-foreground">{c.description}</span> : null}
                  </li>
                ))}
              </ul>
            )}
          </section>

          <section className="flex flex-col gap-2">
            <h3 className="text-sm font-medium">{t("comments")}</h3>
            {comments.empty ? (
              <EmptyState>{t("noComments")}</EmptyState>
            ) : (
              <ul className="flex flex-col gap-3">
                {comments.items.map((c) => (
                  <li key={c.id} className="rounded-md border p-3 text-sm">
                    <p className="text-muted-foreground">
                      {c.user?.login} · {c.created_at}
                    </p>
                    <p className="mt-1 whitespace-pre-wrap">{c.body}</p>
                  </li>
                ))}
              </ul>
            )}
            <Pager page={comments.page} hasMore={comments.hasMore} onPage={comments.setPage} />
          </section>
        </>
      ) : null}
    </PageFrame>
  )
}
