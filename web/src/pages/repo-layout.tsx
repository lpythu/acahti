import { Outlet, useOutletContext, useParams } from "react-router-dom"

import { RepoPanelHeader } from "@/components/repo-panel-header"
import { useLoad } from "@/hooks/use-load"
import { api, type RepoHeader } from "@/lib/api"

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

export function RepoLayout() {
  const { owner = "", name = "" } = useParams()
  const { data, error, loading, reload } = useLoad(() => api.repo(owner, name), [owner, name])

  return (
    <div className="flex min-h-0 flex-1 flex-col">
      <RepoPanelHeader owner={owner} name={name} team={data?.repo.team || ""} />
      {error && !data ? <p className="px-4 py-3 text-sm text-destructive">{error}</p> : null}
      <div className="flex min-h-0 flex-1 flex-col">
        <Outlet context={{ owner, name, data, error, loading, reload } satisfies RepoCtx} />
      </div>
    </div>
  )
}
