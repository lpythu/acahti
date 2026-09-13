import { Link, useLocation } from "react-router-dom"
import type { ReactNode } from "react"
import { cn } from "cn"

import { NavMain } from "@/components/nav-main"
import { AcahtiMark } from "@/components/logo"
import {
  Sidebar,
  SidebarContent,
  SidebarHeader,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
} from "@/components/ui/sidebar"
import { useT } from "@/i18n/i18n"

export function AppSidebar({
  area,
  items,
  home,
  ...props
}: React.ComponentProps<typeof Sidebar> & {
  area: "console" | "admin"
  items: { title: string; url: string; icon?: ReactNode }[]
  home: string
}) {
  const t = useT()
  const loc = useLocation()
  return (
    <Sidebar
      collapsible="offcanvas"
      {...props}
      className={cn(
        "top-(--header-height) h-[calc(100svh-var(--header-height))]",
        props.className,
      )}
    >
      <SidebarHeader>
        <SidebarMenu>
          <SidebarMenuItem>
            <SidebarMenuButton
              className="data-[slot=sidebar-menu-button]:p-1.5!"
              render={<Link to={home} />}
            >
              <AcahtiMark className="size-5 text-foreground" />
              <span className="text-base font-semibold">
                {t("brand")}
                {area === "admin" ? ` · ${t("admin")}` : ""}
              </span>
            </SidebarMenuButton>
          </SidebarMenuItem>
        </SidebarMenu>
      </SidebarHeader>
      <SidebarContent>
        <NavMain items={items} active={loc.pathname} />
      </SidebarContent>
    </Sidebar>
  )
}
