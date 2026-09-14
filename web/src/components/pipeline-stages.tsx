import { cn } from "cn"

import { resolveTone } from "@/components/status-badge"
import type { Stage } from "@/lib/pipeline"

const DOT: Record<string, string> = {
  success: "bg-emerald-500",
  running: "bg-sky-500",
  pending: "bg-muted-foreground/40",
  warning: "bg-amber-500",
  danger: "bg-red-500",
}

export function PipelineStages({ stages }: { stages: Stage[] }) {
  if (!stages.length) return <span className="text-muted-foreground">—</span>

  return (
    <ol className="flex items-start">
      {stages.map((s, i) => (
        <li key={`${s.name}-${i}`} className="flex items-start">
          {i > 0 ? <span className="mt-1.5 h-px w-3 shrink-0 bg-border" /> : null}
          <div className="flex min-w-0 flex-col items-center gap-1">
            <span className={cn("size-2.5 shrink-0 rounded-full", DOT[resolveTone(s.state)])} />
            {s.name ? (
              <span className="max-w-16 truncate text-[10px] leading-none text-muted-foreground">{s.name}</span>
            ) : null}
          </div>
        </li>
      ))}
    </ol>
  )
}
