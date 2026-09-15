import { Link } from "react-router-dom"

import {
  SidebarGroup,
  SidebarGroupContent,
  SidebarGroupLabel,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
} from "@/components/ui/sidebar"

function pathOnly(url: string) {
  return url.split("?")[0]
}

function queryActive(search: string, url: string) {
  const q = url.indexOf("?")
  if (q < 0) return true
  const want = new URLSearchParams(url.slice(q + 1))
  const have = new URLSearchParams(search)
  for (const [k, v] of want) {
    if (have.get(k) !== v) return false
  }
  return true
}

function navActive(path: string, search: string, url: string, end?: boolean) {
  const dest = pathOnly(url)
  const pathOk = end || dest === "/" || dest === "/admin" ? path === dest : path === dest || path.startsWith(dest + "/")
  return pathOk && queryActive(search, url)
}

export function NavMain({
  items,
  active,
  search = "",
  label,
}: {
  items: {
    title: string
    url: string
    icon?: React.ReactNode
    end?: boolean
    /** When set, overrides path/query matching for this item. */
    active?: boolean
  }[]
  active: string
  search?: string
  label?: string
}) {
  return (
    <SidebarGroup>
      {label ? <SidebarGroupLabel>{label}</SidebarGroupLabel> : null}
      <SidebarGroupContent className="flex flex-col gap-2">
        <SidebarMenu>
          {items.map((item) => (
            <SidebarMenuItem key={item.url}>
              <SidebarMenuButton
                tooltip={item.title}
                isActive={item.active ?? navActive(active, search, item.url, item.end)}
                render={<Link to={item.url} />}
              >
                {item.icon}
                <span>{item.title}</span>
              </SidebarMenuButton>
            </SidebarMenuItem>
          ))}
        </SidebarMenu>
      </SidebarGroupContent>
    </SidebarGroup>
  )
}
