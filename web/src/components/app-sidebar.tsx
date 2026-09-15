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
  LockIcon,
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

type NavItem = { title: string; url: string; icon?: ReactNode; end?: boolean; active?: boolean }

function secondaryItems(
  path: string,
  ref: string,
  section: string,
  t: (
    k:
      | "files"
      | "commits"
      | "branches"
      | "tabPulls"
      | "tabPipes"
      | "access"
      | "adminHome"
      | "users"
      | "pipelines"
      | "prs",
  ) => string,
): NavItem[] {
  const base = repoBase(path)
  const q = ref ? `?ref=${encodeURIComponent(ref)}` : ""
  if (base) {
    return [
      { title: t("files"), url: `${base}${q}`, icon: <FileIcon />, end: true },
      { title: t("commits"), url: `${base}/commits${q}`, icon: <HistoryIcon /> },
      { title: t("branches"), url: `${base}/branches`, icon: <GitBranchIcon /> },
      { title: t("tabPulls"), url: `${base}/pulls`, icon: <GitPullRequestIcon /> },
      { title: t("tabPipes"), url: `${base}/pipelines`, icon: <WorkflowIcon /> },
      { title: t("access"), url: `${base}/access`, icon: <LockIcon /> },
    ]
  }
  if (path === "/board") {
    return [
      {
        title: t("pipelines"),
        url: "/board",
        icon: <WorkflowIcon />,
        end: true,
        active: section !== "prs",
      },
      {
        title: t("prs"),
        url: "/board?section=prs",
        icon: <GitPullRequestIcon />,
        active: section === "prs",
      },
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
  const { pathname, search } = useLocation()
  const [sp] = useSearchParams()
  const items = secondaryItems(pathname, sp.get("ref") || "", sp.get("section") || "", t)
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
        {items.length ? <NavMain items={items} active={pathname} search={search} /> : null}
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
    path.startsWith("/board") ||
    path.startsWith("/repos") ||
    path.startsWith("/pipelines") ||
    path.startsWith("/packages") ||
    path.startsWith("/admin")
  )
}
