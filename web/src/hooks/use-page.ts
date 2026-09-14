import { useState } from "react"
import { useSearchParams } from "react-router-dom"

import { useLoad } from "@/hooks/use-load"
import type { Page, PageQuery } from "@/lib/page"

export type UsePage<T, P extends Page<T> = Page<T>> = {
  data: P | null
  items: T[]
  page: number
  pageSize: number
  hasMore: boolean
  empty: boolean
  loading: boolean
  error: string
  reload: () => void
  setPage: (n: number) => void
}

export function usePage<P extends Page<unknown>>(
  loader: (q: PageQuery) => Promise<P>,
  deps: readonly unknown[] = [],
  opts?: { param?: string; pageSize?: number; enabled?: boolean; url?: boolean },
): UsePage<P["items"][number], P> {
  const url = opts?.url !== false
  const param = opts?.param || "page"
  const pageSize = opts?.pageSize ?? 20
  const enabled = opts?.enabled !== false
  const [sp, setSp] = useSearchParams()
  const [local, setLocal] = useState(1)
  const page = url ? Math.max(1, Number(sp.get(param) || 1) || 1) : local
  const load = useLoad(() => loader({ page, page_size: pageSize }), [...deps, page, pageSize], enabled)

  function setPage(n: number) {
    const next = Math.max(1, n)
    if (!url) {
      setLocal(next)
      return
    }
    const q = new URLSearchParams(sp)
    if (next <= 1) q.delete(param)
    else q.set(param, String(next))
    setSp(q, { replace: true })
  }

  const items = (load.data?.items || []) as P["items"]
  return {
    ...load,
    items,
    page,
    pageSize,
    hasMore: Boolean(load.data?.has_more),
    empty: Boolean(load.data) && items.length === 0,
    setPage,
  }
}
