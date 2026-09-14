import {
  useEffect,
  useRef,
  useState,
  type ReactNode,
  type Ref,
  type RefObject,
  type UIEvent,
  type UIEventHandler,
} from "react"

import { cn } from "cn"

type AxisThumb = {
  offset: number
  size: number
  needed: boolean
}

export type OverlayThumbMetrics = {
  v: AxisThumb
  h: AxisThumb
}

const EMPTY_AXIS: AxisThumb = { offset: 0, size: 0, needed: false }
const EMPTY_THUMBS: OverlayThumbMetrics = { v: EMPTY_AXIS, h: EMPTY_AXIS }
const THUMB_MIN = 24
const TRACK_INSET = 8

function axisThumb(
  scrollPos: number,
  scrollSize: number,
  clientSize: number,
  trackSize: number,
): AxisThumb {
  if (scrollSize <= clientSize + 1) return EMPTY_AXIS
  const size = Math.max(THUMB_MIN, (clientSize / scrollSize) * trackSize)
  const maxOffset = trackSize - size
  const range = scrollSize - clientSize
  const offset =
    maxOffset <= 0 || range <= 0 ? 0 : (scrollPos / range) * maxOffset
  return { offset, size, needed: true }
}

function readThumbs(el: HTMLElement): OverlayThumbMetrics {
  const {
    scrollTop,
    scrollHeight,
    clientHeight,
    scrollLeft,
    scrollWidth,
    clientWidth,
  } = el
  const vNeeded = scrollHeight > clientHeight + 1
  const hNeeded = scrollWidth > clientWidth + 1
  const vTrack = Math.max(0, clientHeight - (hNeeded ? TRACK_INSET : 0))
  const hTrack = Math.max(0, clientWidth - (vNeeded ? TRACK_INSET : 0))
  return {
    v: axisThumb(scrollTop, scrollHeight, clientHeight, vTrack),
    h: axisThumb(scrollLeft, scrollWidth, clientWidth, hTrack),
  }
}

function assignRef<T>(ref: Ref<T> | undefined, value: T | null) {
  if (!ref) return
  if (typeof ref === "function") {
    ref(value)
    return
  }
  ref.current = value
}

function thumbsEqual(a: OverlayThumbMetrics, b: OverlayThumbMetrics) {
  return (
    a.v.offset === b.v.offset &&
    a.v.size === b.v.size &&
    a.v.needed === b.v.needed &&
    a.h.offset === b.h.offset &&
    a.h.size === b.h.size &&
    a.h.needed === b.h.needed
  )
}

function axisOverflows(el: HTMLElement, vertical: boolean) {
  return vertical
    ? el.scrollHeight > el.clientHeight + 1
    : el.scrollWidth > el.clientWidth + 1
}

function isScrollableOnAxis(el: HTMLElement, vertical: boolean) {
  const style = getComputedStyle(el)
  const overflow = vertical ? style.overflowY : style.overflowX
  return /(auto|scroll)/.test(overflow) && axisOverflows(el, vertical)
}

function findParentScroller(el: HTMLElement, vertical: boolean) {
  let node = el.parentElement
  while (node) {
    if (isScrollableOnAxis(node, vertical)) return node
    node = node.parentElement
  }
  return null
}

const HIGHLIGHT_SEL =
  "[data-highlighted], [data-slot='command-item'][data-selected='true']"
const HIGHLIGHT_EDGE = 8

function highlightedItem(root: HTMLElement): HTMLElement | null {
  return root.querySelector<HTMLElement>(HIGHLIGHT_SEL)
}

function scrollHighlightedIntoScroller(
  scroller: HTMLElement,
  item: HTMLElement,
  behavior: ScrollBehavior,
) {
  const pane = scroller.getBoundingClientRect()
  const row = item.getBoundingClientRect()
  let delta = 0
  if (row.height > pane.height) {
    delta = row.top - (pane.top + HIGHLIGHT_EDGE)
  } else if (row.bottom > pane.bottom - HIGHLIGHT_EDGE) {
    delta = row.bottom - (pane.bottom - HIGHLIGHT_EDGE)
  } else if (row.top < pane.top + HIGHLIGHT_EDGE) {
    delta = row.top - (pane.top + HIGHLIGHT_EDGE)
  }
  if (Math.abs(delta) < 1) return
  scroller.scrollBy({ top: delta, left: 0, behavior })
}

function wheelPixels(event: WheelEvent, fallback: number) {
  const scale =
    event.deltaMode === 1 ? 16 : event.deltaMode === 2 ? fallback : 1
  return { dx: event.deltaX * scale, dy: event.deltaY * scale }
}

