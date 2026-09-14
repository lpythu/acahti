import type { ReactNode } from "react"
import { useLocation, useSearchParams } from "react-router-dom"
import {
  FileIcon,
  GitBranchIcon,
  GitPullRequestIcon,
  HistoryIcon,
  PanelLeftCloseIcon,
  PanelLeftIcon,
  ShieldIcon,
  UsersIcon,
  WorkflowIcon,
} from "lucide-react"
import { cn } from "cn"

import { NavMain } from "@/components/nav-main"
import { PackagesNavPanel } from "@/components/packages-nav-panel"
import { PipelinesNavPanel } from "@/components/pipelines-nav-panel"
import { ReposNavPanel } from "@/components/repos-nav-panel"
import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  useSidebar,
} from "@/components/ui/sidebar"
import { useT } from "@/i18n/i18n"
import { repoBase, sectionOf } from "@/lib/nav"

function SidebarCollapseToggle() {
  const t = useT()
  const { state, toggleSidebar } = useSidebar()
  const collapsed = state === "collapsed"

  return (
    <SidebarMenu>
      <SidebarMenuItem>
        <SidebarMenuButton
          type="button"
          tooltip={t("toggleSidebar")}
          onClick={toggleSidebar}
          aria-label={t("toggleSidebar")}
          className="size-8! w-8! p-2!"
        >
          {collapsed ? <PanelLeftIcon /> : <PanelLeftCloseIcon />}
        </SidebarMenuButton>
      </SidebarMenuItem>
    </SidebarMenu>
  )
}

function secondaryItems(
  path: string,
  ref: string,
  t: (k: "files" | "commits" | "branches" | "tabPulls" | "tabPipes" | "adminHome" | "users") => string,
): { title: string; url: string; icon?: ReactNode; end?: boolean }[] {
  const base = repoBase(path)
  const q = ref ? `?ref=${encodeURIComponent(ref)}` : ""
  if (base) {
    return [
      { title: t("files"), url: `${base}${q}`, icon: <FileIcon />, end: true },
      { title: t("commits"), url: `${base}/commits${q}`, icon: <HistoryIcon /> },
      { title: t("branches"), url: `${base}/branches`, icon: <GitBranchIcon /> },
      { title: t("tabPulls"), url: `${base}/pulls`, icon: <GitPullRequestIcon /> },
      { title: t("tabPipes"), url: `${base}/pipelines`, icon: <WorkflowIcon /> },
    ]
  }
  if (path.startsWith("/admin")) {
    return [
      { title: t("adminHome"), url: "/admin", icon: <ShieldIcon />, end: true },
      { title: t("users"), url: "/admin/users", icon: <UsersIcon /> },
    ]
  }
  return []
}

export function AppSidebar({ ...props }: React.ComponentProps<typeof Sidebar>) {
  const t = useT()
  const { pathname } = useLocation()
  const [sp] = useSearchParams()
  const items = secondaryItems(pathname, sp.get("ref") || "", t)
  const { isMobile } = useSidebar()

  return (
    <Sidebar
      collapsible="icon"
      variant="inset"
      {...props}
      className={cn("top-12 h-[calc(100svh-3rem)]", props.className)}
    >
      <SidebarContent>
        {sectionOf(pathname) === "repos" && !repoBase(pathname) ? <ReposNavPanel /> : null}
        {sectionOf(pathname) === "pipelines" ? <PipelinesNavPanel /> : null}
        {sectionOf(pathname) === "packages" ? <PackagesNavPanel /> : null}
        {items.length ? <NavMain items={items} active={pathname} /> : null}
      </SidebarContent>
      {isMobile ? null : (
        <SidebarFooter className="mt-auto p-2">
          <SidebarCollapseToggle />
        </SidebarFooter>
      )}
    </Sidebar>
  )
}

export function sidebarHasNav(path: string): boolean {
  return (
    path.startsWith("/repos") ||
    path.startsWith("/pipelines") ||
    path.startsWith("/packages") ||
    path.startsWith("/admin")
  )
}
