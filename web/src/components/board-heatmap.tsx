import { Heatmap } from "@/components/heatmap"
import { Skeleton } from "@/components/ui/skeleton"
import { useEvents } from "@/hooks/use-events"
import { useLoad } from "@/hooks/use-load"
import { useLocale, useT } from "@/i18n/i18n"
import { api } from "@/lib/api"

export function BoardHeatmap() {
  const t = useT()
  const locale = useLocale()
  const { data, loading, reload } = useLoad(() => api.boardHeatmap(), [])

  useEvents((ev) => {
    if (ev.type === "forgejo") void reload()
  })

  if (loading && !data) {
    return <Skeleton className="h-36 w-full rounded-md" />
  }
  if (!data) return null

  return (
    <section className="rounded-md border px-3 py-3">
      <p className="text-sm text-muted-foreground">{t("heatTotal", { n: data.total.toLocaleString(locale) })}</p>
      <Heatmap
        className="mt-3"
        days={data.days}
        locale={locale}
        fewerLabel={t("heatFewer")}
        moreLabel={t("heatMore")}
        renderTooltip={(cell) => {
          const date = new Intl.DateTimeFormat(locale.startsWith("zh") ? "zh-CN" : "en-US", {
            dateStyle: "long",
          }).format(new Date(`${cell.date}T00:00:00`))
          if (cell.value <= 0) return t("heatNone", { date })
          if (cell.value === 1) return t("heatOne", { date })
          return t("heatDay", { date, n: cell.value })
        }}
      />
    </section>
  )
}
