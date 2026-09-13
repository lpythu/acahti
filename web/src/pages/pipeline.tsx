import { useEffect, useState } from "react"
import { Link, useParams } from "react-router-dom"

import { EmptyState } from "@/components/empty-state"
import { PageFrame } from "@/components/page-frame"
import { PipelineSteps } from "@/components/pipeline-steps"
import { StatusBadge } from "@/components/status-badge"
import { Button } from "@/components/ui/button"
import { useEvents } from "@/hooks/use-events"
import { useLoad } from "@/hooks/use-load"
import { useT } from "@/i18n/i18n"
import { api, type Step } from "@/lib/api"

export function PipelinePage() {
  const t = useT()
  const { owner = "", name = "", number = "" } = useParams()
  const n = Number(number)
  const { data, error, loading, reload } = useLoad(() => api.pipeline(owner, name, n), [owner, name, n])
  const [step, setStep] = useState<Step | null>(null)
  const [log, setLog] = useState("")
  const [busy, setBusy] = useState(false)
  useEvents(reload)

  useEffect(() => {
    if (!data) return
    setStep((cur) => {
      if (cur && data.steps.some((s) => s.pid === cur.pid)) return cur
      return data.steps[0] || null
    })
  }, [data])

  useEffect(() => {
    if (!step) {
      setLog("")
      return
    }
    api
      .pipelineLog(owner, name, n, step.id || step.pid)
      .then((r) => setLog(r.log || ""))
      .catch(() => setLog(""))
  }, [owner, name, n, step])

  const p = data?.pipeline
  return (
    <PageFrame loading={loading && !data} error={error} className="gap-6">
      {p ? (
        <>
          <div>
            <Link className="text-sm text-muted-foreground hover:underline" to={`/repos/${owner}/${name}/pipelines`}>
              {owner}/{name}
            </Link>
            <h2 className="text-lg font-medium">
              #{n} {p.title || p.event || t("runs")}
            </h2>
            <StatusBadge status={p.status} />
          </div>
          <div className="flex gap-2">
            {p.status === "blocked" ? (
              <Button
                size="sm"
                disabled={busy}
                onClick={async () => {
                  setBusy(true)
                  try {
                    await api.approve(owner, name, n)
                    await reload()
                  } finally {
                    setBusy(false)
                  }
                }}
              >
                {t("approve")}
              </Button>
            ) : null}
            <Button
              size="sm"
              variant="outline"
              disabled={busy}
              onClick={async () => {
                setBusy(true)
                try {
                  await api.rerun(owner, name, n)
                  await reload()
                } finally {
                  setBusy(false)
                }
              }}
            >
              {t("rerun")}
            </Button>
          </div>
          <PipelineSteps steps={data?.steps || []} active={step?.pid} onSelect={setStep} />
          <section className="flex flex-col gap-2">
            <h3 className="text-sm font-medium">{t("log")}</h3>
            {log ? (
              <pre className="max-h-[28rem] overflow-auto rounded-md bg-muted p-3 font-mono text-xs whitespace-pre-wrap">
                {log}
              </pre>
            ) : (
              <EmptyState>{t("noLog")}</EmptyState>
            )}
          </section>
        </>
      ) : null}
    </PageFrame>
  )
}
