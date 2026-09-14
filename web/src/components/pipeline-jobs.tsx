import { cn } from "cn"

import { RunStatusIcon } from "@/components/run-status-icon"
import { useT } from "@/i18n/i18n"
import type { FileBlob, Step } from "@/lib/api"
import type { Job } from "@/lib/pipeline"

export function PipelineJobs({
  jobs,
  files,
  activeStep,
  activeFile,
  onStep,
  onFile,
}: {
  jobs: Job[]
  files: FileBlob[]
  activeStep?: Step | null
  activeFile?: FileBlob | null
  onStep: (step: Step) => void
  onFile: (file: FileBlob) => void
}) {
  const t = useT()

  return (
    <div className="flex flex-col gap-4 p-2">
      <section>
        <h3 className="px-2 pb-1 text-xs font-medium text-muted-foreground">{t("jobs")}</h3>
        <ol className="flex flex-col gap-1">
          {jobs.map((job) => (
            <li key={job.name}>
              <div className="flex items-center gap-2 px-2 py-1 text-sm font-medium">
                <RunStatusIcon status={job.state} />
                <span className="min-w-0 truncate">{job.name}</span>
              </div>
              <ol className="ml-4 border-l pl-2">
                {job.steps.map((s) => {
                  const selected = !activeFile && activeStep?.pid === s.pid && activeStep.name === s.name
                  return (
                    <li key={`${s.pid}-${s.name}`}>
                      <button
                        type="button"
                        onClick={() => onStep(s)}
                        className={cn(
                          "flex w-full items-center gap-2 rounded-md px-2 py-1 text-left text-sm",
                          selected ? "bg-muted" : "hover:bg-muted/60",
                        )}
                      >
                        <RunStatusIcon status={s.state} />
                        <span className="min-w-0 truncate">{s.name || `#${s.pid}`}</span>
                      </button>
                    </li>
                  )
                })}
              </ol>
            </li>
          ))}
        </ol>
      </section>
      <section>
        <h3 className="px-2 pb-1 text-xs font-medium text-muted-foreground">{t("pipelineFiles")}</h3>
        {files.length ? (
          <ul className="flex flex-col gap-0.5">
            {files.map((f) => {
              const selected = activeFile?.path === f.path
              return (
                <li key={f.path}>
                  <button
                    type="button"
                    onClick={() => onFile(f)}
                    className={cn(
                      "flex w-full items-center rounded-md px-2 py-1 text-left text-sm",
                      selected ? "bg-muted" : "hover:bg-muted/60",
                    )}
                  >
                    <span className="min-w-0 truncate">{f.path}</span>
                  </button>
                </li>
              )
            })}
          </ul>
        ) : (
          <p className="px-2 text-xs text-muted-foreground">{t("noPipelineFiles")}</p>
        )}
      </section>
    </div>
  )
}
