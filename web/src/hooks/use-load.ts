import { useCallback, useEffect, useRef, useState } from "react"

import { useT } from "@/i18n/i18n"

export function useLoad<T>(
  loader: () => Promise<T>,
  deps: readonly unknown[] = [],
  enabled = true,
) {
  const t = useT()
  const [data, setData] = useState<T | null>(null)
  const [error, setError] = useState("")
  const [loading, setLoading] = useState(enabled)
  const loaderRef = useRef(loader)
  loaderRef.current = loader

  const run = useCallback(
    async (soft: boolean) => {
      if (!enabled) return
      if (!soft) {
        setLoading(true)
      }
      try {
        const next = await loaderRef.current()
        setData(next)
        setError("")
      } catch (err) {
        setError(err instanceof Error ? err.message : t("loadError"))
      } finally {
        setLoading(false)
      }
    },
    [enabled, t],
  )

  useEffect(() => {
    if (!enabled) {
      setLoading(false)
      return
    }
    void run(false)
    // deps identify the resource; reload() is a soft refresh
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [enabled, ...deps])

  const reload = useCallback(() => run(true), [run])
  const apply = useCallback((fn: (cur: T | null) => T | null) => {
    setData((cur) => fn(cur))
  }, [])
  return { data, error, loading, reload, apply }
}
