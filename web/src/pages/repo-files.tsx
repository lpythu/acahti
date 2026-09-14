import { useSearchParams } from "react-router-dom"

import { Markdown, isMarkdownPath } from "@/components/markdown"
import { RepoFileTree } from "@/components/repo-file-tree"
import { CodeBlock } from "@/components/ui/code-block"
import { ResizableHandle, ResizablePanel, ResizablePanelGroup } from "@/components/ui/resizable"
import { AutoHideScroll } from "@/components/ui/auto-hide-scroll"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { Skeleton } from "@/components/ui/skeleton"
import { useLoad } from "@/hooks/use-load"
import { api } from "@/lib/api"
import { langOf } from "@/lib/lang"
import { useRepo } from "@/pages/repo-layout"

export function RepoFilesPage() {
  const repo = useRepo()
  const [sp, setSp] = useSearchParams()
  const ref = sp.get("ref") || ""
  const path = sp.get("path") || ""
  const root = repo.data
  const preview = useLoad(
    () => api.repo(repo.owner, repo.name, ref || undefined, path || undefined),
    [repo.owner, repo.name, ref, path],
    Boolean(path),
  )
  const currentRef = root?.ref || ref || "dev"
  const branches = root?.branches.length ? root.branches : [{ name: currentRef }]
  const entries = root?.entries || []
  const file = path ? preview.data?.file : undefined
  const readme = path ? preview.data?.readme : root?.readme
  const readmeName = path ? `${path.replace(/\/$/, "")}/README.md` : "README.md"
  const loading = repo.loading && !root
  const error = repo.error || (path ? preview.error : "")
  const previewLoading = Boolean(path) && preview.loading && !preview.data

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

  const title = file?.name || (readme ? readmeName.split("/").pop() : "")
  const md = file ? isMarkdownPath(file.name) : Boolean(readme)

  return (
    <div className="flex min-h-0 flex-1 flex-col">
      {error && !root ? <p className="px-4 py-3 text-sm text-destructive">{error}</p> : null}
      <ResizablePanelGroup orientation="horizontal" className="min-h-0 flex-1">
        <ResizablePanel defaultSize={22} minSize={14} className="flex min-h-0 flex-col">
          <div className="border-b p-2">
            <Select
              value={currentRef}
              onValueChange={(v) => setQuery({ ref: String(v ?? ""), path: "" })}
            >
              <SelectTrigger size="sm" className="w-full">
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
          </div>
          <AutoHideScroll className="min-h-0 flex-1">
            {loading ? (
              <div className="flex flex-col gap-2 p-3">
                {Array.from({ length: 8 }, (_, i) => (
                  <Skeleton key={i} className="h-5 w-full" />
                ))}
              </div>
            ) : (
              <RepoFileTree
                owner={repo.owner}
                name={repo.name}
                gitRef={currentRef === (root?.repo.default_branch || "dev") ? "" : currentRef}
                entries={entries}
                selected={path || (root?.readme ? "README.md" : "")}
                onPick={(next) => setQuery({ path: next })}
              />
            )}
          </AutoHideScroll>
        </ResizablePanel>
        <ResizableHandle />
        <ResizablePanel defaultSize={78} className="flex min-h-0 flex-col">
          {title ? <div className="border-b px-4 py-2 text-sm font-medium">{title}</div> : null}
          <div className="min-h-0 flex-1">
            {previewLoading ? (
              <div className="flex flex-col gap-2 p-4">
                {Array.from({ length: 10 }, (_, i) => (
                  <Skeleton key={i} className="h-4 w-full" />
                ))}
              </div>
            ) : file && !md ? (
              <CodeBlock code={file.content} language={langOf(file.name)} />
            ) : file && md ? (
              <AutoHideScroll className="size-full">
                <div className="p-6">
                  <Markdown>{file.content}</Markdown>
                </div>
              </AutoHideScroll>
            ) : readme ? (
              <AutoHideScroll className="size-full">
                <div className="p-6">
                  <Markdown>{readme}</Markdown>
                </div>
              </AutoHideScroll>
            ) : null}
          </div>
        </ResizablePanel>
      </ResizablePanelGroup>
    </div>
  )
}
