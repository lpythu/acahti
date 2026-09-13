import type { ReactNode } from "react"
import { cn } from "cn"

import { EmptyState } from "@/components/empty-state"
import { Skeleton } from "@/components/ui/skeleton"

export function PageSkeleton({ lines = 6 }: { lines?: number }) {
  return (
    <div className="flex flex-col gap-3" aria-busy>
      {Array.from({ length: lines }, (_, i) => (
        <Skeleton key={i} className="h-4 w-full" />
      ))}
    </div>
  )
}

export function TableSkeleton({ rows = 6 }: { rows?: number }) {
  return (
    <div className="flex flex-col gap-2" aria-busy>
      <Skeleton className="h-8 w-full" />
      {Array.from({ length: rows }, (_, i) => (
        <Skeleton key={i} className="h-10 w-full" />
      ))}
    </div>
  )
}

export function PageFrame({
  loading,
  error,
  empty,
  emptyText,
  header,
  children,
  className,
  skeleton = "lines",
}: {
  loading: boolean
  error?: string
  empty?: boolean
  emptyText?: string
  header?: ReactNode
  children?: ReactNode
  className?: string
  skeleton?: "lines" | "table"
}) {
  return (
    <div className={cn("flex flex-col gap-4 px-4 py-4 lg:px-6 md:py-6", className)}>
      {header}
      {error && !loading ? <p className="text-sm text-destructive">{error}</p> : null}
      {loading ? (
        skeleton === "table" ? <TableSkeleton /> : <PageSkeleton />
      ) : empty ? (
        <EmptyState>{emptyText || ""}</EmptyState>
      ) : (
        children
      )}
    </div>
  )
}
