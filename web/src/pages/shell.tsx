import { type ReactNode } from "react"
import { Navigate, Outlet, useLocation } from "react-router-dom"
import {
  GitBranchIcon,
  KeyRoundIcon,
  KeySquareIcon,
  LayoutDashboardIcon,
  PackageIcon,
  ShieldIcon,
  UsersIcon,
  WorkflowIcon,
} from "lucide-react"

import { AppSidebar } from "@/components/app-sidebar"
import { SiteHeader } from "@/components/site-header"
import { SidebarInset, SidebarProvider } from "@/components/ui/sidebar"
import { useT } from "@/i18n/i18n"
import type { Me } from "@/lib/api"
import { useSession } from "@/lib/session"
import type { MessageKey } from "@/i18n/messages"

function titleFor(path: string, t: (k: MessageKey) => string) {
  if (path.startsWith("/account/keys") || path.startsWith("/keys")) return t("keys")
  if (path.startsWith("/account/token") || path.startsWith("/token")) return t("token")
  if (path.includes("/pulls/")) return t("prs")
  if (path.includes("/pipelines/") && path.startsWith("/repos/")) return t("pipelines")
  if (path.startsWith("/repos")) return t("repos")
  if (path.startsWith("/pipelines")) return t("pipelines")
  if (path.startsWith("/packages")) return t("packages")
  if (path.startsWith("/admin/users")) return t("users")
  if (path.startsWith("/admin")) return t("admin")
  return t("board")
}

function AuthGate({
  adminOnly,
  children,
}: {
  adminOnly?: boolean
  children: (me: Me) => ReactNode
}) {
  const { me, ready } = useSession()

  if (!ready) {
    return <div className="bg-background min-h-svh" />
  }
  if (!me) {
    return <Navigate to="/login" replace />
  }
  if (adminOnly && !me.admin) {
    return <Navigate to="/" replace />
  }
  return children(me)
}

function ShellFrame({
  me,
  area,
  items,
  home,
}: {
  me: Me
  area: "console" | "admin"
  items: { title: string; url: string; icon?: ReactNode }[]
  home: string
}) {
  const t = useT()
  const loc = useLocation()
  return (
    <SidebarProvider
      className="flex min-h-svh flex-col"
      style={
        {
          "--sidebar-width": "calc(var(--spacing) * 52)",
          "--header-height": "calc(var(--spacing) * 12)",
        } as React.CSSProperties
      }
    >
      <SiteHeader title={titleFor(loc.pathname, t)} me={me} area={area} />
      <div className="flex min-h-0 flex-1">
        <AppSidebar variant="inset" area={area} items={items} home={home} />
        <SidebarInset>
          <div className="flex flex-1 flex-col">
            <div className="@container/main flex flex-1 flex-col">
              <Outlet context={me} />
            </div>
          </div>
        </SidebarInset>
      </div>
    </SidebarProvider>
  )
}

export function ConsoleShell() {
  const t = useT()
  return (
    <AuthGate>
      {(me) => (
        <ShellFrame
          me={me}
          area="console"
          home="/"
          items={[
            { title: t("board"), url: "/", icon: <LayoutDashboardIcon /> },
            { title: t("repos"), url: "/repos", icon: <GitBranchIcon /> },
            { title: t("pipelines"), url: "/pipelines", icon: <WorkflowIcon /> },
            { title: t("packages"), url: "/packages", icon: <PackageIcon /> },
            { title: t("keys"), url: "/account/keys", icon: <KeyRoundIcon /> },
            { title: t("token"), url: "/account/token", icon: <KeySquareIcon /> },
          ]}
        />
      )}
    </AuthGate>
  )
}

export function AdminShell() {
  const t = useT()
  return (
    <AuthGate adminOnly>
      {(me) => (
        <ShellFrame
          me={me}
          area="admin"
          home="/admin"
          items={[
            { title: t("adminHome"), url: "/admin", icon: <ShieldIcon /> },
            { title: t("users"), url: "/admin/users", icon: <UsersIcon /> },
          ]}
        />
      )}
    </AuthGate>
  )
}