/** Overlay thumbs for a scroller we do not own (select list, owned viewport). */
export function useOverlayScrollThumbs(
  elRef: RefObject<HTMLElement | null>,
  opts?: { ignoreUntrusted?: boolean },
) {
  const hideTimer = useRef(0)
  const mountedRef = useRef(true)
  const [thumb, setThumb] = useState<OverlayThumbMetrics>(EMPTY_THUMBS)
  const [active, setActive] = useState(false)

  useEffect(() => {
    mountedRef.current = true
    const el = elRef.current
    if (!el) return

    const sync = () => {
      if (!mountedRef.current) return
      const next = readThumbs(el)
      setThumb((prev) => (thumbsEqual(prev, next) ? prev : next))
    }
    const flash = (event?: Event) => {
      if (!mountedRef.current) return
      if (opts?.ignoreUntrusted && event && !event.isTrusted) return
      sync()
      setActive(true)
      window.clearTimeout(hideTimer.current)
      hideTimer.current = window.setTimeout(() => {
        if (mountedRef.current) setActive(false)
      }, 800)
    }

    el.addEventListener("scroll", flash, { passive: true })
    sync()
    const ro = new ResizeObserver(sync)
    ro.observe(el)
    const child = el.firstElementChild
    if (child) ro.observe(child)
    return () => {
      mountedRef.current = false
      el.removeEventListener("scroll", flash)
      ro.disconnect()
      window.clearTimeout(hideTimer.current)
    }
  }, [elRef, opts?.ignoreUntrusted])

  return { thumb, active }
}

/** Floating thumbs on the host's inner edges. Host must be `relative`. */
export function OverlayScrollThumbs({
  thumb,
  active,
}: {
  thumb: OverlayThumbMetrics
  active: boolean
}) {
  return (
    <>
      {thumb.v.needed ? (
        <div
          aria-hidden
          className={cn(
            "pointer-events-none absolute inset-y-0 right-0 z-10 w-2 transition-opacity duration-200",
            active ? "opacity-100" : "opacity-0",
          )}
        >
          <div
            className="bg-muted-foreground/45 absolute right-0.5 w-1.5 rounded-full"
            style={{ top: thumb.v.offset, height: thumb.v.size }}
          />
        </div>
      ) : null}
      {thumb.h.needed ? (
        <div
          aria-hidden
          className={cn(
            "pointer-events-none absolute inset-x-0 bottom-0 z-10 h-2 transition-opacity duration-200",
            active ? "opacity-100" : "opacity-0",
          )}
        >
          <div
            className="bg-muted-foreground/45 absolute bottom-0.5 h-1.5 rounded-full"
            style={{ left: thumb.h.offset, width: thumb.h.size }}
          />
        </div>
      ) : null}
    </>
  )
}

/**
 * Default overflow scroller. Native bars are hidden (no gutter / padding steal).
 * Thumbs float on the inner edges while scrolling.
 */
export function AutoHideScroll({
  className,
  onScroll,
  scrollRef,
  children,
}: {
  className?: string
  onScroll?: UIEventHandler<HTMLDivElement>
  scrollRef?: Ref<HTMLDivElement | null>
  children: ReactNode
}) {
  const localRef = useRef<HTMLDivElement>(null)
  const { thumb, active } = useOverlayScrollThumbs(localRef)

  useEffect(() => {
    const el = localRef.current
    if (!el) return
    const onWheel = (event: WheelEvent) => {
      if (event.ctrlKey) return
      const { dx, dy } = wheelPixels(event, el.clientHeight)
      const vertical = Math.abs(dy) >= Math.abs(dx)
      if (axisOverflows(el, vertical)) return
      const parent = findParentScroller(el, vertical)
      if (!parent) return
      event.preventDefault()
      parent.scrollBy({
        top: vertical ? dy : 0,
        left: vertical ? 0 : dx,
        behavior: "auto",
      })
    }
    el.addEventListener("wheel", onWheel, { passive: false })
    return () => el.removeEventListener("wheel", onWheel)
  }, [])

  useEffect(() => {
    const el = localRef.current
    if (!el) return
    let lastAt = 0
    let raf = 0
    const follow = () => {
      window.cancelAnimationFrame(raf)
      raf = window.requestAnimationFrame(() => {
        const item = highlightedItem(el)
        if (!item) return
        const now = performance.now()
        const rapid = now - lastAt < 90
        lastAt = now
        scrollHighlightedIntoScroller(el, item, rapid ? "auto" : "smooth")
      })
    }
    const mo = new MutationObserver((records) => {
      for (const rec of records) {
        if (rec.type !== "attributes") continue
        const name = rec.attributeName
        if (name === "data-highlighted" || name === "data-selected") {
          follow()
          return
        }
      }
    })
    mo.observe(el, {
      attributes: true,
      attributeFilter: ["data-highlighted", "data-selected"],
      subtree: true,
    })
    return () => {
      mo.disconnect()
      window.cancelAnimationFrame(raf)
    }
  }, [])

  function setRefs(node: HTMLDivElement | null) {
    localRef.current = node
    assignRef(scrollRef, node)
  }

  function handleScroll(e: UIEvent<HTMLDivElement>) {
    onScroll?.(e)
  }

  return (
    <div
      className={cn(
        "relative flex min-h-0 min-w-0 flex-col overflow-hidden",
        className,
      )}
    >
      <div
        ref={setRefs}
        onScroll={handleScroll}
        className="scrollbar-auto-hide-overlay min-h-0 min-w-0 flex-1 overflow-auto overscroll-contain"
      >
        {children}
      </div>
      <OverlayScrollThumbs thumb={thumb} active={active} />
    </div>
  )
}
