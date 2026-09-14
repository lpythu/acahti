export type Page<T> = {
  items: T[]
  page: number
  page_size: number
  has_more: boolean
}

export type PageQuery = { page?: number; page_size?: number }

export function pageQS(q: PageQuery = {}, extra?: Record<string, string | undefined>): string {
  const p = new URLSearchParams()
  p.set("page", String(q.page && q.page > 0 ? q.page : 1))
  p.set("page_size", String(q.page_size && q.page_size > 0 ? q.page_size : 20))
  for (const [k, v] of Object.entries(extra || {})) {
    if (v) p.set(k, v)
  }
  return `?${p.toString()}`
}
