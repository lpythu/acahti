import { useMemo, useState, type ReactNode } from "react"
import { Link, useLocation, useSearchParams } from "react-router-dom"
import { BookMarkedIcon, ChevronRightIcon, FolderIcon, ListChevronsDownUpIcon, ListChevronsUpDownIcon, WorkflowIcon } from "lucide-react"
import { cn } from "cn"

import { Collapsible, CollapsibleContent, CollapsibleTrigger } from "@/components/ui/collapsible"
import {
  SidebarGroup,
  SidebarGroupAction,
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
import { api, repoName, splitRepo, type NavTeam, type Repo } from "@/lib/api"
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

function liveRepos(repos: Repo[] | undefined): Repo[] {
  return (repos || []).filter((r) => !r.archived)
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
    const repos = row.repos.filter((r) => {
      const name = repoName(r.full_name || r.name).toLowerCase()
      const full = (r.full_name || "").toLowerCase()
      return name.includes(needle) || full.includes(needle)
    })
    if (repos.length) out.push({ ...row, repos })
  }
  return out
}

function repoActive(pathname: string, filterRepo: string, owner: string, name: string) {
  const pipe = parsePipelinePath(pathname)
  return (
    filterRepo === `${owner}/${name}` ||
    (pipe?.owner === owner && pipe.name === name) ||
    pathname === `/repos/${owner}/${name}` ||
    pathname.startsWith(`/repos/${owner}/${name}/`)
  )
}

function RepoLink({
  section,
  repo,
  pathname,
  filterRepo,
  itemIcon,
  nested,
}: {
  section: CodeTeamSection
  repo: Repo
  pathname: string
  filterRepo: string
  itemIcon: ReactNode
  nested?: boolean
}) {
  const { owner, name } = splitRepo(repo.full_name || repo.name)
  const Comp = nested ? SidebarMenuSubButton : SidebarMenuButton
  return (
    <Comp isActive={repoActive(pathname, filterRepo, owner, name)} render={<Link to={itemHref(section, owner, name)} />}>
      {itemIcon}
      <span>{name}</span>
    </Comp>
  )
}

function TeamNode({
  section,
  team,
  repos,
  pathname,
  filterTeam,
  filterRepo,
  itemIcon,
  open: openProp,
  onOpenChange,
  forceOpen,
}: {
  section: CodeTeamSection
  team: string
  repos: Repo[]
  pathname: string
  filterTeam: string
  filterRepo: string
  itemIcon: ReactNode
  open?: boolean
  onOpenChange: (open: boolean) => void
  forceOpen?: boolean
}) {
  const pipe = parsePipelinePath(pathname)
  const teamActive = !filterRepo && !pipe && filterTeam === team && !repoBase(pathname)
  const inTeam = repos.some((r) => {
    const { owner, name } = splitRepo(r.full_name || r.name)
    return repoActive(pathname, filterRepo, owner, name)
  })
  const open = forceOpen || (openProp ?? (inTeam || teamActive))

  return (
    <SidebarMenuItem>
      <Collapsible open={open} onOpenChange={onOpenChange}>
        <div className="flex min-w-0 items-center">
          <SidebarMenuButton tooltip={team} isActive={teamActive} className="flex-1" render={<Link to={teamHref(section, team)} />}>
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
            {repos.map((r) => (
              <SidebarMenuSubItem key={r.full_name || r.name}>
                <RepoLink section={section} repo={r} pathname={pathname} filterRepo={filterRepo} itemIcon={itemIcon} nested />
              </SidebarMenuSubItem>
            ))}
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
  const rows = useMemo(() => {
    const src = (tree.data ?? [])
      .map((row) => ({ ...row, repos: liveRepos(row.repos) }))
      .filter((row) => row.team || row.repos.length)
    return filterTree(src, q, unassignedLabel)
  }, [tree.data, q, unassignedLabel])
  const teams = rows.filter((row) => row.team)
  const loose = rows.find((row) => !row.team)?.repos ?? []
  const searching = Boolean(q.trim())
  const [openByKey, setOpenByKey] = useState<Record<string, boolean>>({})
  const allOpen = teams.length > 0 && teams.every((row) => openByKey[row.team])

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
      {searching && !teams.length && !loose.length ? (
        <p className="px-4 py-1.5 text-xs text-muted-foreground">{t("noMatchingRepos")}</p>
      ) : null}
      {teams.length ? (
        <SidebarGroup>
          <SidebarGroupLabel>{t("teams")}</SidebarGroupLabel>
          {searching ? null : (
            <SidebarGroupAction
              title={allOpen ? t("collapseAll") : t("expandAll")}
              aria-label={allOpen ? t("collapseAll") : t("expandAll")}
              onClick={() => {
                const next = !allOpen
                setOpenByKey(Object.fromEntries(teams.map((row) => [row.team, next])))
              }}
            >
              {allOpen ? <ListChevronsDownUpIcon /> : <ListChevronsUpDownIcon />}
            </SidebarGroupAction>
          )}
          <SidebarGroupContent>
            <SidebarMenu>
              {teams.map((row) => (
                <TeamNode
                  key={row.team}
                  section={section}
                  team={row.team}
                  repos={row.repos}
                  pathname={pathname}
                  filterTeam={filterTeam}
                  filterRepo={filterRepo}
                  itemIcon={itemIcon}
                  open={openByKey[row.team]}
                  onOpenChange={(open) => setOpenByKey((prev) => ({ ...prev, [row.team]: open }))}
                  forceOpen={searching}
                />
              ))}
            </SidebarMenu>
          </SidebarGroupContent>
        </SidebarGroup>
      ) : null}
      {loose.length ? (
        <SidebarGroup>
          <SidebarGroupLabel>{unassignedLabel}</SidebarGroupLabel>
          <SidebarGroupContent>
            <SidebarMenu>
              {loose.map((r) => (
                <SidebarMenuItem key={r.full_name || r.name}>
                  <RepoLink section={section} repo={r} pathname={pathname} filterRepo={filterRepo} itemIcon={itemIcon} />
                </SidebarMenuItem>
              ))}
            </SidebarMenu>
          </SidebarGroupContent>
        </SidebarGroup>
      ) : null}
    </>
  )
}
