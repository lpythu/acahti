import { RunStatusIcon } from "@/components/run-status-icon"
import { statusText } from "@/components/status-badge"
import { Badge } from "@/components/ui/badge"
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip"
import { useT } from "@/i18n/i18n"

const VISIBLE = 4

export function PipelineJobDots({ jobs }: { jobs: { name: string; state: string }[] }) {
  const t = useT()
  if (!jobs.length) return null

  const named = jobs.filter((s) => s.name.trim() && s.name !== ".")
  if (!named.length) return null
  const shown = named.slice(0, VISIBLE)
  const extra = named.slice(VISIBLE)

  return (
    <ol className="flex flex-wrap items-center justify-start gap-1" aria-label={t("jobs")}>
      {shown.map((s, i) => {
        const label = statusText(s.state, t)
        return (
          <li key={`${s.name}-${i}`}>
            <Tooltip>
              <TooltipTrigger
                render={
                  <Badge variant="outline" className="font-normal">
                    <RunStatusIcon status={s.state} className="size-3" />
                    <span className="max-w-20 truncate">{s.name}</span>
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
      {extra.length ? (
        <li>
          <Tooltip>
            <TooltipTrigger
              render={<Badge variant="outline" className="font-normal" />}
              aria-label={`+${extra.length}`}
            >
              +{extra.length}
            </TooltipTrigger>
            <TooltipContent className="flex-col items-start gap-1">
              {extra.map((s, i) => (
                <span key={`${s.name}-${i}`}>
                  {s.name}
                  <span className="opacity-70"> {statusText(s.state, t)}</span>
                </span>
              ))}
            </TooltipContent>
          </Tooltip>
        </li>
      ) : null}
    </ol>
  )
}
