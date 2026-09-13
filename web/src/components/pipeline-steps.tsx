import { cn } from "cn"

import type { Step } from "@/lib/api"

export function PipelineSteps({
  steps,
  active,
  onSelect,
}: {
  steps: Step[]
  active?: number
  onSelect?: (step: Step) => void
}) {
  return (
    <ol className="flex flex-wrap items-center gap-2">
      {steps.map((s, i) => {
        const selected = active === s.pid
        return (
          <li key={`${s.pid}-${s.name}`} className="flex items-center gap-2">
            {i > 0 ? <span className="text-muted-foreground">→</span> : null}
            <button
              type="button"
              onClick={() => onSelect?.(s)}
              className={cn(
                "rounded-md border px-2.5 py-1 text-sm",
                selected ? "border-foreground bg-muted" : "border-border hover:bg-muted/60",
              )}
            >
              <span className="font-medium">{s.name || `#${s.pid}`}</span>
              <span className="ml-2 text-xs text-muted-foreground">{s.state}</span>
            </button>
          </li>
        )
      })}
    </ol>
  )
}
