import { useState } from "react"
import { Link, useParams } from "react-router-dom"
import { CheckIcon, CopyIcon, FolderTreeIcon } from "lucide-react"

import { CommitDiff } from "@/components/commit-diff"
import { EmptyState } from "@/components/empty-state"
import { Pager } from "@/components/paged-list"
import { PageFrame } from "@/components/page-frame"
import { PipelineRunRow } from "@/components/pipeline-run-row"
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar"
import { Button } from "@/components/ui/button"
import { usePage } from "@/hooks/use-page"
import { useT } from "@/i18n/i18n"
import { api } from "@/lib/api"
import { formatRelative, formatStamp } from "@/lib/format"
import { commitAuthor, commitTitle, shortSha } from "@/lib/git"

export function RepoCommitPage() {
  const t = useT()
  const { owner = "", name = "", sha = "" } = useParams()
  const list = usePage((q) => api.commit(owner, name, sha, q), [owner, name, sha])
  const data = list.data
  const error = list.error
  const loading = list.loading
  const [copied, setCopied] = useState(false)
  const c = data?.commit
  const title = commitTitle(c?.commit?.message)
  const body = (c?.commit?.message || "").split("\n").slice(1).join("\n").trim()
  const author = c?.commit?.author
  const login = commitAuthor(c)
  const files = data?.items || []
  const stats = data?.stats
  const pipes = data?.pipelines || []
  const parents = c?.parents || []

  async function copySha() {
    if (!c?.sha) return
    await navigator.clipboard.writeText(c.sha)
    setCopied(true)
    window.setTimeout(() => setCopied(false), 1500)
  }

  return (
    <PageFrame loading={loading && !data} error={error} className="gap-5">
      {c ? (
        <>
          <div className="flex items-start justify-between gap-3">
            <div className="min-w-0">
              <h2 className="text-lg font-medium">{title || shortSha(c.sha)}</h2>
              {body ? <pre className="mt-2 font-sans text-sm whitespace-pre-wrap text-muted-foreground">{body}</pre> : null}
              <div className="mt-3 flex flex-wrap items-center gap-2 text-sm text-muted-foreground">
                <Avatar size="sm">
                  {c.author?.avatar_url ? <AvatarImage src={c.author.avatar_url} alt={login} /> : null}
                  <AvatarFallback>{login.slice(0, 1).toUpperCase()}</AvatarFallback>
                </Avatar>
                <span className="font-medium text-foreground">{login}</span>
                <span>{t("authored")}</span>
                <span title={formatStamp(author?.date)}>{formatRelative(author?.date) || formatStamp(author?.date)}</span>
              </div>
            </div>
            <Button size="sm" variant="outline" render={<Link to={`/repos/${owner}/${name}?ref=${c.sha}`} />}>
              <FolderTreeIcon />
              {t("browseFiles")}
            </Button>
          </div>
          <div className="flex flex-wrap items-center gap-2 text-sm">
            {parents.map((p) => (
              <Link
                key={p.sha}
                className="rounded-md border px-2 py-1 font-mono text-xs hover:bg-muted"
                to={`/repos/${owner}/${name}/commits/${p.sha}`}
              >
                {t("parent")} {shortSha(p.sha)}
              </Link>
            ))}
            <span className="inline-flex items-center gap-1 rounded-md border px-2 py-1 font-mono text-xs">
              {shortSha(c.sha)}
              <button type="button" className="text-muted-foreground hover:text-foreground" onClick={() => void copySha()}>
                {copied ? <CheckIcon className="size-3.5" /> : <CopyIcon className="size-3.5" />}
              </button>
            </span>
          </div>
          <p className="text-sm text-muted-foreground">
            {t("filesChanged", { n: stats?.total || files.length })}
            <span className="ml-2 tabular-nums text-emerald-700 dark:text-emerald-400">+{stats?.additions || 0}</span>
            <span className="ml-1 tabular-nums text-red-700 dark:text-red-400">−{stats?.deletions || 0}</span>
          </p>
          <section className="flex flex-col gap-2">
            <h3 className="text-sm font-medium">{t("pipelines")}</h3>
            {pipes.length === 0 ? (
              <EmptyState>{t("noPipelines")}</EmptyState>
            ) : (
              <ul className="divide-y rounded-md border">
                {pipes.map((p) => (
                  <PipelineRunRow key={p.number} pipe={p} hideRepo />
                ))}
              </ul>
            )}
          </section>
          <CommitDiff files={files} />
          <Pager page={list.page} hasMore={list.hasMore} onPage={list.setPage} />
        </>
      ) : null}
    </PageFrame>
  )
}
