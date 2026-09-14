import { Link, useNavigate } from "react-router-dom"
import { KeyRoundIcon, LogOutIcon, ShieldIcon } from "lucide-react"

import { Avatar, AvatarFallback } from "@/components/ui/avatar"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { useT } from "@/i18n/i18n"
import { api } from "@/lib/api"
import { useSession } from "@/lib/session"

export function NavUser({
  user,
  admin,
}: {
  user: { name: string; email: string }
  admin: boolean
}) {
  const t = useT()
  const nav = useNavigate()
  const { clear } = useSession()
  const initials = user.name.slice(0, 2).toUpperCase()

  async function logout() {
    await api.logout()
    clear()
    nav("/login", { replace: true })
  }

  return (
    <DropdownMenu>
      <DropdownMenuTrigger className="rounded-lg outline-none focus-visible:ring-3 focus-visible:ring-ring/50">
        <Avatar className="size-8 rounded-lg grayscale">
          <AvatarFallback className="rounded-lg">{initials}</AvatarFallback>
        </Avatar>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="w-56">
        <div className="flex items-center gap-2 px-2 py-1.5">
          <Avatar className="size-8">
            <AvatarFallback className="rounded-lg">{initials}</AvatarFallback>
          </Avatar>
          <div className="grid min-w-0 flex-1 leading-tight">
            <span className="truncate text-sm font-medium">{user.name}</span>
            <span className="truncate text-xs text-muted-foreground">{user.email}</span>
          </div>
        </div>
        <DropdownMenuSeparator />
        <DropdownMenuItem render={<Link to="/account/keys" />}>
          <KeyRoundIcon />
          {t("keys")}
        </DropdownMenuItem>
        {admin ? (
          <DropdownMenuItem render={<Link to="/admin" target="_blank" rel="noopener noreferrer" />}>
            <ShieldIcon />
            {t("admin")}
          </DropdownMenuItem>
        ) : null}
        <DropdownMenuSeparator />
        <DropdownMenuItem onClick={() => void logout()}>
          <LogOutIcon />
          {t("logout")}
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  )
}
