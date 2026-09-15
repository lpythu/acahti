import { useNavigate } from "react-router-dom"
import { LogOutIcon } from "lucide-react"

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
}: {
  user: { name: string; username: string }
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
        type="button"
        className="inline-flex h-7 items-center px-1 text-sm font-medium text-foreground outline-none hover:opacity-70 focus-visible:ring-3 focus-visible:ring-ring/50"
      >
        {user.name}
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="w-56">
        <div className="px-1.5 py-1.5">
          <p className="truncate text-sm font-medium">{user.name}</p>
          <p className="truncate text-xs text-muted-foreground">{user.username}</p>
        </div>
        <DropdownMenuSeparator />
        <DropdownMenuItem onClick={() => void logout()}>
          <LogOutIcon />
          {t("logout")}
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  )
}
