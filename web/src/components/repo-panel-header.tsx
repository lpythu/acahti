import { Link, useLocation, useSearchParams } from "react-router-dom"

import {
  Breadcrumb,
  BreadcrumbItem,
  BreadcrumbLink,
  BreadcrumbList,
  BreadcrumbPage,
  BreadcrumbSeparator,
} from "@/components/ui/breadcrumb"
import { useT } from "@/i18n/i18n"
import type { MessageKey } from "@/i18n/messages"
import { shortSha } from "@/lib/git"

const SECTION_KEY: Record<string, MessageKey> = {
  commits: "commits",
  branches: "branches",
  tags: "tags",
  pulls: "tabPulls",
  pipelines: "tabPipes",
  access: "access",
  secrets: "tabSecrets",
}

function RepoBreadcrumb({ owner, name, team }: { owner: string; name: string; team: string }) {
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
        {team ? (
          <>
            <BreadcrumbItem>
              <BreadcrumbLink render={<Link to={`/repos?team=${encodeURIComponent(team)}`} />}>{team}</BreadcrumbLink>
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

export function RepoPanelHeader({
  owner,
  name,
  team,
}: {
  owner: string
  name: string
  team: string
}) {
  return (
    <div className="flex h-12 shrink-0 items-center border-b px-4 lg:px-6">
      <div className="min-w-0">
        <RepoBreadcrumb owner={owner} name={name} team={team} />
      </div>
    </div>
  )
}
