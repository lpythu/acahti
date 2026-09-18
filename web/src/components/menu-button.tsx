import type { ReactNode } from "react"
import type { VariantProps } from "class-variance-authority"

import { Button, buttonVariants } from "@/components/ui/button"
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover"

type ButtonProps = VariantProps<typeof buttonVariants>

export function MenuButton({
  label,
  variant = "default",
  size = "sm",
  align = "end",
  side = "bottom",
  className,
  disabled,
  open,
  onOpenChange,
  children,
}: {
  label: ReactNode
  variant?: ButtonProps["variant"]
  size?: ButtonProps["size"]
  align?: "start" | "center" | "end"
  side?: "top" | "bottom" | "left" | "right"
  className?: string
  disabled?: boolean
  open?: boolean
  onOpenChange?: (open: boolean) => void
  children: ReactNode
}) {
  return (
    <Popover open={open} onOpenChange={onOpenChange}>
      <PopoverTrigger render={<Button type="button" variant={variant} size={size} disabled={disabled} />}>
        {label}
      </PopoverTrigger>
      <PopoverContent align={align} side={side} className={className}>
        {children}
      </PopoverContent>
    </Popover>
  )
}
