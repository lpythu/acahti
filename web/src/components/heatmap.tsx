import { useLayoutEffect, useMemo, useRef, useState, type ReactNode } from "react"
import { createPortal } from "react-dom"
import { cn } from "cn"

export type HeatDay = { date: string; value: number }

type HeatCell = { date: string; value: number }

const COLORS = ["var(--heat-0)", "var(--heat-1)", "var(--heat-2)", "var(--heat-3)", "var(--heat-4)"] as const
const WEEKS = 53

export function Heatmap({
  days,
  locale = "en",
  fewerLabel,
  moreLabel,
  renderTooltip,
  className,
}: {
  days: readonly HeatDay[]
  locale?: string
  fewerLabel?: ReactNode
  moreLabel?: ReactNode
  renderTooltip?: (cell: HeatCell) => string
  className?: string
}) {
  const grid = useMemo(() => calendarGrid(days, locale), [days, locale])
  const levels = useMemo(() => heatLevels(grid.values), [grid.values])
  const n = grid.columns.length
  const [tip, setTip] = useState<{ text: string; left: number; top: number } | null>(null)
  const tipRef = useRef<HTMLDivElement>(null)

  useLayoutEffect(() => {
    const el = tipRef.current
    if (!el || !tip) return
    const r = el.getBoundingClientRect()
    const pad = 8
    let shift = 0
    if (r.left < pad) shift = pad - r.left
    else if (r.right > window.innerWidth - pad) shift = window.innerWidth - pad - r.right
    if (shift) el.style.left = `${tip.left + shift}px`
  }, [tip])

  return (
    <div className={cn("flex flex-col gap-3", className)}>
      <div className="flex min-w-0">
        <div className="w-8 shrink-0" />
        <div
          className="text-muted-foreground mb-1 grid min-w-0 flex-1 text-[11px] leading-none"
          style={{ gridTemplateColumns: `repeat(${n}, minmax(0, 1fr))` }}
        >
          {grid.columns.map((col) => (
            <span key={col.key} className="overflow-hidden whitespace-nowrap">
              {col.label}
            </span>
          ))}
        </div>
      </div>
      <div className="flex min-w-0 items-start">
        <div className="text-muted-foreground grid w-8 shrink-0 grid-rows-7 self-stretch pr-1 text-[11px] leading-none">
          {grid.rows.map((row, rowIndex) => (
            <span key={row} className="flex items-center justify-end">
              {rowIndex === 0 || rowIndex === 2 || rowIndex === 4 ? row : ""}
            </span>
          ))}
        </div>
        <div
          className="grid min-w-0 flex-1"
          style={{ gridTemplateColumns: `repeat(${n}, minmax(0, 1fr))` }}
        >
          {grid.rows.flatMap((row) =>
            grid.columns.map((col) => {
              const key = cellKey(col.key, row)
              const value = grid.values.get(key) ?? 0
              const date = grid.dates.get(key) ?? ""
              const text = date && renderTooltip ? renderTooltip({ date, value }) : ""
              return (
                <div
                  key={key}
                  className="aspect-square min-w-0"
                  style={{ padding: "max(0.5px, 12.5%)" }}
                  onPointerEnter={(e) => {
                    if (!text) return
                    const r = e.currentTarget.getBoundingClientRect()
                    setTip({ text, left: r.left + r.width / 2, top: r.top })
                  }}
                  onPointerLeave={() => setTip(null)}
                >
                  <div
                    role="img"
                    aria-label={text || undefined}
                    className="size-full"
                    style={{
                      backgroundColor: COLORS[levelFor(value, levels)],
                      borderRadius: "min(3px, 20%)",
                    }}
                  />
                </div>
              )
            }),
          )}
        </div>
      </div>
      <div className="text-muted-foreground flex items-center justify-end gap-1.5 text-xs">
        {fewerLabel ? <span>{fewerLabel}</span> : null}
        {COLORS.map((color) => (
          <span key={color} aria-hidden className="inline-block size-3 rounded-[3px]" style={{ backgroundColor: color }} />
        ))}
        {moreLabel ? <span>{moreLabel}</span> : null}
      </div>
      {tip
        ? createPortal(
            <div
              ref={tipRef}
              role="tooltip"
              className="pointer-events-none fixed z-50 max-w-[calc(100vw-16px)] -translate-x-1/2 -translate-y-full whitespace-nowrap rounded-md bg-foreground px-3 py-1.5 text-xs text-background"
              style={{ left: tip.left, top: tip.top - 8 }}
            >
              {tip.text}
            </div>,
            document.body,
          )
        : null}
    </div>
  )
}

