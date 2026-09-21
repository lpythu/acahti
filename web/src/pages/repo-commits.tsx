import { useMemo, useState, type FormEvent } from "react"
import { Link, useSearchParams } from "react-router-dom"
import { CheckIcon, CopyIcon, FolderTreeIcon, GitBranchIcon, SearchIcon } from "lucide-react"

import { PagedList } from "@/components/paged-list"
import { RunStatusIcon } from "@/components/run-status-icon"
import { statusText } from "@/components/status-badge"
import { Avatar, AvatarFallback } from "@/components/ui/avatar"
import { Input } from "@/components/ui/input"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { useLoad } from "@/hooks/use-load"
import { usePage } from "@/hooks/use-page"
import { useT } from "@/i18n/i18n"
import { api, type Commit } from "@/lib/api"
import { dayKey, formatRelative, formatStamp } from "@/lib/format"
import { commitAuthor, commitTitle, filterCommitPage, isCommitSha, shortSha } from "@/lib/git"
import { useRepo } from "@/pages/repo-layout"

function CommitRow({ owner, name, c }: { owner: string; name: string; c: Commit }) {
  const t = useT()
  const [copied, setCopied] = useState(false)
  const href = `/repos/${owner}/${name}/commits/${c.sha}`
  const author = commitAuthor(c)
  const date = c.commit?.author?.date
  const check = c.check_status

  async function copySha() {
    await navigator.clipboard.writeText(c.sha)
    setCopied(true)
    window.setTimeout(() => setCopied(false), 1500)
  }

  const checkIcon = check ? <RunStatusIcon status={check} className="size-4" /> : null

  return (
    <li className="flex items-start justify-between gap-3 px-3 py-3">
      <div className="flex min-w-0 items-start gap-2">
        <Avatar size="sm" className="mt-0.5">
          <AvatarFallback>{author.slice(0, 1).toUpperCase()}</AvatarFallback>
        </Avatar>
        <div className="min-w-0">
          <Link className="block truncate text-sm font-medium hover:underline" to={href}>
            {commitTitle(c.commit?.message) || shortSha(c.sha)}
          </Link>
          <p className="mt-0.5 truncate text-xs text-muted-foreground">
            {author} {t("authored")}{" "}
            <span title={formatStamp(date)}>{formatRelative(date) || formatStamp(date)}</span>
          </p>
        </div>
      </div>
      <div className="flex shrink-0 items-center gap-1">
        {checkIcon ? (
          <Link
            className="rounded-md p-1.5 text-muted-foreground hover:bg-muted"
            to={href}
            aria-label={statusText(check!, t)}
            title={statusText(check!, t)}
          >
            {checkIcon}
          </Link>
        ) : null}
        <button
          type="button"
          className="rounded-md p-1.5 text-muted-foreground hover:bg-muted hover:text-foreground"
          aria-label={t("copy")}
          onClick={() => void copySha()}
        >
          {copied ? <CheckIcon className="size-3.5" /> : <CopyIcon className="size-3.5" />}
        </button>
        <Link
          className="rounded-md p-1.5 text-muted-foreground hover:bg-muted hover:text-foreground"
          to={`/repos/${owner}/${name}?ref=${c.sha}`}
          aria-label={t("browseFiles")}
        >
          <FolderTreeIcon className="size-3.5" />
        </Link>
        <Link className="rounded-md border px-1.5 py-0.5 font-mono text-xs hover:bg-muted" to={href}>
          {shortSha(c.sha)}
        </Link>
      </div>
    </li>
  )
}

