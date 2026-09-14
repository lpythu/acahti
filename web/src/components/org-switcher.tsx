import { ChevronsUpDownIcon } from "lucide-react"

import { Avatar, AvatarFallback } from "@/components/ui/avatar"
import {
  DropdownMenu,
  DropdownMenuCheckboxItem,
  DropdownMenuContent,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"

function orgLabel(org: string) {
  return org ? org.toUpperCase() : "ORG"
}

export function OrgSwitcher({ org }: { org: string }) {
  const label = orgLabel(org)
  const initials = label.slice(0, 2)

  return (
    <DropdownMenu>
      <DropdownMenuTrigger className="flex h-10 min-w-0 items-center gap-2 rounded-md px-2 outline-none hover:bg-accent focus-visible:ring-3 focus-visible:ring-ring/50 md:h-12 md:w-(--sidebar-width) md:px-4">
        <Avatar size="sm" className="rounded-lg after:rounded-lg">
          <AvatarFallback className="rounded-lg text-xs">{initials}</AvatarFallback>
        </Avatar>
        <span className="min-w-0 flex-1 truncate text-left text-sm font-medium">{label}</span>
        <ChevronsUpDownIcon className="size-4 shrink-0 text-muted-foreground" />
      </DropdownMenuTrigger>
      <DropdownMenuContent align="start" className="w-56 min-w-56">
        <DropdownMenuCheckboxItem checked className="gap-2 pr-8">
          <Avatar size="sm" className="rounded-lg after:rounded-lg">
            <AvatarFallback className="rounded-lg text-xs">{initials}</AvatarFallback>
          </Avatar>
          <div className="grid min-w-0 flex-1 leading-tight">
            <span className="truncate text-sm font-medium">{label}</span>
            <span className="truncate text-xs text-muted-foreground">{org}</span>
          </div>
        </DropdownMenuCheckboxItem>
      </DropdownMenuContent>
    </DropdownMenu>
  )
}
