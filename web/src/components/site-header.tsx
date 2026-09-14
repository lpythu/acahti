import type { ReactNode } from "react"
import { Link, useLocation } from "react-router-dom"
import { cn } from "cn"

import { OrgSwitcher } from "@/components/org-switcher"
import { NavUser } from "@/components/nav-user"
import { SidebarTrigger } from "@/components/ui/sidebar"
import { useT } from "@/i18n/i18n"
import type { Me } from "@/lib/api"
import { sectionOf } from "@/lib/nav"

function TopTab({
  to,
  active,
  children,
}: {
  to: string
  active: boolean
  children: ReactNode
}) {
  return (
    <Link
      to={to}
      aria-current={active ? "page" : undefined}
      className={cn(
        "inline-flex h-10 shrink-0 items-center border-b-2 px-3 text-sm font-medium transition-colors md:h-12",
        active
          ? "border-primary text-foreground"
          : "border-transparent text-muted-foreground hover:border-border hover:text-foreground",
      )}
    >
      {children}
    </Link>
  )
}

export function SiteHeader({
  me,
  hasSidebarNav,
}: {
  me: Me
  hasSidebarNav: boolean
}) {
  const t = useT()
  const section = sectionOf(useLocation().pathname)
  return (
    <header className="z-40 shrink-0 border-b bg-background">
      <div className="flex h-10 items-center gap-1 overflow-hidden pr-3 pl-[max(0.5rem,env(safe-area-inset-left,0px))] md:h-12 md:gap-0 md:pr-6 md:pl-0">
        {hasSidebarNav ? <SidebarTrigger className="md:hidden" /> : null}
        <OrgSwitcher org={me.org} />
        <nav
          className="scrollbar-auto-hide-overlay flex h-full min-w-0 flex-1 items-stretch gap-0.5 overflow-x-auto"
          aria-label={t("primaryNav")}
        >
          <TopTab to="/board" active={section === "board"}>
            {t("board")}
          </TopTab>
          <TopTab to="/repos" active={section === "repos"}>
            {t("repos")}
          </TopTab>
          <TopTab to="/pipelines" active={section === "pipelines"}>
            {t("pipelines")}
          </TopTab>
          <TopTab to="/packages" active={section === "packages"}>
            {t("packages")}
          </TopTab>
        </nav>
        <div className="ml-auto shrink-0">
          <NavUser
            admin={me.admin}
            user={{
              name: me.git_name || me.user,
              email: me.git_email || (me.admin ? t("admin") : me.user),
            }}
          />
        </div>
      </div>
    </header>
  )
}
