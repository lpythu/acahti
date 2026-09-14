import { useEffect, useState, type ReactNode } from "react"
import { Link, useLocation, useSearchParams } from "react-router-dom"
import { BookMarkedIcon, ChevronRightIcon, FolderIcon, WorkflowIcon } from "lucide-react"
import { cn } from "cn"

import { Collapsible, CollapsibleContent, CollapsibleTrigger } from "@/components/ui/collapsible"
import {
  SidebarGroup,
  SidebarGroupContent,
  SidebarGroupLabel,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarMenuSub,
  SidebarMenuSubButton,
  SidebarMenuSubItem,
} from "@/components/ui/sidebar"
import { useEvents } from "@/hooks/use-events"
import { useLoad } from "@/hooks/use-load"
import { useT } from "@/i18n/i18n"
import { api, splitRepo, type Repo } from "@/lib/api"
import { parsePipelinePath, repoBase } from "@/lib/nav"

export type CodeTeamSection = "repos" | "pipelines"

function teamHref(section: CodeTeamSection, team: string) {
  const base = section === "pipelines" ? "/pipelines" : "/repos"
  return `${base}?team=${encodeURIComponent(team)}`
}

function itemHref(section: CodeTeamSection, owner: string, name: string) {
  if (section === "pipelines") {
    return `/pipelines?repo=${encodeURIComponent(`${owner}/${name}`)}`
  }
  return `/repos/${owner}/${name}`
}

function TeamNode({
  section,
  team,
  repos,
  pathname,
  filterTeam,
  filterRepo,
  itemIcon,
}: {
  section: CodeTeamSection
  team: string
  repos: Repo[]
  pathname: string
  filterTeam: string
  filterRepo: string
  itemIcon: ReactNode
}) {
  const pipe = parsePipelinePath(pathname)
  const teamActive = !filterRepo && !pipe && filterTeam === team && !repoBase(pathname)
  const inTeam = repos.some((r) => {
    const { owner, name } = splitRepo(r.full_name || r.name)
    if (filterRepo === `${owner}/${name}`) return true
    if (pipe?.owner === owner && pipe.name === name) return true
    const base = `/repos/${owner}/${name}`
    return pathname === base || pathname.startsWith(`${base}/`)
  })
  const [open, setOpen] = useState(true)

  useEffect(() => {
    if (inTeam || teamActive) setOpen(true)
  }, [inTeam, teamActive])

  return (
    <SidebarMenuItem>
      <Collapsible open={open} onOpenChange={setOpen}>
        <div className="flex min-w-0 items-center">
          <SidebarMenuButton
            tooltip={team}
            isActive={teamActive}
            className="flex-1"
            render={<Link to={teamHref(section, team)} />}
          >
            <FolderIcon />
            <span>{team}</span>
          </SidebarMenuButton>
          <CollapsibleTrigger
            className={cn(
              "flex size-8 shrink-0 items-center justify-center rounded-md text-sidebar-foreground ring-sidebar-ring outline-hidden hover:bg-sidebar-accent hover:text-sidebar-accent-foreground focus-visible:ring-2 group-data-[collapsible=icon]:hidden",
            )}
            aria-label={team}
          >
            <ChevronRightIcon className={cn("size-4 transition-transform", open && "rotate-90")} />
          </CollapsibleTrigger>
        </div>
        <CollapsibleContent>
          <SidebarMenuSub>
            {repos.map((r) => {
              const { owner, name } = splitRepo(r.full_name || r.name)
              const url = itemHref(section, owner, name)
              const active =
                filterRepo === `${owner}/${name}` ||
                (pipe?.owner === owner && pipe.name === name) ||
                pathname === `/repos/${owner}/${name}` ||
                pathname.startsWith(`/repos/${owner}/${name}/`)
              return (
                <SidebarMenuSubItem key={r.full_name || r.name}>
                  <SidebarMenuSubButton isActive={active} render={<Link to={url} />}>
                    {itemIcon}
                    <span>{name}</span>
                  </SidebarMenuSubButton>
                </SidebarMenuSubItem>
              )
            })}
          </SidebarMenuSub>
        </CollapsibleContent>
      </Collapsible>
    </SidebarMenuItem>
  )
}

export function CodeTeamNav({ section }: { section: CodeTeamSection }) {
  const t = useT()
  const { pathname } = useLocation()
  const [sp] = useSearchParams()
  const filterTeam = sp.get("team") || ""
  const filterRepo = sp.get("repo") || ""
  const tree = useLoad(() => api.navTree(), [])
  useEvents((ev) => {
    if (ev.type === "forgejo") void tree.reload()
  })

  if (!tree.data?.length) return null

  const itemIcon = section === "pipelines" ? <WorkflowIcon /> : <BookMarkedIcon />

  return (
    <SidebarGroup>
      <SidebarGroupLabel>{t("teams")}</SidebarGroupLabel>
      <SidebarGroupContent>
        <SidebarMenu>
          {tree.data.map((row) => (
            <TeamNode
              key={row.team}
              section={section}
              team={row.team}
              repos={row.repos}
              pathname={pathname}
              filterTeam={filterTeam}
              filterRepo={filterRepo}
              itemIcon={itemIcon}
            />
          ))}
        </SidebarMenu>
      </SidebarGroupContent>
    </SidebarGroup>
  )
}
