import { RunStatusIcon } from "@/components/run-status-icon"
import { statusText } from "@/components/status-badge"
import { Badge } from "@/components/ui/badge"
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip"
import { useT } from "@/i18n/i18n"
import type { Job } from "@/lib/pipeline"

function stepLabel(name?: string) {
  const n = (name || "").trim()
  if (!n || n === "." || n === "pipeline") return ""
  return n
}

export function PipelineJobDots({ jobs }: { jobs: Job[] }) {
  const t = useT()
  if (!jobs.length) return null

  return (
    <ol className="flex flex-wrap items-center justify-start gap-1.5" aria-label={t("jobs")}>
      {jobs.map((job) => {
        const jobLabel = statusText(job.state, t)
        const steps = job.steps
          .map((s) => ({ name: stepLabel(s.name), state: s.state }))
          .filter((s) => s.name && s.name !== job.name)
        return (
          <li
            key={job.name}
            className="min-w-0 rounded-md border bg-muted/40 px-1.5 py-1"
            title={`${job.name}: ${jobLabel}`}
          >
            <div className="flex max-w-full items-center gap-1 text-[10px] font-medium leading-none text-muted-foreground">
              <RunStatusIcon status={job.state} className="size-3 shrink-0" />
              <span className="truncate">{job.name}</span>
            </div>
            {steps.length ? (
              <ol className="mt-1 hidden flex-wrap items-center gap-1 @4xl/main:flex">
                {steps.map((s, i) => {
                  const label = statusText(s.state, t)
                  return (
                    <li key={`${s.name}-${i}`}>
                      <Tooltip>
                        <TooltipTrigger
                          render={
                            <Badge variant="outline" className="h-5 px-1.5 font-normal">
                              <RunStatusIcon status={s.state} className="size-3" />
                              <span>{s.name}</span>
                            </Badge>
                          }
                          aria-label={`${s.name}: ${label}`}
                        />
                        <TooltipContent>
                          <span>{s.name}</span>
                          <span className="opacity-70">{label}</span>
                        </TooltipContent>
                      </Tooltip>
                    </li>
                  )
                })}
              </ol>
            ) : null}
          </li>
        )
      })}
    </ol>
  )
}
