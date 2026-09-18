"use client"

import { useEffect, useState } from "react"
import { Tooltip as TooltipPrimitive } from "@base-ui/react/tooltip"
import { cn } from "cn"

const SKIP = "input,textarea,select,[data-slot=tooltip-content]"

function fullText(el: HTMLElement) {
  return (el.innerText || "").replace(/\s+/g, " ").trim()
}

function titled(el: HTMLElement) {
  for (let node: HTMLElement | null = el; node; node = node.parentElement) {
    if (node.title.trim()) return true
  }
  return false
}

function isEllipsis(el: HTMLElement) {
  if (!el.isConnected || el.matches(SKIP) || el.closest(SKIP) || titled(el)) return false
  const css = getComputedStyle(el)
  if (css.display === "none" || css.visibility === "hidden") return false
  if (!el.getClientRects().length) return false
  const clamped = css.webkitLineClamp !== "none" && css.webkitLineClamp !== ""
  if (css.textOverflow === "ellipsis") {
    return el.clientWidth > 0 && el.scrollWidth - el.clientWidth > 1
  }
  return clamped && el.clientHeight > 0 && el.scrollHeight - el.clientHeight > 1
}

function findEllipsis(from: EventTarget | null): HTMLElement | null {
  if (!(from instanceof Element) || from.closest("[data-slot=tooltip-content]")) return null
  let node: HTMLElement | null = from instanceof HTMLElement ? from : from.parentElement
  while (node && node !== document.body) {
    if (isEllipsis(node)) return node
    const { overflowX, overflowY } = getComputedStyle(node)
    if (overflowX !== "visible" || overflowY !== "visible") {
      for (const child of node.children) {
        if (child instanceof HTMLElement && isEllipsis(child)) return child
      }
    }
    node = node.parentElement
  }
  return null
}

function OverflowTooltip() {
  const [anchor, setAnchor] = useState<HTMLElement | null>(null)
  const text = anchor ? fullText(anchor) : ""

  useEffect(() => {
    let current: HTMLElement | null = null
    const show = (el: HTMLElement | null) => {
      if (el === current) return
      current = el
      setAnchor(el)
    }
    const onOver = (e: PointerEvent) => {
      if (e.pointerType === "touch") return
      if (e.target instanceof Element && e.target.closest("[data-slot=tooltip-content]")) return
      show(findEllipsis(e.target))
    }
    const hide = () => show(null)
    document.addEventListener("pointerover", onOver, true)
    document.addEventListener("pointerdown", hide, true)
    window.addEventListener("scroll", hide, true)
    window.addEventListener("resize", hide)
    return () => {
      document.removeEventListener("pointerover", onOver, true)
      document.removeEventListener("pointerdown", hide, true)
      window.removeEventListener("scroll", hide, true)
      window.removeEventListener("resize", hide)
    }
  }, [])

  if (!anchor || !text) return null

  return (
    <Tooltip open disableHoverablePopup>
      <TooltipContent anchor={anchor} className="pointer-events-none max-w-xs break-all">
        {text}
      </TooltipContent>
    </Tooltip>
  )
}

function TooltipProvider({
  delay = 0,
  children,
  ...props
}: TooltipPrimitive.Provider.Props) {
  return (
    <TooltipPrimitive.Provider data-slot="tooltip-provider" delay={delay} {...props}>
      {children}
      <OverflowTooltip />
    </TooltipPrimitive.Provider>
  )
}

function Tooltip({ ...props }: TooltipPrimitive.Root.Props) {
  return <TooltipPrimitive.Root data-slot="tooltip" {...props} />
}

function TooltipTrigger({ ...props }: TooltipPrimitive.Trigger.Props) {
  return <TooltipPrimitive.Trigger data-slot="tooltip-trigger" {...props} />
}

function TooltipContent({
  className,
  side = "top",
  sideOffset = 4,
  align = "center",
  alignOffset = 0,
  anchor,
  children,
  ...props
}: TooltipPrimitive.Popup.Props &
  Pick<
    TooltipPrimitive.Positioner.Props,
    "align" | "alignOffset" | "anchor" | "side" | "sideOffset"
  >) {
  return (
    <TooltipPrimitive.Portal>
      <TooltipPrimitive.Positioner
        align={align}
        alignOffset={alignOffset}
        anchor={anchor}
        side={side}
        sideOffset={sideOffset}
        className="isolate z-50"
      >
        <TooltipPrimitive.Popup
          data-slot="tooltip-content"
          className={cn(
            "z-50 inline-flex w-fit max-w-xs origin-(--transform-origin) items-center gap-1.5 rounded-md bg-foreground px-3 py-1.5 text-xs text-background has-data-[slot=kbd]:pr-1.5 data-[side=bottom]:slide-in-from-top-2 data-[side=inline-end]:slide-in-from-left-2 data-[side=inline-start]:slide-in-from-right-2 data-[side=left]:slide-in-from-right-2 data-[side=right]:slide-in-from-left-2 data-[side=top]:slide-in-from-bottom-2 **:data-[slot=kbd]:relative **:data-[slot=kbd]:isolate **:data-[slot=kbd]:z-50 **:data-[slot=kbd]:rounded-sm data-[state=delayed-open]:animate-in data-[state=delayed-open]:fade-in-0 data-[state=delayed-open]:zoom-in-95 data-open:animate-in data-open:fade-in-0 data-open:zoom-in-95 data-closed:animate-out data-closed:fade-out-0 data-closed:zoom-out-95",
            className
          )}
          {...props}
        >
          {children}
          <TooltipPrimitive.Arrow className="z-50 size-2.5 translate-y-[calc(-50%-2px)] rotate-45 rounded-[2px] bg-foreground fill-foreground data-[side=bottom]:top-1 data-[side=inline-end]:top-1/2! data-[side=inline-end]:-left-1 data-[side=inline-end]:-translate-y-1/2 data-[side=inline-start]:top-1/2! data-[side=inline-start]:-right-1 data-[side=inline-start]:-translate-y-1/2 data-[side=left]:top-1/2! data-[side=left]:-right-1 data-[side=left]:-translate-y-1/2 data-[side=right]:top-1/2! data-[side=right]:-left-1 data-[side=right]:-translate-y-1/2 data-[side=top]:-bottom-2.5" />
        </TooltipPrimitive.Popup>
      </TooltipPrimitive.Positioner>
    </TooltipPrimitive.Portal>
  )
}

export { Tooltip, TooltipTrigger, TooltipContent, TooltipProvider }
