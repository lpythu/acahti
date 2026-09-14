import { useEffect, useMemo, useState } from "react"
import { Link, useLocation, useSearchParams } from "react-router-dom"
import { BoxIcon, ChevronRightIcon, FolderIcon } from "lucide-react"
import { cn } from "cn"

import { MoreButton } from "@/components/paged-list"
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from "@/components/ui/collapsible"
import {
  SidebarGroup,
  SidebarGroupContent,
  SidebarGroupLabel,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarMenuSub,
  SidebarMenuSubButton,
  SidebarMenuSubItem,
} from "@/components/ui/sidebar"
import { usePage } from "@/hooks/use-page"
import { useT } from "@/i18n/i18n"
import { api, type PackageRow } from "@/lib/api"
import { packageHref, parsePackagePath } from "@/lib/nav"

function KindNode({
  kind,
  names,
  pathname,
  filterKind,
  defaultOpen,
}: {
  kind: string
  names: string[]
  pathname: string
  filterKind: string
  defaultOpen: boolean
}) {
  const current = parsePackagePath(pathname)
  const inKind = current?.kind === kind
  const kindActive = pathname === "/packages" && filterKind === kind
  const [open, setOpen] = useState(defaultOpen || inKind || kindActive)

  useEffect(() => {
    if (inKind || kindActive) setOpen(true)
  }, [inKind, kindActive])

  return (
    <SidebarMenuItem>
      <Collapsible open={open} onOpenChange={setOpen}>
        <div className="flex min-w-0 items-center">
          <SidebarMenuButton
            tooltip={kind}
            isActive={kindActive}
            className="flex-1"
            render={<Link to={`/packages?kind=${encodeURIComponent(kind)}`} />}
          >
            <FolderIcon />
            <span>{kind}</span>
          </SidebarMenuButton>
          <CollapsibleTrigger
            className={cn(
              "flex size-8 shrink-0 items-center justify-center rounded-md text-sidebar-foreground ring-sidebar-ring outline-hidden hover:bg-sidebar-accent hover:text-sidebar-accent-foreground focus-visible:ring-2 group-data-[collapsible=icon]:hidden",
            )}
            aria-label={kind}
          >
            <ChevronRightIcon className={cn("size-4 transition-transform", open && "rotate-90")} />
          </CollapsibleTrigger>
        </div>
        <CollapsibleContent>
          <SidebarMenuSub>
            {names.map((name) => {
              const url = packageHref(kind, name)
              const active = current?.kind === kind && current.name === name
              return (
                <SidebarMenuSubItem key={name}>
                  <SidebarMenuSubButton isActive={active} render={<Link to={url} />}>
                    <BoxIcon />
                    <span>{name}</span>
                  </SidebarMenuSubButton>
                </SidebarMenuSubItem>
              )
            })}
          </SidebarMenuSub>
        </CollapsibleContent>
      </Collapsible>
    </SidebarMenuItem>
  )
}

function groupByKind(pkgs: PackageRow[]): { kind: string; names: string[] }[] {
  const map = new Map<string, Set<string>>()
  for (const p of pkgs) {
    const set = map.get(p.type) || new Set<string>()
    set.add(p.name)
    map.set(p.type, set)
  }
  return [...map.entries()]
    .sort(([a], [b]) => a.localeCompare(b))
    .map(([kind, names]) => ({ kind, names: [...names].sort((a, b) => a.localeCompare(b)) }))
}

export function PackagesNavPanel() {
  const t = useT()
  const { pathname } = useLocation()
  const [sp] = useSearchParams()
  const filterKind = sp.get("kind") || ""
  const list = usePage((q) => api.packages(q), [], { url: false })
  const groups = useMemo(() => groupByKind(list.items), [list.items])
  const current = parsePackagePath(pathname)
  const openAll = groups.length <= 3 && !filterKind && !current

  if (!groups.length) return null

  return (
    <SidebarGroup>
      <SidebarGroupLabel>{t("packageKind")}</SidebarGroupLabel>
      <SidebarGroupContent>
        <SidebarMenu>
          {groups.map((g) => (
            <KindNode
              key={g.kind}
              kind={g.kind}
              names={g.names}
              pathname={pathname}
              filterKind={filterKind}
              defaultOpen={openAll || g.kind === current?.kind}
            />
          ))}
          <MoreButton page={list.page} hasMore={list.hasMore} onPage={list.setPage} />
        </SidebarMenu>
      </SidebarGroupContent>
    </SidebarGroup>
  )
}
