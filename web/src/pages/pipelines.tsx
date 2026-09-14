import { useState } from "react"
import { Link, useNavigate, useSearchParams } from "react-router-dom"
import { PlayIcon } from "lucide-react"

import { PagedList } from "@/components/paged-list"
import { PipelineStages } from "@/components/pipeline-stages"
import { RunStatusIcon } from "@/components/run-status-icon"
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar"
import { Button } from "@/components/ui/button"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { usePage } from "@/hooks/use-page"
import { useT } from "@/i18n/i18n"
import { api, splitRepo, type Pipeline } from "@/lib/api"
import { formatUnix } from "@/lib/format"
import { pipelineHref } from "@/lib/nav"
import { jobDotsOf, triggerKey, triggerVars } from "@/lib/pipeline"

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
  const list = usePage((q) => api.pipelines({ ...q, repo, group }), [repo, group])
  const [busy, setBusy] = useState("")
  const [actionErr, setActionErr] = useState("")

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
    <PagedList
      list={{ ...list, error: list.error || actionErr }}
      emptyText={t("noPipelines")}
      header={<p className="text-sm text-muted-foreground">{group || repo || t("pipelinesDesc")}</p>}
      skeleton="table"
    >
      {(items) => (
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>{t("pipelineName")}</TableHead>
              <TableHead>{t("latestPipelineStatus")}</TableHead>
              <TableHead>{t("jobs")}</TableHead>
              <TableHead>{t("triggerInfo")}</TableHead>
              <TableHead>{t("latestPipelineStarted")}</TableHead>
              <TableHead className="w-10" />
            </TableRow>
          </TableHeader>
          <TableBody>
            {items.map((p) => {
              const { owner, name } = splitRepo(p.repo)
              const href = pipelineHref(owner, name, p.number)
              return (
                <TableRow key={`${p.repo}-${p.number}`}>
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
                    <PipelineStages stages={jobDotsOf(p)} />
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
      )}
    </PagedList>
  )
}
