import { useState } from "react"
import { Link, useNavigate, useSearchParams } from "react-router-dom"

import { PagedList } from "@/components/paged-list"
import { PipelineRunRow } from "@/components/pipeline-run-row"
import { StatusBadge } from "@/components/status-badge"
import { useEvents } from "@/hooks/use-events"
import { usePage } from "@/hooks/use-page"
import { useT } from "@/i18n/i18n"
import { api, repoName, splitRepo, type Pipeline } from "@/lib/api"
import { pipelineHref } from "@/lib/nav"
import { asPipeline, upsertRun } from "@/lib/pipeline"
import { displayName } from "@/lib/user"

type BoardSection = "pipes" | "prs"

export function BoardPage() {
  const t = useT()
  const nav = useNavigate()
  const [sp] = useSearchParams()
  const section: BoardSection = sp.get("section") === "prs" ? "prs" : "pipes"
  const pipes = usePage((q) => api.boardPipes(q), [], { enabled: section === "pipes" })
  const prs = usePage((q) => api.boardPRs(q), [], { enabled: section === "prs" })
  const [busy, setBusy] = useState("")
  const [actionErr, setActionErr] = useState("")

  useEvents((ev) => {
    if (ev.type === "forgejo") {
      if (section === "prs") void prs.reload()
      return
    }
    if (ev.type !== "pipeline.updated" || section !== "pipes") return
    const next = asPipeline(ev.data)
    if (!next) return
    const attention = new Set(["blocked", "failure", "error", "killed", "declined"])
    pipes.apply((page) => {
      const base = page ?? { items: [] as Pipeline[], page: pipes.page, page_size: pipes.pageSize, has_more: false }
      const withoutRepo = {
        ...base,
        items: base.items.filter((p: Pipeline) => p.repo !== next.repo),
      }
      if (!attention.has(next.status)) return withoutRepo
      return upsertRun(withoutRepo, next, pipes.page)
    })
  })

  async function approve(p: Pipeline) {
    const { owner, name } = splitRepo(p.repo)
    setBusy(`${p.repo}-${p.number}`)
    setActionErr("")
    try {
      await api.approve(owner, name, p.number)
      await pipes.reload()
    } catch (err) {
      setActionErr(err instanceof Error ? err.message : t("loadError"))
    } finally {
      setBusy("")
    }
  }

  async function rerun(p: Pipeline) {
    const { owner, name } = splitRepo(p.repo)
    setBusy(`${p.repo}-${p.number}`)
    setActionErr("")
    try {
      const next = await api.rerun(owner, name, p.number)
      nav(pipelineHref(owner, name, next.number))
    } catch (err) {
      setActionErr(err instanceof Error ? err.message : t("loadError"))
      setBusy("")
    }
  }

  if (section === "prs") {
    return (
      <PagedList list={prs} emptyText={t("inboxEmpty")} skeleton="lines">
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
                </li>
              )
            })}
          </ul>
        )}
      </PagedList>
    )
  }

  return (
    <PagedList
      list={{ ...pipes, error: pipes.error || actionErr }}
      emptyText={t("inboxEmpty")}
      skeleton="lines"
    >
      {(items) => (
        <ul className="divide-y rounded-md border">
          {items.map((p) => (
            <PipelineRunRow
              key={`${p.repo}-${p.number}`}
              pipe={p}
              busy={busy === `${p.repo}-${p.number}`}
              onApprove={p.status === "blocked" ? () => void approve(p) : undefined}
              onRun={p.status === "blocked" ? undefined : () => void rerun(p)}
            />
          ))}
        </ul>
      )}
    </PagedList>
  )
}
