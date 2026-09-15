import { Link, useNavigate } from "react-router-dom"
import { LogOutIcon, ShieldIcon } from "lucide-react"

import { Button } from "@/components/ui/button"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
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
  user: { name: string; username: string }
  admin: boolean
}) {
  const t = useT()
  const nav = useNavigate()
  const { clear } = useSession()

  async function logout() {
    await api.logout()
    clear()
    nav("/login", { replace: true })
  }

  return (
    <DropdownMenu>
      <DropdownMenuTrigger
        render={<Button variant="ghost" size="sm" className="px-2 font-medium hover:bg-transparent aria-expanded:bg-transparent" />}
      >
        {user.name}
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="w-56">
        <DropdownMenuLabel className="px-1.5 py-1.5 font-normal">
          <p className="text-sm font-medium text-foreground">{user.name}</p>
          <p className="text-xs text-muted-foreground">{user.username}</p>
        </DropdownMenuLabel>
        {admin ? (
          <>
            <DropdownMenuSeparator />
            <DropdownMenuItem render={<Link to="/admin" target="_blank" rel="noopener noreferrer" />}>
              <ShieldIcon />
              {t("admin")}
            </DropdownMenuItem>
          </>
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