function calendarGrid(days: readonly HeatDay[], locale: string) {
  const weeks = padWeeks(chunkWeeks(days), WEEKS)
  const today = todayKey(new Date())
  const first = weeks.find((week) => week[0]?.date)?.[0]?.date
  const anchor = parseDay(first ?? "") ?? mondayAnchor()
  const rows = Array.from({ length: 7 }, (_, row) => weekdayLabel(row, locale, anchor))
  const columns = weeks.map((week, index) => ({
    key: week[0]?.date ?? `week-${index}`,
    label: weekMonthLabel(week, index === 0 || weeks.slice(0, index).every((item) => item.length === 0), locale),
  }))
  const values = new Map<string, number>()
  const dates = new Map<string, string>()
  weeks.forEach((week, weekIndex) => {
    const column = columns[weekIndex]?.key ?? `week-${weekIndex}`
    rows.forEach((row, rowIndex) => {
      const day = week[rowIndex]
      const key = cellKey(column, row)
      values.set(key, day && day.date <= today ? day.value : 0)
      if (day?.date) dates.set(key, day.date)
    })
  })
  return { rows, columns, values, dates }
}

function chunkWeeks(days: readonly HeatDay[]): HeatDay[][] {
  const weeks: HeatDay[][] = []
  for (let i = 0; i < days.length; i += 7) weeks.push([...days.slice(i, i + 7)])
  return weeks
}

function padWeeks(weeks: HeatDay[][], count: number): HeatDay[][] {
  if (weeks.length >= count) return weeks.slice(-count)
  return [...Array.from({ length: count - weeks.length }, () => [] as HeatDay[]), ...weeks]
}

function intlLocale(locale: string): string {
  return locale.startsWith("zh") ? "zh-CN" : "en-US"
}

function weekdayLabel(row: number, locale: string, anchor: Date): string {
  const date = new Date(anchor.getFullYear(), anchor.getMonth(), anchor.getDate() + row)
  return new Intl.DateTimeFormat(intlLocale(locale), { weekday: "short" }).format(date)
}

function mondayAnchor(): Date {
  return new Date(2026, 8, 7)
}

function weekMonthLabel(week: readonly HeatDay[], isFirst: boolean, locale: string): string {
  const firstOfMonth = week.find((day) => day.date.endsWith("-01"))
  const source = firstOfMonth ?? (isFirst ? week[0] : undefined)
  const date = parseDay(source?.date ?? "")
  if (!date) return ""
  return new Intl.DateTimeFormat(intlLocale(locale), { month: "short" }).format(date)
}

function heatLevels(values: Map<string, number>): number[] {
  let max = 0
  for (const value of values.values()) if (value > max) max = value
  const count = COLORS.length - 1
  const domain = Math.max(max, count)
  return Array.from({ length: count }, (_, index) => (domain * (index + 1)) / count)
}

function levelFor(value: number, levels: number[]): number {
  if (value <= 0) return 0
  const index = levels.findIndex((level) => value <= level)
  return index === -1 ? levels.length : index + 1
}

function cellKey(column: string, row: string): string {
  return `${column}\0${row}`
}

function todayKey(now: Date): string {
  const month = String(now.getMonth() + 1).padStart(2, "0")
  const day = String(now.getDate()).padStart(2, "0")
  return `${now.getFullYear()}-${month}-${day}`
}

function parseDay(value: string): Date | null {
  if (!value) return null
  const date = new Date(`${value}T00:00:00`)
  return Number.isNaN(date.getTime()) ? null : date
}
