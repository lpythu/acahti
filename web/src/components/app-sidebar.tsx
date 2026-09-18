import type { ReactNode } from "react"
import { useLocation, useSearchParams } from "react-router-dom"
import {
  Code2Icon,
  FolderIcon,
  GitBranchIcon,
  GitCommitVerticalIcon,
  GitPullRequestIcon,
  PanelLeftCloseIcon,
  PanelLeftIcon,
  Building2Icon,
  ShieldIcon,
  TagIcon,
  UsersIcon,
  KeyRoundIcon,
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
import { useLoad } from "@/hooks/use-load"
import { api } from "@/lib/api"
import { repoBase, sectionOf } from "@/lib/nav"
import { useSession } from "@/lib/session"

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
  t: (
    k:
      | "files"
      | "commits"
      | "branches"
      | "tags"
      | "tabPulls"
      | "tabPipes"
      | "access"
      | "tabSecrets"
      | "adminHome"
      | "adminSecrets"
      | "orgs"
      | "users"
      | "teams",
  ) => string,
  canManageSecrets: boolean,
): NavItem[] {
  const base = repoBase(path)
  const q = ref ? `?ref=${encodeURIComponent(ref)}` : ""
  if (base) {
    const items: NavItem[] = [
      { title: t("files"), url: `${base}${q}`, icon: <Code2Icon />, end: true },
      { title: t("commits"), url: `${base}/commits${q}`, icon: <GitCommitVerticalIcon /> },
      { title: t("branches"), url: `${base}/branches`, icon: <GitBranchIcon /> },
      { title: t("tags"), url: `${base}/tags`, icon: <TagIcon /> },
      { title: t("tabPulls"), url: `${base}/pulls`, icon: <GitPullRequestIcon /> },
      { title: t("tabPipes"), url: `${base}/pipelines`, icon: <WorkflowIcon /> },
      { title: t("access"), url: `${base}/access`, icon: <LockIcon /> },
    ]
    if (canManageSecrets) {
      items.push({ title: t("tabSecrets"), url: `${base}/secrets`, icon: <KeyRoundIcon /> })
    }
    return items
  }
  if (path.startsWith("/admin")) {
    return [
      { title: t("adminHome"), url: "/admin", icon: <ShieldIcon />, end: true },
      { title: t("orgs"), url: "/admin/orgs", icon: <Building2Icon /> },
      { title: t("adminSecrets"), url: "/admin/secrets", icon: <KeyRoundIcon /> },
      { title: t("teams"), url: "/admin/teams", icon: <FolderIcon /> },
      { title: t("users"), url: "/admin/users", icon: <UsersIcon /> },
    ]
  }
  return []
}

export function AppSidebar({ ...props }: React.ComponentProps<typeof Sidebar>) {
  const t = useT()
  const { me } = useSession()
  const { pathname, search } = useLocation()
  const [sp] = useSearchParams()
  const base = repoBase(pathname)
  const repoLoad = useLoad(
    () => {
      if (!base) return Promise.resolve(null)
      const parts = base.split("/")
      return api.repo(parts[2], parts[3])
    },
    [base],
    Boolean(base),
  )
  const canManageSecrets = Boolean(me?.admin || repoLoad.data?.repo.can_manage_secrets)
  const items = secondaryItems(pathname, sp.get("ref") || "", t, canManageSecrets)
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
    path.startsWith("/repos") ||
    path.startsWith("/pipelines") ||
    path.startsWith("/packages") ||
    path.startsWith("/admin")
  )
}
