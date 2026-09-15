import type { ReactNode } from "react"

import { Skeleton } from "@/components/ui/skeleton"

export function MenuPanel({
  loading,
  error,
  empty,
  children,
}: {
  loading?: boolean
  error?: string
  empty?: string
  children?: ReactNode
}) {
  if (loading) {
    return (
      <div className="flex flex-col gap-2">
        <Skeleton className="h-7 w-full" />
        <Skeleton className="h-7 w-5/6" />
        <Skeleton className="h-7 w-4/6" />
        <Skeleton className="h-7 w-3/4" />
      </div>
    )
  }
  if (error) {
    return <p className="text-sm text-destructive">{error}</p>
  }
  if (empty) {
    return <p className="text-sm text-muted-foreground">{empty}</p>
  }
  return children
}
