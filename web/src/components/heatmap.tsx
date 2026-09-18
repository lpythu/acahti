import { useMemo, type ReactNode } from "react"
import { cn } from "cn"

export type HeatDay = { date: string; value: number }

type HeatCell = { date: string; value: number }
type HeatColumn = { key: string; label: string }

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

  return (
    <div className={cn("flex flex-col gap-3", className)}>
      <div className="w-full min-w-0" style={{ containerType: "inline-size" }}>
        <div
          className="grid w-full min-w-0 items-start"
          style={{
            gap: `clamp(1px, calc((100cqw - 32px) / ${grid.columns.length} * 0.18), 6px)`,
            gridTemplateColumns: `32px repeat(${grid.columns.length}, minmax(0, 1fr))`,
            gridTemplateRows: `16px repeat(7, auto)`,
          }}
        >
          <span />
          {grid.columns.map((col) => (
            <span key={col.key} className="text-muted-foreground overflow-visible whitespace-nowrap text-[11px] leading-none">
              {col.label}
            </span>
          ))}
          {grid.rows.map((row, rowIndex) => (
            <Row
              key={row}
              row={row}
              showLabel={rowIndex === 0 || rowIndex === 2 || rowIndex === 4}
              columns={grid.columns}
              values={grid.values}
              dates={grid.dates}
              levels={levels}
              renderTooltip={renderTooltip}
            />
          ))}
        </div>
      </div>
      <div className="text-muted-foreground flex items-center justify-end gap-1.5 text-xs">
        {fewerLabel ? <span>{fewerLabel}</span> : null}
        {COLORS.map((color) => (
          <span key={color} aria-hidden className="inline-block size-3 rounded-[3px]" style={{ backgroundColor: color }} />
        ))}
        {moreLabel ? <span>{moreLabel}</span> : null}
      </div>
    </div>
  )
}

function Row({
  row,
  showLabel,
  columns,
  values,
  dates,
  levels,
  renderTooltip,
}: {
  row: string
  showLabel: boolean
  columns: HeatColumn[]
  values: Map<string, number>
  dates: Map<string, string>
  levels: number[]
  renderTooltip?: (cell: HeatCell) => string
}) {
  return (
    <>
      <span className="text-muted-foreground flex self-stretch items-center justify-end pr-1 text-[11px] leading-none">
        {showLabel ? row : ""}
      </span>
      {columns.map((col) => {
        const key = cellKey(col.key, row)
        const value = values.get(key) ?? 0
        const date = dates.get(key) ?? ""
        return (
          <div
            key={key}
            role="img"
            aria-label={date ? `${date} ${value}` : undefined}
            className="aspect-square w-full min-w-0"
            style={{
              backgroundColor: COLORS[levelFor(value, levels)],
              borderRadius: "min(3px, 20%)",
            }}
            title={renderTooltip ? renderTooltip({ date, value }) : date}
          />
        )
      })}
    </>
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
  return now.toISOString().slice(0, 10)
}

function parseDay(value: string): Date | null {
  if (!value) return null
  const date = new Date(`${value}T00:00:00`)
  return Number.isNaN(date.getTime()) ? null : date
}
