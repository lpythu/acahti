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
import { useLoad } from "@/hooks/use-load"
import { useT } from "@/i18n/i18n"
import { api, codeGroupOf, groupRepos, splitRepo, type Repo } from "@/lib/api"
import { parsePipelinePath, repoBase } from "@/lib/nav"

export type CodeGroupSection = "repos" | "pipelines"

function groupHref(section: CodeGroupSection, group: string) {
  const base = section === "pipelines" ? "/pipelines" : "/repos"
  return `${base}?group=${encodeURIComponent(group)}`
}

function itemHref(section: CodeGroupSection, owner: string, name: string) {
  if (section === "pipelines") {
    return `/pipelines?repo=${encodeURIComponent(`${owner}/${name}`)}`
  }
  return `/repos/${owner}/${name}`
}

function GroupNode({
  section,
  group,
  repos,
  pathname,
  filterGroup,
  filterRepo,
  defaultOpen,
  itemIcon,
}: {
  section: CodeGroupSection
  group: string
  repos: Repo[]
  pathname: string
  filterGroup: string
  filterRepo: string
  defaultOpen: boolean
  itemIcon: ReactNode
}) {
  const pipe = parsePipelinePath(pathname)
  const inGroup = repos.some((r) => {
    const { owner, name } = splitRepo(r.full_name || r.name)
    if (filterRepo === `${owner}/${name}`) return true
    if (pipe?.owner === owner && pipe.name === name) return true
    const base = `/repos/${owner}/${name}`
    return pathname === base || pathname.startsWith(`${base}/`)
  })
  const groupActive = !filterRepo && !pipe && filterGroup === group && !repoBase(pathname)
  const [open, setOpen] = useState(defaultOpen || inGroup || groupActive)

  useEffect(() => {
    if (inGroup || groupActive) setOpen(true)
  }, [inGroup, groupActive])

  return (
    <SidebarMenuItem>
      <Collapsible open={open} onOpenChange={setOpen}>
        <div className="flex min-w-0 items-center">
          <SidebarMenuButton
            tooltip={group}
            isActive={groupActive}
            className="flex-1"
            render={<Link to={groupHref(section, group)} />}
          >
            <FolderIcon />
            <span>{group}</span>
          </SidebarMenuButton>
          <CollapsibleTrigger
            className={cn(
              "flex size-8 shrink-0 items-center justify-center rounded-md text-sidebar-foreground ring-sidebar-ring outline-hidden hover:bg-sidebar-accent hover:text-sidebar-accent-foreground focus-visible:ring-2 group-data-[collapsible=icon]:hidden",
            )}
            aria-label={group}
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

export function CodeGroupNav({ section }: { section: CodeGroupSection }) {
  const t = useT()
  const { pathname } = useLocation()
  const [sp] = useSearchParams()
  const filterGroup = sp.get("group") || ""
  const filterRepo = sp.get("repo") || ""
  const { data } = useLoad(async () => (await api.repos()).repos || [], [])
  const groups = groupRepos(data || [])
  const pipe = parsePipelinePath(pathname)
  const current = repoBase(pathname)
  const currentGroup = pipe
    ? codeGroupOf(pipe.name)
    : current
      ? codeGroupOf(current.split("/")[3] || "")
      : filterRepo
        ? codeGroupOf(splitRepo(filterRepo).name)
        : ""
  const openAll = groups.length <= 1 && !filterGroup && !currentGroup

  if (!groups.length) return null

  const itemIcon = section === "pipelines" ? <WorkflowIcon /> : <BookMarkedIcon />

  return (
    <SidebarGroup>
      <SidebarGroupLabel>{t("codeGroup")}</SidebarGroupLabel>
      <SidebarGroupContent>
        <SidebarMenu>
          {groups.map((g) => (
            <GroupNode
              key={g.group}
              section={section}
              group={g.group}
              repos={g.repos}
              pathname={pathname}
              filterGroup={filterGroup}
              filterRepo={filterRepo}
              defaultOpen={openAll || g.group === currentGroup}
              itemIcon={itemIcon}
            />
          ))}
        </SidebarMenu>
      </SidebarGroupContent>
    </SidebarGroup>
  )
}
