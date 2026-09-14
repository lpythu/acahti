import { useMemo, useState } from "react"
import { Link, useNavigate, useSearchParams } from "react-router-dom"
import { PlayIcon } from "lucide-react"

import { PageFrame } from "@/components/page-frame"
import { PipelineStages } from "@/components/pipeline-stages"
import { RunStatusIcon } from "@/components/run-status-icon"
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar"
import { Button } from "@/components/ui/button"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { useLoad } from "@/hooks/use-load"
import { useT } from "@/i18n/i18n"
import { api, codeGroupOf, splitRepo, type Pipeline } from "@/lib/api"
import { formatUnix } from "@/lib/format"
import { pipelineHref } from "@/lib/nav"
import { latestByRepo, stagesOf, triggerKey, triggerVars, withWorkflows } from "@/lib/pipeline"

function RunStatus({ pipe }: { pipe: Pipeline }) {
  return (
    <span className="inline-flex items-center gap-1.5 tabular-nums">
      <span>#{pipe.number}</span>
      <RunStatusIcon status={pipe.status} />
    </span>
  )
}

export function PipelinesPage() {
  const t = useT()
  const nav = useNavigate()
  const [sp] = useSearchParams()
  const group = sp.get("group") || ""
  const repo = sp.get("repo") || ""
  const { data, error, loading } = useLoad(async () => withWorkflows(latestByRepo((await api.pipelines()).pipes || [])), [])
  const [busy, setBusy] = useState("")
  const [actionErr, setActionErr] = useState("")

  const rows = useMemo(() => {
    return (data || []).filter((p) => {
      const { owner, name } = splitRepo(p.repo)
      if (repo) return p.repo === repo
      if (group) return codeGroupOf(name) === group
      return Boolean(owner && name)
    })
  }, [data, group, repo])

  async function runPipe(p: Pipeline) {
    const { owner, name } = splitRepo(p.repo)
    setBusy(p.repo)
    setActionErr("")
    try {
      const next = await api.trigger(owner, name, p.branch || "dev")
      nav(pipelineHref(owner, name, next.number))
    } catch (err) {
      setActionErr(err instanceof Error ? err.message : t("loadError"))
      setBusy("")
    }
  }

  return (
    <PageFrame
      loading={loading && !data}
      error={error || actionErr}
      empty={!!data && rows.length === 0}
      emptyText={t("noRuns")}
      header={<p className="text-sm text-muted-foreground">{group || repo || t("pipelinesDesc")}</p>}
      skeleton="table"
    >
      {rows.length ? (
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>{t("pipelineName")}</TableHead>
              <TableHead>{t("latestRunStatus")}</TableHead>
              <TableHead>{t("latestRunStages")}</TableHead>
              <TableHead>{t("triggerInfo")}</TableHead>
              <TableHead>{t("latestRunStarted")}</TableHead>
              <TableHead className="w-10" />
            </TableRow>
          </TableHeader>
          <TableBody>
            {rows.map((p) => {
              const { owner, name } = splitRepo(p.repo)
              const href = pipelineHref(owner, name, p.number)
              return (
                <TableRow key={p.repo}>
                  <TableCell>
                    <Link className="hover:underline" to={href}>
                      {name}
                    </Link>
                  </TableCell>
                  <TableCell>
                    <Link className="hover:underline" to={href}>
                      <RunStatus pipe={p} />
                    </Link>
                  </TableCell>
                  <TableCell>
                    <PipelineStages stages={stagesOf(p)} />
                  </TableCell>
                  <TableCell>
                    <div className="flex min-w-0 items-center gap-2">
                      <Avatar size="sm">
                        {p.avatar ? <AvatarImage src={p.avatar} alt={p.author || name} /> : null}
                        <AvatarFallback>{(p.author || name).slice(0, 1).toUpperCase()}</AvatarFallback>
                      </Avatar>
                      <span className="min-w-0 truncate">{t(triggerKey(p.event), triggerVars(p))}</span>
                    </div>
                  </TableCell>
                  <TableCell className="tabular-nums text-muted-foreground">
                    {formatUnix(p.started || p.created)}
                  </TableCell>
                  <TableCell>
                    <Button
                      type="button"
                      size="icon-xs"
                      variant="ghost"
                      disabled={busy === p.repo}
                      aria-label={t("run")}
                      onClick={() => void runPipe(p)}
                    >
                      <PlayIcon />
                    </Button>
                  </TableCell>
                </TableRow>
              )
            })}
          </TableBody>
        </Table>
      ) : null}
    </PageFrame>
  )
}
