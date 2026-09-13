import { NavUser } from "@/components/nav-user"
import { Separator } from "@/components/ui/separator"
import { SidebarTrigger } from "@/components/ui/sidebar"
import { useT } from "@/i18n/i18n"
import type { Me } from "@/lib/api"

export function SiteHeader({
  title,
  me,
  area,
}: {
  title: string
  me: Me
  area: "console" | "admin"
}) {
  const t = useT()
  return (
    <header className="z-40 flex h-(--header-height) shrink-0 items-center gap-2 border-b bg-background">
      <div className="flex w-full items-center gap-1 px-4 lg:gap-2 lg:px-6">
        <SidebarTrigger className="-ml-1" />
        <Separator orientation="vertical" className="mx-2 h-4 data-vertical:self-auto" />
        <h1 className="text-base font-medium">{title}</h1>
        <div className="ml-auto">
          <NavUser
            area={area}
            admin={me.admin}
            user={{
              name: me.user,
              email: me.admin ? t("admin") : me.user,
            }}
          />
        </div>
      </div>
    </header>
  )
}
