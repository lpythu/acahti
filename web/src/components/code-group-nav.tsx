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
  pathname,
  filterGroup,
  filterRepo,
  defaultOpen,
  itemIcon,
}: {
  section: CodeGroupSection
  group: string
  pathname: string
  filterGroup: string
  filterRepo: string
  defaultOpen: boolean
  itemIcon: ReactNode
}) {
  const pipe = parsePipelinePath(pathname)
  const groupActive = !filterRepo && !pipe && filterGroup === group && !repoBase(pathname)
  const [open, setOpen] = useState(defaultOpen || groupActive)
  const repos = usePage((q) => api.repos(q, group), [group], {
    url: false,
    enabled: open,
  })
  const inGroup = repos.items.some((r) => {
    const { owner, name } = splitRepo(r.full_name || r.name)
    if (filterRepo === `${owner}/${name}`) return true
    if (pipe?.owner === owner && pipe.name === name) return true
    const base = `/repos/${owner}/${name}`
    return pathname === base || pathname.startsWith(`${base}/`)
  })

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

export function CodeGroupNav({ section }: { section: CodeGroupSection }) {
  const t = useT()
  const { pathname } = useLocation()
  const [sp] = useSearchParams()
  const filterGroup = sp.get("group") || ""
  const filterRepo = sp.get("repo") || ""
  const groups = usePage((q) => api.repoGroups(q), [], { url: false })
  const pipe = parsePipelinePath(pathname)
  const current = repoBase(pathname)
  const headOwner = pipe?.owner || (current ? current.split("/")[2] : splitRepo(filterRepo).owner)
  const headName = pipe?.name || (current ? current.split("/")[3] : splitRepo(filterRepo).name)
  const head = useLoad(
    () => api.repo(headOwner, headName),
    [headOwner, headName],
    Boolean(headOwner && headName),
  )
  const currentGroup = filterGroup || head.data?.repo.group || ""
  const openAll = groups.items.length <= 1 && !filterGroup && !currentGroup

  if (!groups.items.length) return null

  const itemIcon = section === "pipelines" ? <WorkflowIcon /> : <BookMarkedIcon />

  return (
    <SidebarGroup>
      <SidebarGroupLabel>{t("codeGroup")}</SidebarGroupLabel>
      <SidebarGroupContent>
        <SidebarMenu>
          {groups.items.map((g) => (
            <GroupNode
              key={g.group}
              section={section}
              group={g.group}
              pathname={pathname}
              filterGroup={filterGroup}
              filterRepo={filterRepo}
              defaultOpen={openAll || g.group === currentGroup}
              itemIcon={itemIcon}
            />
          ))}
          <MoreButton page={groups.page} hasMore={groups.hasMore} onPage={groups.setPage} />
        </SidebarMenu>
      </SidebarGroupContent>
    </SidebarGroup>
  )
}
