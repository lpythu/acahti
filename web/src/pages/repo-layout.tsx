import { NavLink, Outlet, useOutletContext, useParams } from "react-router-dom"
import { FileIcon, GitBranchIcon, GitPullRequestIcon, HistoryIcon, WorkflowIcon } from "lucide-react"
import { cn } from "cn"

import { CloneMenu } from "@/components/clone-menu"
import { useLoad } from "@/hooks/use-load"
import { useT } from "@/i18n/i18n"
import { api, type RepoOverview } from "@/lib/api"

export type RepoCtx = {
  owner: string
  name: string
  data: RepoOverview | null
  error: string
  loading: boolean
  reload: () => void
}

export function useRepo() {
  return useOutletContext<RepoCtx>()
}

export function RepoLayout() {
  const t = useT()
  const { owner = "", name = "" } = useParams()
  const { data, error, loading, reload } = useLoad(() => api.repo(owner, name), [owner, name])
  const base = `/repos/${owner}/${name}`
  const nav = [
    { to: base, end: true, label: t("files"), icon: FileIcon },
    { to: `${base}/commits`, label: t("commits"), icon: HistoryIcon },
    { to: `${base}/branches`, label: t("branches"), icon: GitBranchIcon },
    { to: `${base}/pulls`, label: t("tabPulls"), icon: GitPullRequestIcon },
    { to: `${base}/pipelines`, label: t("tabPipes"), icon: WorkflowIcon },
  ]

  return (
    <div className="flex min-h-0 flex-1">
      <nav className="flex w-44 shrink-0 flex-col gap-0.5 border-r p-2">
        {nav.map((item) => (
          <NavLink
            key={item.to}
            to={item.to}
            end={item.end}
            className={({ isActive }) =>
              cn(
                "flex items-center gap-2 rounded-md px-2 py-1.5 text-sm hover:bg-muted",
                isActive && "bg-muted font-medium",
              )
            }
          >
            <item.icon className="size-4 shrink-0" />
            {item.label}
          </NavLink>
        ))}
      </nav>
      <div className="flex min-w-0 flex-1 flex-col">
        <div className="flex items-center justify-between gap-3 border-b px-4 py-3 lg:px-6">
          <div className="min-w-0">
            <h2 className="truncate text-lg font-medium">{data?.repo.full_name || `${owner}/${name}`}</h2>
            <p className="text-sm text-muted-foreground">{t("protectNote")}</p>
          </div>
          {data ? <CloneMenu https={data.clone_https} ssh={data.clone_ssh} /> : null}
        </div>
        {error && !data ? <p className="px-4 py-3 text-sm text-destructive">{error}</p> : null}
        <Outlet context={{ owner, name, data, error, loading, reload } satisfies RepoCtx} />
      </div>
    </div>
  )
}
