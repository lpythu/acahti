import { useState } from "react"
import { useNavigate } from "react-router-dom"
import { toast } from "sonner"

import { PagedList } from "@/components/paged-list"
import { Button } from "@/components/ui/button"
import { Switch } from "@/components/ui/switch"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { usePage } from "@/hooks/use-page"
import { useT } from "@/i18n/i18n"
import { api } from "@/lib/api"
import { shortSha } from "@/lib/git"
import { useSession } from "@/lib/session"
import { useRepo } from "@/pages/repo-layout"

export function RepoBranchesPage() {
  const t = useT()
  const nav = useNavigate()
  const { me } = useSession()
  const { owner, name, data, reload: reloadRepo } = useRepo()
  const list = usePage((q) => api.branches(owner, name, q), [owner, name])
  const manage = Boolean(me?.admin || data?.repo.permissions?.admin)
  const [busy, setBusy] = useState("")

  async function patch(branch: string, body: { default?: boolean; protected?: boolean }) {
    setBusy(branch)
    try {
      await api.patchBranch(owner, name, branch, body)
      await Promise.all([list.reload(), reloadRepo()])
    } catch (e) {
      toast.error(e instanceof Error ? e.message : t("loadError"))
    } finally {
      setBusy("")
    }
  }

  return (
    <PagedList list={list} emptyText={t("noRepos")} skeleton="table">
      {(items) => (
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>{t("branches")}</TableHead>
              <TableHead>{t("sha")}</TableHead>
              <TableHead>{t("defaultBranch")}</TableHead>
              <TableHead>{t("protected")}</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {items.map((b) => (
              <TableRow key={b.name}>
                <TableCell>
                  <button
                    className="hover:underline"
                    type="button"
                    onClick={() => nav(`/repos/${owner}/${name}?ref=${encodeURIComponent(b.name)}`)}
                  >
                    {b.name}
                  </button>
                </TableCell>
                <TableCell className="font-mono text-xs">{shortSha(b.sha)}</TableCell>
                <TableCell>
                  {b.default ? (
                    <span className="text-sm">{t("defaultBranch")}</span>
                  ) : manage ? (
                    <Button
                      type="button"
                      size="sm"
                      variant="outline"
                      disabled={busy === b.name}
                      onClick={() => void patch(b.name, { default: true })}
                    >
                      {t("setDefaultBranch")}
                    </Button>
                  ) : null}
                </TableCell>
                <TableCell>
                  {manage ? (
                    <Switch
                      size="sm"
                      checked={b.protected}
                      disabled={busy === b.name}
                      aria-label={t("protected")}
                      onCheckedChange={(v) => void patch(b.name, { protected: v === true })}
                    />
                  ) : b.protected ? (
                    t("protected")
                  ) : null}
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      )}
    </PagedList>
  )
}
