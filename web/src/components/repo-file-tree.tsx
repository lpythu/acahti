import { useState } from "react"
import { ChevronRightIcon, FileIcon, FolderIcon } from "lucide-react"
import { cn } from "cn"

import { useLoad } from "@/hooks/use-load"
import { api, type ContentEntry } from "@/lib/api"

function sortEntries(entries: ContentEntry[]) {
  return entries.slice().sort((a, b) => {
    if (a.type === b.type) return a.name.localeCompare(b.name)
    return a.type === "dir" ? -1 : 1
  })
}

async function loadAllEntries(owner: string, name: string, gitRef: string, path: string) {
  const all: ContentEntry[] = []
  for (let page = 1; ; page++) {
    const res = await api.contents(owner, name, {
      page,
      page_size: 50,
      ref: gitRef || undefined,
      path: path || undefined,
    })
    all.push(...(res.items || []))
    if (!res.has_more) break
  }
  return sortEntries(all)
}

function TreeDir({
  owner,
  name,
  gitRef,
  path,
  selected,
  depth,
  onPick,
}: {
  owner: string
  name: string
  gitRef: string
  path: string
  selected: string
  depth: number
  onPick: (path: string, dir: boolean) => void
}) {
  const list = useLoad(() => loadAllEntries(owner, name, gitRef, path), [owner, name, gitRef, path])
  const entries = list.data || []
  return (
    <ul>
      {entries.map((e) => (
        <TreeNode
          key={e.path}
          owner={owner}
          name={name}
          gitRef={gitRef}
          entry={e}
          selected={selected}
          depth={depth}
          onPick={onPick}
        />
      ))}
    </ul>
  )
}

function TreeNode({
  owner,
  name,
  gitRef,
  entry,
  selected,
  depth,
  onPick,
}: {
  owner: string
  name: string
  gitRef: string
  entry: ContentEntry
  selected: string
  depth: number
  onPick: (path: string, dir: boolean) => void
}) {
  const dir = entry.type === "dir"
  const active = selected === entry.path
  const childActive = selected.startsWith(`${entry.path}/`)
  const [open, setOpen] = useState(childActive)

  return (
    <li>
      <div className="flex min-w-0 items-center">
        {dir ? (
          <button
            type="button"
            className="flex size-6 shrink-0 items-center justify-center text-muted-foreground hover:text-foreground"
            style={{ marginLeft: depth * 12 }}
            aria-expanded={open}
            onClick={() => setOpen((v) => !v)}
          >
            <ChevronRightIcon className={cn("size-3.5 transition-transform", open && "rotate-90")} />
          </button>
        ) : (
          <span className="size-6 shrink-0" style={{ marginLeft: depth * 12 }} />
        )}
        <button
          type="button"
          className={cn(
            "flex min-w-0 flex-1 items-center gap-1.5 rounded-md px-1.5 py-1 text-left text-sm hover:bg-muted",
            active && "bg-muted font-medium",
          )}
          onClick={() => {
            if (dir) setOpen(true)
            onPick(entry.path, dir)
          }}
        >
          {dir ? (
            <FolderIcon className="size-4 shrink-0 text-muted-foreground" />
          ) : (
            <FileIcon className="size-4 shrink-0 text-muted-foreground" />
          )}
          <span className="truncate">{entry.name}</span>
        </button>
      </div>
      {dir && open ? (
        <TreeDir
          owner={owner}
          name={name}
          gitRef={gitRef}
          path={entry.path}
          selected={selected}
          depth={depth + 1}
          onPick={onPick}
        />
      ) : null}
    </li>
  )
}

export function RepoFileTree({
  owner,
  name,
  gitRef,
  path,
  selected,
  onPick,
}: {
  owner: string
  name: string
  gitRef: string
  path: string
  selected: string
  onPick: (path: string, dir: boolean) => void
}) {
  return (
    <TreeDir
      owner={owner}
      name={name}
      gitRef={gitRef}
      path={path}
      selected={selected}
      depth={0}
      onPick={onPick}
    />
  )
}
