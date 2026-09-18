import { useMemo } from "react"
import { CheckIcon, ChevronsUpDownIcon } from "lucide-react"
import { toast } from "sonner"

import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { cn } from "cn"
import { useT } from "@/i18n/i18n"
import { api, type Org } from "@/lib/api"

export function orgLabel(org: string, fullName?: string) {
  const name = org.trim()
  const full = fullName?.trim()
  if (full && full.toLowerCase() !== name.toLowerCase()) return full
  return name ? name.toUpperCase() : ""
}

export function OrgSwitcher({
  className,
  org,
  orgs = [],
}: {
  className?: string
  org: string
  orgs?: Org[]
}) {
  const t = useT()
  const list = useMemo(() => {
    if (orgs.length) return orgs
    return org ? [{ name: org, full_name: org }] : []
  }, [org, orgs])
  const current = list.find((item) => item.name === org) ?? list[0]
  const title = orgLabel(current?.name || org, current?.full_name)

  async function switchTo(name: string) {
    if (!name || name === org) return
    try {
      await api.switchOrg(name)
      window.location.reload()
    } catch (err) {
      toast.error(err instanceof Error ? err.message : t("orgSwitchFailed"))
    }
  }

  return (
    <DropdownMenu>
      <DropdownMenuTrigger
        className={cn(
          "flex h-9 w-full min-w-0 items-center justify-between gap-3 rounded-md py-0 text-left outline-none hover:bg-accent",
          "focus-visible:ring-2 focus-visible:ring-ring",
          className,
        )}
      >
        <span className="min-w-0 truncate text-sm font-semibold tracking-tight">{title}</span>
        <ChevronsUpDownIcon className="size-4 shrink-0 text-muted-foreground opacity-70" />
      </DropdownMenuTrigger>
      <DropdownMenuContent align="start" className="min-w-56">
        {list.length === 0 ? (
          <DropdownMenuItem disabled className="text-muted-foreground">
            {t("orgEmpty")}
          </DropdownMenuItem>
        ) : (
          list.map((item) => {
            const active = item.name === (current?.name || org)
            const label = orgLabel(item.name, item.full_name)
            return (
              <DropdownMenuItem key={item.name} className="gap-2" onClick={() => void switchTo(item.name)}>
                <div className="min-w-0 flex-1">
                  <div className="truncate text-sm font-medium">{label}</div>
                  {label.toLowerCase() !== item.name.toLowerCase() ? (
                    <div className="truncate text-xs text-muted-foreground">{item.name}</div>
                  ) : null}
                </div>
                {active ? <CheckIcon className="size-4 shrink-0" /> : null}
              </DropdownMenuItem>
            )
          })
        )}
      </DropdownMenuContent>
    </DropdownMenu>
  )
}
