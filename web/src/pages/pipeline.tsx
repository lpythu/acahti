import { useEffect, useMemo, useState } from "react"
import { Link, useNavigate, useParams } from "react-router-dom"
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
import type { FileBlob, Step } from "@/lib/api"
import { api } from "@/lib/api"
import { formatUnix } from "@/lib/format"
import { langOf } from "@/lib/lang"
import { pipelineHref } from "@/lib/nav"
import { asPipeline, argosLinksOf, inFlight, jobsOf, namedSecrets, triggerKey, triggerVars, waitLine } from "@/lib/pipeline"
import { useRepo } from "@/pages/repo-layout"

export function PipelinePage() {
  const t = useT()
  const nav = useNavigate()
  const { data: repoData } = useRepo()
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
    apply((cur) => (cur ? { ...cur, pipeline: next } : cur))
  })

  const jobs = useMemo(() => (p ? jobsOf(p) : []), [p])
  const argosLinks = useMemo(() => (p ? argosLinksOf(p) : []), [p])
  const yamlSecretNames = useMemo(() => namedSecrets(files), [files])
  const activeStep = useMemo(() => {
    if (step && jobs.some((j) => j.steps.some((s) => s.pid === step.pid && s.name === step.name))) {
      return step
    }
    return jobs[0]?.steps[0] ?? null
  }, [jobs, step])
  const stepLogId = activeStep?.id || activeStep?.pid || 0

  useEffect(() => {
    if (file || !stepLogId) {
      setLog("")
      setLogLoading(false)
      return
    }
    setLog("")
    setLogLoading(true)
    api
      .pipelineLog(owner, name, n, stepLogId)
      .then((r) => setLog(!r.log || r.log === "null" ? "" : r.log))
      .catch(() => setLog(""))
      .finally(() => setLogLoading(false))
  }, [owner, name, n, stepLogId, file])

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

  const wait = waitLine(p)
  const activeJob = jobs.find((j) => j.steps.some((s) => s.pid === activeStep?.pid && s.name === activeStep?.name))
  const jobWait = waitLine({
    status: activeStep?.state || p.status,
    wait: activeJob?.wait || p.wait,
    queue_position: activeJob?.queue_position || p.queue_position,
    agent: activeJob?.agent || p.agent,
  })

  return (
    <div className="flex min-h-0 flex-1 flex-col">
      <div className="flex shrink-0 items-start justify-between gap-3 border-b px-4 py-3 lg:px-6">
        <div className="min-w-0">
          <div className="flex flex-wrap items-center gap-2">
            <h2 className="text-lg font-medium">
              #{n} {p.title || p.event || t("pipelines")}
            </h2>
            <StatusBadge status={p.status} />
            {argosLinks.map((item) => (
              <Button key={item.url} size="sm" variant="outline" render={<a href={item.url} target="_blank" rel="noreferrer" />}>
                {t("argosDash")}
                {argosLinks.length > 1 ? ` ${item.name.replace(/^e2e\./, "")}` : ""}
              </Button>
            ))}
          </div>
          <p className="mt-1 truncate text-sm text-muted-foreground">
            {t(triggerKey(p.event, p.ref), triggerVars(p))}
            {p.branch ? ` · ${p.branch}` : ""}
            {p.commit ? ` · ${p.commit.slice(0, 7)}` : ""}
            {` · ${formatUnix(p.started || p.created)}`}
          </p>
          {wait ? (
            <p className="mt-1 text-sm text-muted-foreground">
              {t(wait.key, wait.vars)}
              {p.agent ? ` · ${t("waitAgent", { agent: p.agent })}` : ""}
            </p>
          ) : p.agent ? (
            <p className="mt-1 text-sm text-muted-foreground">{t("waitAgent", { agent: p.agent })}</p>
          ) : null}
          <p className="mt-1 text-sm text-muted-foreground">
            {yamlSecretNames.length
              ? t("yamlSecrets", { names: yamlSecretNames.join(", ") })
              : t("yamlSecretsNone")}
          </p>
        </div>
        <div className="flex shrink-0 items-center gap-2">
          {p.status === "blocked" ? (
            <Button size="sm" disabled={busy} onClick={() => void approve()}>
              {t("approve")}
            </Button>
          ) : null}
          {inFlight(p.status) && p.status !== "blocked" ? (
            <Button size="sm" variant="outline" disabled={busy} onClick={() => void cancel()}>
              {t("cancel")}
            </Button>
          ) : null}
          {!inFlight(p.status) ? (
            <Button size="sm" variant="destructive" disabled={busy} onClick={() => void remove()}>
              {t("deletePipeline")}
            </Button>
          ) : null}
          <Button size="sm" variant="outline" disabled={busy} onClick={() => void rerun()}>
            {t("rerun")}
          </Button>
          {repoData?.repo.can_manage_secrets ? (
            <Button size="sm" variant="outline" render={<Link to={`/repos/${owner}/${name}/secrets`} />}>
              {t("secretsPage")}
            </Button>
          ) : null}
        </div>
      </div>
      <ResizablePanelGroup orientation="horizontal" className="min-h-0 flex-1">
        <ResizablePanel defaultSize={24} minSize={16} className="flex min-h-0 flex-col">
          <AutoHideScroll className="min-h-0 flex-1">
            <PipelineJobs
              jobs={jobs}
              files={files ?? []}
              activeStep={activeStep}
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
          ) : log || activeStep?.error || p.error ? (
            <AutoHideScroll className="min-h-0 flex-1">
              <pre className="p-4 font-mono text-xs leading-relaxed break-words whitespace-pre-wrap">
                {log || activeStep?.error || p.error}
              </pre>
            </AutoHideScroll>
          ) : (
            <EmptyState>{jobWait ? t(jobWait.key, jobWait.vars) : t("noLog")}</EmptyState>
          )}
        </ResizablePanel>
      </ResizablePanelGroup>
    </div>
  )
}
