import { useMemo, useState, type ReactNode } from "react"
import { Link, useLocation, useSearchParams } from "react-router-dom"
import { toast } from "sonner"
import {
  DndContext,
  DragOverlay,
  PointerSensor,
  closestCenter,
  pointerWithin,
  useDraggable,
  useDroppable,
  useSensor,
  useSensors,
  type CollisionDetection,
  type DragEndEvent,
  type DragStartEvent,
} from "@dnd-kit/core"
import { cn } from "cn"
import { BookMarkedIcon, ChevronRightIcon, FolderIcon, ListChevronsDownUpIcon, ListChevronsUpDownIcon, WorkflowIcon } from "lucide-react"

import { Collapsible, CollapsibleContent } from "@/components/ui/collapsible"
import {
  SidebarGroup,
  SidebarGroupAction,
  SidebarGroupContent,
  SidebarGroupLabel,
  SidebarInput,
  SidebarMenu,
  SidebarMenuBadge,
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
import { useSession } from "@/lib/session"

export type CodeTeamSection = "repos" | "pipelines"

const UNASSIGNED = ""

function teamDropId(team: string) {
  return `team:${team}`
}

function parseTeamDrop(id: string) {
  return id.startsWith("team:") ? id.slice(5) : null
}

const dropCollision: CollisionDetection = (args) => {
  const hits = pointerWithin(args)
  return hits.length ? hits : closestCenter(args)
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

function repoKey(r: Repo) {
  return r.full_name || r.name
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

function moveInTree(tree: NavTeam[] | null, fullName: string, from: string, to: string): NavTeam[] | null {
  if (!tree || from === to) return tree
  let moved: Repo | undefined
  const next = tree.map((row) => {
    if (row.team !== from) return row
    return {
      ...row,
      repos: row.repos.filter((r) => {
        if (repoKey(r) === fullName) {
          moved = r
          return false
        }
        return true
      }),
    }
  })
  if (!moved) return tree
  let found = false
  const out = next.map((row) => {
    if (row.team !== to) return row
    found = true
    return { ...row, repos: [...row.repos, moved!] }
  })
  if (!found) out.push({ team: to, repos: [moved] })
  return out.filter((row) => row.team || row.repos.length)
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

function RepoRow({
  section,
  repo,
  from,
  pathname,
  filterRepo,
  itemIcon,
  nested,
  canDrag,
}: {
  section: CodeTeamSection
  repo: Repo
  from: string
  pathname: string
  filterRepo: string
  itemIcon: ReactNode
  nested?: boolean
  canDrag: boolean
}) {
  const fullName = repoKey(repo)
  const { listeners, setNodeRef, isDragging } = useDraggable({
    id: `repo:${fullName}`,
    data: { from, fullName },
    disabled: !canDrag,
  })
  const Item = nested ? SidebarMenuSubItem : SidebarMenuItem
  return (
    <Item
      ref={setNodeRef}
      className={cn(canDrag && "cursor-grab active:cursor-grabbing", isDragging && "opacity-40")}
      {...listeners}
    >
      <RepoLink section={section} repo={repo} pathname={pathname} filterRepo={filterRepo} itemIcon={itemIcon} nested={nested} />
    </Item>
  )
}

function UnassignedGroup({
  section,
  repos,
  pathname,
  filterRepo,
  itemIcon,
  canDrag,
  label,
}: {
  section: CodeTeamSection
  repos: Repo[]
  pathname: string
  filterRepo: string
  itemIcon: ReactNode
  canDrag: boolean
  label: string
}) {
  const { setNodeRef, isOver } = useDroppable({ id: teamDropId(UNASSIGNED), disabled: !canDrag })
  return (
    <SidebarGroup ref={setNodeRef} className={cn(isOver && "rounded-md bg-sidebar-accent")}>
      <SidebarGroupLabel>{label}</SidebarGroupLabel>
      <SidebarGroupContent>
        <SidebarMenu>
          {repos.map((r) => (
            <RepoRow
              key={repoKey(r)}
              section={section}
              repo={r}
              from={UNASSIGNED}
              pathname={pathname}
              filterRepo={filterRepo}
              itemIcon={itemIcon}
              canDrag={canDrag}
            />
          ))}
        </SidebarMenu>
      </SidebarGroupContent>
    </SidebarGroup>
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
  canDrag,
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
  canDrag: boolean
}) {
  const t = useT()
  const { setNodeRef, isOver } = useDroppable({ id: teamDropId(team), disabled: !canDrag })
  const pipe = parsePipelinePath(pathname)
  const teamActive = !filterRepo && !pipe && filterTeam === team && !repoBase(pathname)
  const inTeam = repos.some((r) => {
    const { owner, name } = splitRepo(r.full_name || r.name)
    return repoActive(pathname, filterRepo, owner, name)
  })
  const open = forceOpen || isOver || (openProp ?? (inTeam || teamActive))
  const n = repos.length

  return (
    <SidebarMenuItem ref={setNodeRef} className={cn(isOver && "rounded-md bg-sidebar-accent")}>
      <Collapsible open={open} onOpenChange={onOpenChange}>
        <SidebarMenuButton tooltip={team} isActive={teamActive} onClick={() => onOpenChange(!open)}>
          <FolderIcon />
          <span className="min-w-0 flex-1 truncate">{team}</span>
          <SidebarMenuBadge title={t("reposCount", { n })}>{n}</SidebarMenuBadge>
          <ChevronRightIcon className={cn("size-4 shrink-0 transition-transform", open && "rotate-90")} />
        </SidebarMenuButton>
        <CollapsibleContent>
          <SidebarMenuSub>
            {repos.map((r) => (
              <RepoRow
                key={repoKey(r)}
                section={section}
                repo={r}
                from={team}
                pathname={pathname}
                filterRepo={filterRepo}
                itemIcon={itemIcon}
                nested
                canDrag={canDrag}
              />
            ))}
          </SidebarMenuSub>
        </CollapsibleContent>
      </Collapsible>
    </SidebarMenuItem>
  )
}

export function CodeTeamNav({ section }: { section: CodeTeamSection }) {
  const t = useT()
  const { me } = useSession()
  const canDrag = Boolean(me?.admin)
  const { pathname } = useLocation()
  const [sp] = useSearchParams()
  const [q, setQ] = useState("")
  const [dragging, setDragging] = useState("")
  const filterTeam = sp.get("team") || ""
  const filterRepo = sp.get("repo") || ""
  const tree = useLoad(() => api.navTree(), [])
  useEvents((ev) => {
    if (ev.type === "catalog.updated") void tree.reload()
  })

  const sensors = useSensors(useSensor(PointerSensor, { activationConstraint: { distance: 6 } }))
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

  function onDragStart(e: DragStartEvent) {
    const fullName = String(e.active.data.current?.fullName || "")
    setDragging(repoName(fullName))
  }

  async function onDragEnd(e: DragEndEvent) {
    setDragging("")
    const fullName = String(e.active.data.current?.fullName || "")
    const from = String(e.active.data.current?.from ?? "")
    const to = e.over ? parseTeamDrop(String(e.over.id)) : null
    if (!fullName || to === null || from === to) return
    const { owner, name } = splitRepo(fullName)
    tree.apply((cur) => moveInTree(cur, fullName, from, to))
    try {
      if (to) await api.moveRepo(owner, name, to, from)
      else await api.removeTeamRepo(from, name)
      toast.success(t("repoMoved"))
      await tree.reload()
    } catch (err) {
      toast.error(err instanceof Error ? err.message : t("loadError"))
      await tree.reload()
    }
  }

  const body = (
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
                  canDrag={canDrag}
                />
              ))}
            </SidebarMenu>
          </SidebarGroupContent>
        </SidebarGroup>
      ) : null}
      {loose.length || (canDrag && dragging) ? (
        <UnassignedGroup
          section={section}
          repos={loose}
          pathname={pathname}
          filterRepo={filterRepo}
          itemIcon={itemIcon}
          canDrag={canDrag}
          label={unassignedLabel}
        />
      ) : null}
    </>
  )

  return (
    <DndContext sensors={sensors} collisionDetection={dropCollision} onDragStart={onDragStart} onDragEnd={(e) => void onDragEnd(e)}>
      {body}
      <DragOverlay>{dragging ? <p className="rounded-md bg-sidebar-accent px-2 py-1 text-sm shadow-sm">{dragging}</p> : null}</DragOverlay>
    </DndContext>
  )
}
