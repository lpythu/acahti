import { Avatar, AvatarFallback } from "@/components/ui/avatar"

function orgLabel(org: string) {
  return org ? org.toUpperCase() : "ORG"
}

export function OrgSwitcher({ org }: { org: string }) {
  const label = orgLabel(org)
  const initials = label.slice(0, 2)

  return (
    <div className="flex h-10 min-w-0 items-center gap-2 rounded-md px-2 md:h-12 md:w-(--sidebar-width) md:px-4">
      <Avatar size="sm" className="rounded-lg after:rounded-lg">
        <AvatarFallback className="rounded-lg text-xs">{initials}</AvatarFallback>
      </Avatar>
      <span className="min-w-0 flex-1 truncate text-left text-sm font-medium">{label}</span>
    </div>
  )
}
