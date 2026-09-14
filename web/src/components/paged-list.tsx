import type { ReactNode } from "react"
import { ChevronLeftIcon, ChevronRightIcon } from "lucide-react"

import { PageFrame } from "@/components/page-frame"
import { Button } from "@/components/ui/button"
import type { UsePage } from "@/hooks/use-page"
import { useT } from "@/i18n/i18n"

export function Pager({
  page,
  hasMore,
  onPage,
}: {
  page: number
  hasMore: boolean
  onPage: (n: number) => void
}) {
  const t = useT()
  if (page <= 1 && !hasMore) return null
  return (
    <div className="flex items-center justify-end gap-2">
      <Button
        type="button"
        variant="outline"
        size="icon-sm"
        disabled={page <= 1}
        aria-label={t("prev")}
        onClick={() => onPage(page - 1)}
      >
        <ChevronLeftIcon />
      </Button>
      <span className="text-sm text-muted-foreground">{t("pageN", { n: page })}</span>
      <Button
        type="button"
        variant="outline"
        size="icon-sm"
        disabled={!hasMore}
        aria-label={t("next")}
        onClick={() => onPage(page + 1)}
      >
        <ChevronRightIcon />
      </Button>
    </div>
  )
}

export function MoreButton({ page, hasMore, onPage }: { page: number; hasMore: boolean; onPage: (n: number) => void }) {
  const t = useT()
  if (!hasMore) return null
  return (
    <Button type="button" variant="ghost" size="sm" className="w-full" onClick={() => onPage(page + 1)}>
      {t("more")}
    </Button>
  )
}

export function PagedList<T>({
  list,
  emptyText,
  header,
  skeleton = "table",
  className,
  children,
}: {
  list: UsePage<T>
  emptyText?: string
  header?: ReactNode
  skeleton?: "lines" | "table"
  className?: string
  children: (items: T[]) => ReactNode
}) {
  return (
    <PageFrame
      loading={list.loading && !list.data}
      error={list.error}
      empty={list.empty}
      emptyText={emptyText}
      header={header}
      skeleton={skeleton}
      className={className}
    >
      {children(list.items)}
      <Pager page={list.page} hasMore={list.hasMore} onPage={list.setPage} />
    </PageFrame>
  )
}
