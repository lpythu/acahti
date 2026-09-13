import { FileIcon, FolderIcon } from "lucide-react"
import { useSearchParams } from "react-router-dom"
import { cn } from "cn"

import { PageFrame } from "@/components/page-frame"
import { Button } from "@/components/ui/button"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { useLoad } from "@/hooks/use-load"
import { useT } from "@/i18n/i18n"
import { api } from "@/lib/api"
import { useRepo } from "@/pages/repo-layout"

export function RepoFilesPage() {
  const t = useT()
  const repo = useRepo()
  const [sp, setSp] = useSearchParams()
  const ref = sp.get("ref") || ""
  const path = sp.get("path") || ""
  const browse = Boolean(ref || path)
  const extra = useLoad(
    () => api.repo(repo.owner, repo.name, ref || undefined, path || undefined),
    [repo.owner, repo.name, ref, path],
    browse,
  )
  const data = browse ? extra.data : repo.data
  const error = browse ? extra.error : repo.error
  const loading = browse ? extra.loading && !extra.data : repo.loading && !repo.data

  function setQuery(next: { ref?: string; path?: string }) {
    const q = new URLSearchParams(sp)
    if (next.ref !== undefined) {
      if (next.ref) q.set("ref", next.ref)
      else q.delete("ref")
    }
    if (next.path !== undefined) {
      if (next.path) q.set("path", next.path)
      else q.delete("path")
    }
    setSp(q, { replace: true })
  }

  const parent = path.includes("/") ? path.replace(/\/[^/]+$/, "") : ""
  const entries = data?.file
    ? [{ name: data.file.name, path: data.file.path, type: "file" }]
    : data?.entries || []
  const body = data?.file?.content || data?.readme || ""
  const empty = !!data && !data.file && !data.readme && entries.length === 0
  const branches = data?.branches.length ? data.branches : [{ name: data?.ref || ref || "dev" }]

  return (
    <PageFrame
      loading={loading}
      error={error}
      empty={empty}
      emptyText={t("noRepos")}
      className="min-h-0 flex-1"
      header={
        <div className="flex flex-wrap items-center gap-2">
          <Select
            value={data?.ref || ref || "dev"}
            onValueChange={(v) => setQuery({ ref: String(v ?? ""), path: "" })}
          >
            <SelectTrigger size="sm">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {branches.map((b) => (
                <SelectItem key={b.name} value={b.name}>
                  {b.name}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
          {path ? (
            <Button variant="outline" size="sm" onClick={() => setQuery({ path: parent })}>
              ..
            </Button>
          ) : null}
          {path ? <span className="font-mono text-xs text-muted-foreground">{path}</span> : null}
        </div>
      }
    >
      {data ? (
        <div className="grid min-h-0 flex-1 gap-4 md:grid-cols-[16rem_1fr]">
          <ul className="flex flex-col gap-0.5 rounded-md border p-1">
            {entries.map((e) => (
              <li key={e.path}>
                <button
                  type="button"
                  className={cn(
                    "flex w-full items-center gap-2 rounded-md px-2 py-1.5 text-left text-sm hover:bg-muted",
                    data.file?.path === e.path && "bg-muted font-medium",
                  )}
                  onClick={() => setQuery({ path: e.path })}
                >
                  {e.type === "dir" ? <FolderIcon className="size-4 shrink-0" /> : <FileIcon className="size-4 shrink-0" />}
                  <span className="truncate">{e.name}</span>
                </button>
              </li>
            ))}
          </ul>
          {body ? (
            <pre className="min-h-40 overflow-auto rounded-md border bg-muted/40 p-3 text-xs whitespace-pre-wrap">
              {body}
            </pre>
          ) : (
            <p className="text-sm text-muted-foreground">{t("readme")}</p>
          )}
        </div>
      ) : null}
    </PageFrame>
  )
}
