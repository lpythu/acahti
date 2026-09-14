import { Link, Outlet, useLocation, useOutletContext, useParams, useSearchParams } from "react-router-dom"

import { CloneMenu } from "@/components/clone-menu"
import {
  Breadcrumb,
  BreadcrumbItem,
  BreadcrumbLink,
  BreadcrumbList,
  BreadcrumbPage,
  BreadcrumbSeparator,
} from "@/components/ui/breadcrumb"
import { useLoad } from "@/hooks/use-load"
import { useT } from "@/i18n/i18n"
import type { MessageKey } from "@/i18n/messages"
import { api, type RepoHeader } from "@/lib/api"
import { shortSha } from "@/lib/git"

export type RepoCtx = {
  owner: string
  name: string
  data: RepoHeader | null
  error: string
  loading: boolean
  reload: () => void
}

export function useRepo() {
  return useOutletContext<RepoCtx>()
}

const SECTION_KEY: Record<string, MessageKey> = {
  commits: "commits",
  branches: "branches",
  pulls: "tabPulls",
  pipelines: "tabPipes",
}

function RepoBreadcrumb({ owner, name, group }: { owner: string; name: string; group: string }) {
  const t = useT()
  const { pathname } = useLocation()
  const [sp] = useSearchParams()
  const base = `/repos/${owner}/${name}`
  const extra = pathname.slice(base.length).replace(/^\//, "")
  const [section, ...rest] = extra ? extra.split("/") : []
  const filePath = section ? "" : sp.get("path") || ""
  const fileParts = filePath ? filePath.split("/").filter(Boolean) : []
  const sectionKey = section ? SECTION_KEY[section] : undefined
  const onRepoRoot = !section && fileParts.length === 0

  return (
    <Breadcrumb>
      <BreadcrumbList>
        <BreadcrumbItem>
          <BreadcrumbLink render={<Link to="/repos" />}>{t("repos")}</BreadcrumbLink>
        </BreadcrumbItem>
        <BreadcrumbSeparator />
        {group ? (
          <>
            <BreadcrumbItem>
              <BreadcrumbLink render={<Link to={`/repos?group=${encodeURIComponent(group)}`} />}>{group}</BreadcrumbLink>
            </BreadcrumbItem>
            <BreadcrumbSeparator />
          </>
        ) : null}
        <BreadcrumbItem>
          {onRepoRoot ? (
            <BreadcrumbPage>{name}</BreadcrumbPage>
          ) : (
            <BreadcrumbLink render={<Link to={base} />}>{name}</BreadcrumbLink>
          )}
        </BreadcrumbItem>
        {sectionKey ? (
          <>
            <BreadcrumbSeparator />
            <BreadcrumbItem>
              {rest.length ? (
                <BreadcrumbLink render={<Link to={`${base}/${section}`} />}>{t(sectionKey)}</BreadcrumbLink>
              ) : (
                <BreadcrumbPage>{t(sectionKey)}</BreadcrumbPage>
              )}
            </BreadcrumbItem>
          </>
        ) : null}
        {rest.map((part, i) => {
          const last = i === rest.length - 1
          const to = `${base}/${section}/${rest.slice(0, i + 1).join("/")}`
          return (
            <span key={to} className="contents">
              <BreadcrumbSeparator />
              <BreadcrumbItem>
                {last ? (
                  <BreadcrumbPage>
                    {section === "commits" ? shortSha(part) : part.startsWith("#") ? part : `#${part}`}
                  </BreadcrumbPage>
                ) : (
                  <BreadcrumbLink render={<Link to={to} />}>{part}</BreadcrumbLink>
                )}
              </BreadcrumbItem>
            </span>
          )
        })}
        {fileParts.map((part, i) => {
          const last = i === fileParts.length - 1
          const path = fileParts.slice(0, i + 1).join("/")
          const q = new URLSearchParams(sp)
          q.set("path", path)
          return (
            <span key={path} className="contents">
              <BreadcrumbSeparator />
              <BreadcrumbItem>
                {last ? (
                  <BreadcrumbPage>{part}</BreadcrumbPage>
                ) : (
                  <BreadcrumbLink render={<Link to={`${base}?${q.toString()}`} />}>{part}</BreadcrumbLink>
                )}
              </BreadcrumbItem>
            </span>
          )
        })}
      </BreadcrumbList>
    </Breadcrumb>
  )
}

export function RepoLayout() {
  const { owner = "", name = "" } = useParams()
  const { data, error, loading, reload } = useLoad(() => api.repo(owner, name), [owner, name])

  return (
    <div className="flex min-h-0 flex-1 flex-col">
      <div className="flex items-center justify-between gap-3 border-b px-4 py-3 lg:px-6">
        <div className="min-w-0">
          <RepoBreadcrumb owner={owner} name={name} group={data?.repo.group || ""} />
        </div>
        {data ? <CloneMenu https={data.clone_https} ssh={data.clone_ssh} /> : null}
      </div>
      {error && !data ? <p className="px-4 py-3 text-sm text-destructive">{error}</p> : null}
      <div className="flex min-h-0 flex-1 flex-col">
        <Outlet context={{ owner, name, data, error, loading, reload } satisfies RepoCtx} />
      </div>
    </div>
  )
}