export function RepoCommitsPage() {
  const t = useT()
  const { owner, name, data: head } = useRepo()
  const [sp, setSp] = useSearchParams()
  const urlRef = sp.get("ref") || ""
  const urlQ = sp.get("q") || ""
  const currentRef = urlRef || head?.ref || head?.repo.default_branch || ""
  const branches = useLoad(() => api.branches(owner, name, { page: 1, page_size: 100 }), [owner, name])
  const list = usePage(
    async (q) => {
      const query = urlQ.trim()
      if (query && isCommitSha(query)) {
        try {
          const detail = await api.commit(owner, name, query, { page: 1, page_size: 1 })
          return {
            items: detail.commit ? [detail.commit] : [],
            page: 1,
            page_size: q.page_size ?? 20,
            has_more: false,
          }
        } catch {
          return { items: [], page: 1, page_size: q.page_size ?? 20, has_more: false }
        }
      }
      const res = await api.commits(owner, name, {
        page: q.page,
        page_size: q.page_size,
        ref: urlRef || undefined,
        q: query || undefined,
      })
      return filterCommitPage(res, query)
    },
    [owner, name, urlRef, urlQ],
  )

  const branchNames = useMemo(() => {
    const names = (branches.data?.items || []).map((b) => b.name)
    if (currentRef && !names.includes(currentRef)) names.unshift(currentRef)
    return names
  }, [branches.data, currentRef])

  function setRef(next: string) {
    const q = new URLSearchParams(sp)
    const def = head?.repo.default_branch || ""
    if (!next || next === def) q.delete("ref")
    else q.set("ref", next)
    q.delete("page")
    setSp(q, { replace: true })
  }

  function setQuery(next: string) {
    const q = new URLSearchParams(sp)
    const v = next.trim()
    if (!v) q.delete("q")
    else q.set("q", v)
    q.delete("page")
    setSp(q, { replace: true })
  }

  function onSearch(e: FormEvent<HTMLFormElement>) {
    e.preventDefault()
    const field = e.currentTarget.querySelector("input")
    setQuery(field?.value || "")
  }

  const searchingSha = Boolean(urlQ) && isCommitSha(urlQ)
  const searchHint = currentRef ? t("searchCommitsHint", { branch: currentRef }) : t("searchCommitsHintNoBranch")
  const emptyText = urlQ
    ? searchingSha
      ? t("noMatchingCommit")
      : t("noMatchingCommits", { branch: currentRef || "—" })
    : t("noCommits")

  const groups = useMemo(() => {
    const m = new Map<string, Commit[]>()
    for (const c of list.items) {
      const key = dayKey(c.commit?.author?.date) || "—"
      const day = m.get(key) || []
      day.push(c)
      m.set(key, day)
    }
    return [...m.entries()]
  }, [list.items])

  return (
    <PagedList
      list={list}
      emptyText={emptyText}
      skeleton="lines"
      header={
        <div className="flex flex-wrap items-center justify-between gap-2">
          {currentRef ? (
            <Select value={currentRef} onValueChange={(v) => setRef(String(v ?? ""))}>
              <SelectTrigger size="sm" className="w-fit min-w-40" aria-label={t("branches")}>
                <GitBranchIcon className="size-3.5" />
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {branchNames.map((b) => (
                  <SelectItem key={b} value={b}>
                    {b}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          ) : (
            <span />
          )}
          <form key={urlQ} className="relative min-w-56 max-w-sm flex-1" onSubmit={onSearch}>
            <SearchIcon className="pointer-events-none absolute top-1/2 left-2.5 size-3.5 -translate-y-1/2 text-muted-foreground" />
            <Input
              type="search"
              name="q"
              defaultValue={urlQ}
              onChange={(e) => {
                if (!e.target.value.trim() && urlQ) setQuery("")
              }}
              onKeyDown={(e) => {
                if (e.key !== "Enter") return
                e.preventDefault()
                setQuery(e.currentTarget.value)
              }}
              placeholder={searchHint}
              aria-label={searchHint}
              autoComplete="off"
              spellCheck={false}
              className="h-7 pl-8"
            />
            <button type="submit" className="sr-only">
              {t("searchCommits")}
            </button>
          </form>
        </div>
      }
    >
      {() =>
        groups.length ? (
          <div className="flex flex-col gap-6">
            {groups.map(([day, items]) => (
              <section key={day}>
                <h2 className="mb-2 text-sm font-medium text-muted-foreground">{t("commitsOn", { day })}</h2>
                <ul className="divide-y rounded-md border">
                  {items.map((c) => (
                    <CommitRow key={c.sha} owner={owner} name={name} c={c} />
                  ))}
                </ul>
              </section>
            ))}
          </div>
        ) : null
      }
    </PagedList>
  )
}
