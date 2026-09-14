import { useEffect, useState, type ReactNode } from "react"
import { Link, useLocation, useSearchParams } from "react-router-dom"
import { BookMarkedIcon, ChevronRightIcon, FolderIcon, WorkflowIcon } from "lucide-react"
import { cn } from "cn"

import { MoreButton } from "@/components/paged-list"
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
import { useLoad } from "@/hooks/use-load"
import { usePage } from "@/hooks/use-page"
import { useT } from "@/i18n/i18n"
import { api, splitRepo } from "@/lib/api"
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
  pathname,
  filterTeam,
  filterRepo,
  defaultOpen,
  itemIcon,
}: {
  section: CodeTeamSection
  team: string
  pathname: string
  filterTeam: string
  filterRepo: string
  defaultOpen: boolean
  itemIcon: ReactNode
}) {
  const pipe = parsePipelinePath(pathname)
  const teamActive = !filterRepo && !pipe && filterTeam === team && !repoBase(pathname)
  const [open, setOpen] = useState(defaultOpen || teamActive)
  const repos = usePage((q) => api.repos(q, team), [team], {
    url: false,
    enabled: open,
  })
  const inTeam = repos.items.some((r) => {
    const { owner, name } = splitRepo(r.full_name || r.name)
    if (filterRepo === `${owner}/${name}`) return true
    if (pipe?.owner === owner && pipe.name === name) return true
    const base = `/repos/${owner}/${name}`
    return pathname === base || pathname.startsWith(`${base}/`)
  })

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
            {repos.items.map((r) => {
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
            <MoreButton page={repos.page} hasMore={repos.hasMore} onPage={repos.setPage} />
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
  const teams = usePage((q) => api.repoTeams(q), [], { url: false })
  const pipe = parsePipelinePath(pathname)
  const current = repoBase(pathname)
  const headOwner = pipe?.owner || (current ? current.split("/")[2] : splitRepo(filterRepo).owner)
  const headName = pipe?.name || (current ? current.split("/")[3] : splitRepo(filterRepo).name)
  const head = useLoad(
    () => api.repo(headOwner, headName),
    [headOwner, headName],
    Boolean(headOwner && headName),
  )
  const currentTeam = filterTeam || head.data?.repo.team || ""
  const openAll = teams.items.length <= 1 && !filterTeam && !currentTeam

  if (!teams.items.length) return null

  const itemIcon = section === "pipelines" ? <WorkflowIcon /> : <BookMarkedIcon />

  return (
    <SidebarGroup>
      <SidebarGroupLabel>{t("teams")}</SidebarGroupLabel>
      <SidebarGroupContent>
        <SidebarMenu>
          {teams.items.map((row) => (
            <TeamNode
              key={row.team}
              section={section}
              team={row.team}
              pathname={pathname}
              filterTeam={filterTeam}
              filterRepo={filterRepo}
              defaultOpen={openAll || row.team === currentTeam}
              itemIcon={itemIcon}
            />
          ))}
          <MoreButton page={teams.page} hasMore={teams.hasMore} onPage={teams.setPage} />
        </SidebarMenu>
      </SidebarGroupContent>
    </SidebarGroup>
  )
}
