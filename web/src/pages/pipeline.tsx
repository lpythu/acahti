import { useEffect, useMemo, useState } from "react"
import { useNavigate, useParams } from "react-router-dom"
import { toast } from "sonner"

import { EmptyState } from "@/components/empty-state"
import { PipelineJobs } from "@/components/pipeline-jobs"
import { StatusBadge } from "@/components/status-badge"
import { AutoHideScroll } from "@/components/ui/auto-hide-scroll"
import { Button } from "@/components/ui/button"
import { CodeBlock } from "@/components/ui/code-block"
import { ResizableHandle, ResizablePanel, ResizablePanelGroup } from "@/components/ui/resizable"
import { Skeleton } from "@/components/ui/skeleton"
import { useEvents } from "@/hooks/use-events"
import { useLoad } from "@/hooks/use-load"
import { useT } from "@/i18n/i18n"
import type { FileBlob, PipelineDetail, Step } from "@/lib/api"
import { api } from "@/lib/api"
import { formatUnix } from "@/lib/format"
import { langOf } from "@/lib/lang"
import { pipelineHref } from "@/lib/nav"
import { asPipeline, declaredJobNames, jobsOf, triggerKey, triggerVars } from "@/lib/pipeline"

export function PipelinePage() {
  const t = useT()
  const nav = useNavigate()
  const { owner = "", name = "", number = "" } = useParams()
  const n = Number(number)
  const { data, error, loading, reload, apply } = useLoad(() => api.pipeline(owner, name, n), [owner, name, n])
  const files = data?.files
  const p = data?.pipeline
  const [step, setStep] = useState<Step | null>(null)
  const [file, setFile] = useState<FileBlob | null>(null)
  const [log, setLog] = useState("")
  const [logLoading, setLogLoading] = useState(false)
  const [busy, setBusy] = useState(false)
  useEvents((ev) => {
    if (ev.type !== "pipeline.updated") return
    const next = asPipeline(ev.data)
    if (!next || next.repo !== `${owner}/${name}` || next.number !== n) return
    apply((cur) => (cur ? ({ ...cur, pipeline: next, steps: next.steps || cur.steps } satisfies PipelineDetail) : cur))
  })

  const jobs = useMemo(
    () => (p ? jobsOf(p, data?.steps, declaredJobNames(files, p.event, p.branch || p.ref)) : []),
    [p, data?.steps, files],
  )

  useEffect(() => {
    if (!jobs.length) return
    setStep((cur) => {
      if (cur && jobs.some((j) => j.steps.some((s) => s.pid === cur.pid))) return cur
      return jobs[0].steps[0] || null
    })
    setFile(null)
  }, [jobs])

  useEffect(() => {
    if (file || !step) {
      setLog("")
      setLogLoading(false)
      return
    }
    setLog("")
    setLogLoading(true)
    api
      .pipelineLog(owner, name, n, step.id || step.pid)
      .then((r) => setLog(r.log || ""))
      .catch(() => setLog(""))
      .finally(() => setLogLoading(false))
  }, [owner, name, n, step, file])

  async function rerun() {
    setBusy(true)
    try {
      const next = await api.rerun(owner, name, n)
      toast.success(t("rerunQueued"))
      nav(pipelineHref(owner, name, next.number))
    } catch (err) {
      toast.error(err instanceof Error ? err.message : t("loadError"))
    } finally {
      setBusy(false)
    }
  }

  async function approve() {
    setBusy(true)
    try {
      await api.approve(owner, name, n)
      toast.success(t("approved"))
      await reload()
    } catch (err) {
      toast.error(err instanceof Error ? err.message : t("loadError"))
    } finally {
      setBusy(false)
    }
  }

  async function cancel() {
    setBusy(true)
    try {
      await api.cancel(owner, name, n)
      toast.success(t("canceled"))
      await reload()
    } catch (err) {
      toast.error(err instanceof Error ? err.message : t("loadError"))
    } finally {
      setBusy(false)
    }
  }

  async function remove() {
    setBusy(true)
    try {
      await api.deletePipeline(owner, name, n)
      toast.success(t("pipelineDeleted"))
      nav(`/repos/${owner}/${name}/pipelines`)
    } catch (err) {
      toast.error(err instanceof Error ? err.message : t("loadError"))
    } finally {
      setBusy(false)
    }
  }

  if (loading && !data) {
    return (
      <div className="flex min-h-0 flex-1 flex-col gap-3 p-4">
        <Skeleton className="h-8 w-64" />
        <Skeleton className="h-full w-full" />
      </div>
    )
  }

  if (error && !data) {
    return <p className="px-4 py-3 text-sm text-destructive">{error}</p>
  }

  if (!p) return null

  return (
    <div className="flex min-h-0 flex-1 flex-col">
      <div className="flex shrink-0 items-start justify-between gap-3 border-b px-4 py-3 lg:px-6">
        <div className="min-w-0">
          <div className="flex flex-wrap items-center gap-2">
            <h2 className="text-lg font-medium">
              #{n} {p.title || p.event || t("pipelines")}
            </h2>
            <StatusBadge status={p.status} />
          </div>
          <p className="mt-1 truncate text-sm text-muted-foreground">
            {t(triggerKey(p.event), triggerVars(p))}
            {p.branch ? ` · ${p.branch}` : ""}
            {p.commit ? ` · ${p.commit.slice(0, 7)}` : ""}
            {` · ${formatUnix(p.started || p.created)}`}
          </p>
        </div>
        <div className="flex shrink-0 items-center gap-2">
          {p.status === "blocked" ? (
            <Button size="sm" disabled={busy} onClick={() => void approve()}>
              {t("approve")}
            </Button>
          ) : null}
          {p.status === "running" || p.status === "pending" ? (
            <Button size="sm" variant="outline" disabled={busy} onClick={() => void cancel()}>
              {t("cancel")}
            </Button>
          ) : null}
          {p.status !== "running" && p.status !== "pending" && p.status !== "blocked" ? (
            <Button size="sm" variant="destructive" disabled={busy} onClick={() => void remove()}>
              {t("deletePipeline")}
            </Button>
          ) : null}
          <Button size="sm" variant="outline" disabled={busy} onClick={() => void rerun()}>
            {t("rerun")}
          </Button>
        </div>
      </div>
      <ResizablePanelGroup orientation="horizontal" className="min-h-0 flex-1">
        <ResizablePanel defaultSize={24} minSize={16} className="flex min-h-0 flex-col">
          <AutoHideScroll className="min-h-0 flex-1">
            <PipelineJobs
              jobs={jobs}
              files={files ?? []}
              activeStep={step}
              activeFile={file}
              onStep={(s) => {
                setFile(null)
                setStep(s)
              }}
              onFile={setFile}
            />
          </AutoHideScroll>
        </ResizablePanel>
        <ResizableHandle />
        <ResizablePanel defaultSize={76} className="flex min-h-0 flex-col">
          {file ? (
            <AutoHideScroll className="min-h-0 flex-1">
              <div className="px-4 py-3">
                <p className="mb-2 font-mono text-sm">{file.path}</p>
                <CodeBlock code={file.content} language={langOf(file.name)} />
              </div>
            </AutoHideScroll>
          ) : logLoading ? (
            <div className="flex flex-col gap-2 p-4">
              <Skeleton className="h-4 w-full" />
              <Skeleton className="h-4 w-5/6" />
              <Skeleton className="h-4 w-4/6" />
            </div>
          ) : log || step?.error || p.error ? (
            <AutoHideScroll className="min-h-0 flex-1">
              <pre className="p-4 font-mono text-xs whitespace-pre-wrap">{log || step?.error || p.error}</pre>
            </AutoHideScroll>
          ) : (
            <EmptyState>{t("noLog")}</EmptyState>
          )}
        </ResizablePanel>
      </ResizablePanelGroup>
    </div>
  )
}
