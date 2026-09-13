import { useNavigate } from "react-router-dom"

import { PageFrame } from "@/components/page-frame"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { useT } from "@/i18n/i18n"
import { shortSha } from "@/lib/git"
import { useRepo } from "@/pages/repo-layout"

export function RepoBranchesPage() {
  const t = useT()
  const nav = useNavigate()
  const { owner, name, data, error, loading } = useRepo()
  const branches = data?.branches || []

  return (
    <PageFrame
      loading={loading && !data}
      error={error}
      empty={!!data && branches.length === 0}
      emptyText={t("noRepos")}
      skeleton="table"
    >
      {branches.length ? (
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
            {branches.map((b) => (
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
                <TableCell>{b.default ? t("defaultBranch") : ""}</TableCell>
                <TableCell>{b.protected ? t("protected") : ""}</TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      ) : null}
    </PageFrame>
  )
}
