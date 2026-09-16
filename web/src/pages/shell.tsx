import type { ReactNode } from "react"
import { Navigate, Outlet, useLocation } from "react-router-dom"

import { AppSidebar, sidebarHasNav } from "@/components/app-sidebar"
import { BootScreen } from "@/components/boot"
import { SiteHeader } from "@/components/site-header"
import { SidebarInset, SidebarProvider } from "@/components/ui/sidebar"
import { useT } from "@/i18n/i18n"
import type { Me } from "@/lib/api"
import { useSession } from "@/lib/session"

function AuthGate({ children }: { children: (me: Me) => ReactNode }) {
  const { me, ready } = useSession()
  const loc = useLocation()
  const t = useT()

  if (!ready) {
    return <BootScreen label={t("loading")} />
  }
  if (!me) {
    return <Navigate to={`/login?next=${encodeURIComponent(loc.pathname + loc.search)}`} replace />
  }
  return children(me)
}

function ShellFrame({ me }: { me: Me }) {
  const hasSidebarNav = sidebarHasNav(useLocation().pathname)
  return (
    <SidebarProvider
      className="flex h-svh flex-col overflow-hidden"
      style={
        {
          "--sidebar-width": "13rem",
          "--header-height": "3rem",
        } as React.CSSProperties
      }
    >
      <SiteHeader me={me} hasSidebarNav={hasSidebarNav} />
      <div className="flex min-h-0 flex-1 overflow-hidden">
        {hasSidebarNav ? <AppSidebar /> : null}
        <SidebarInset className="min-h-0 min-w-0 overflow-hidden">
          <div className="@container/main flex min-h-0 flex-1 flex-col overflow-hidden">
            <Outlet context={me} />
          </div>
        </SidebarInset>
      </div>
    </SidebarProvider>
  )
}

export function AppShell() {
  return <AuthGate>{(me) => <ShellFrame me={me} />}</AuthGate>
}

export function AdminGate() {
  const { me, ready } = useSession()
  const t = useT()
  if (!ready) {
    return <BootScreen label={t("loading")} />
  }
  if (!me?.admin) {
    return <Navigate to="/board" replace />
  }
  return <Outlet context={me} />
}
