import { useEffect, useMemo, useState, type ReactNode } from "react"
import { Link, useLocation, useSearchParams } from "react-router-dom"
import { BookMarkedIcon, ChevronRightIcon, FolderIcon, WorkflowIcon } from "lucide-react"
import { cn } from "cn"

import { Collapsible, CollapsibleContent, CollapsibleTrigger } from "@/components/ui/collapsible"
import {
  SidebarGroup,
  SidebarGroupContent,
  SidebarGroupLabel,
  SidebarInput,
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
import { api, splitRepo, type NavTeam, type Repo } from "@/lib/api"
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

function repoName(r: Repo) {
  return splitRepo(r.full_name || r.name).name || r.name
}

function filterTree(tree: NavTeam[], q: string, unassignedLabel: string): NavTeam[] {
  const needle = q.trim().toLowerCase()
  if (!needle) return tree
  const out: NavTeam[] = []
  for (const row of tree) {
    const label = (row.team || unassignedLabel).toLowerCase()
    if (label.includes(needle)) {
      out.push(row)
      continue
    }
    const repos = (row.repos || []).filter((r) => {
      const name = repoName(r).toLowerCase()
      const full = (r.full_name || "").toLowerCase()
      return name.includes(needle) || full.includes(needle)
    })
    if (repos.length) out.push({ ...row, repos })
  }
  return out
}

function TeamNode({
  section,
  team,
  label,
  repos,
  pathname,
  filterTeam,
  filterRepo,
  itemIcon,
  forceOpen,
}: {
  section: CodeTeamSection
  team: string
  label: string
  repos: Repo[]
  pathname: string
  filterTeam: string
  filterRepo: string
  itemIcon: ReactNode
  forceOpen?: boolean
}) {
  const pipe = parsePipelinePath(pathname)
  const teamActive = Boolean(team) && !filterRepo && !pipe && filterTeam === team && !repoBase(pathname)
  const inTeam = repos.some((r) => {
    const { owner, name } = splitRepo(r.full_name || r.name)
    if (filterRepo === `${owner}/${name}`) return true
    if (pipe?.owner === owner && pipe.name === name) return true
    const base = `/repos/${owner}/${name}`
    return pathname === base || pathname.startsWith(`${base}/`)
  })
  const [open, setOpen] = useState(true)

  useEffect(() => {
    if (inTeam || teamActive || forceOpen) setOpen(true)
  }, [inTeam, teamActive, forceOpen])

  return (
    <SidebarMenuItem>
      <Collapsible open={open} onOpenChange={setOpen}>
        <div className="flex min-w-0 items-center">
          <SidebarMenuButton
            tooltip={label}
            isActive={teamActive}
            className="flex-1"
            render={team ? <Link to={teamHref(section, team)} /> : undefined}
          >
            <FolderIcon />
            <span>{label}</span>
          </SidebarMenuButton>
          <CollapsibleTrigger
            className={cn(
              "flex size-8 shrink-0 items-center justify-center rounded-md text-sidebar-foreground ring-sidebar-ring outline-hidden hover:bg-sidebar-accent hover:text-sidebar-accent-foreground focus-visible:ring-2 group-data-[collapsible=icon]:hidden",
            )}
            aria-label={label}
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
  const [q, setQ] = useState("")
  const filterTeam = sp.get("team") || ""
  const filterRepo = sp.get("repo") || ""
  const tree = useLoad(() => api.navTree(), [])
  useEvents((ev) => {
    if (ev.type === "catalog.updated") void tree.reload()
  })

  const itemIcon = section === "pipelines" ? <WorkflowIcon /> : <BookMarkedIcon />
  const unassignedLabel = t("unassignedRepos")
  const rows = useMemo(
    () => filterTree(tree.data ?? [], q, unassignedLabel),
    [tree.data, q, unassignedLabel],
  )
  const searching = Boolean(q.trim())

  return (
    <>
      <SidebarGroup className="group-data-[collapsible=icon]:hidden">
        <SidebarGroupContent>
          <SidebarInput
            value={q}
            onChange={(e) => setQ(e.target.value)}
            placeholder={t("searchRepos")}
            aria-label={t("searchRepos")}
          />
        </SidebarGroupContent>
      </SidebarGroup>
      <SidebarGroup>
        <SidebarGroupLabel>{t("teams")}</SidebarGroupLabel>
        <SidebarGroupContent>
          {searching && !rows.length ? (
            <p className="px-2 py-1.5 text-xs text-muted-foreground">{t("noMatchingRepos")}</p>
          ) : (
            <SidebarMenu>
              {rows.map((row) => (
                <TeamNode
                  key={row.team || "unassigned"}
                  section={section}
                  team={row.team}
                  label={row.team || unassignedLabel}
                  repos={row.repos || []}
                  pathname={pathname}
                  filterTeam={filterTeam}
                  filterRepo={filterRepo}
                  itemIcon={itemIcon}
                  forceOpen={searching}
                />
              ))}
            </SidebarMenu>
          )}
        </SidebarGroupContent>
      </SidebarGroup>
    </>
  )
}
