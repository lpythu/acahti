import { PageFrame } from "@/components/page-frame"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { useT } from "@/i18n/i18n"
import { shortSha } from "@/lib/git"
import { useRepo } from "@/pages/repo-layout"

export function RepoCommitsPage() {
  const t = useT()
  const { data, error, loading } = useRepo()
  const commits = data?.commits || []

  return (
    <PageFrame
      loading={loading && !data}
      error={error}
      empty={!!data && commits.length === 0}
      emptyText={t("noRuns")}
      skeleton="table"
    >
      {commits.length ? (
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>{t("sha")}</TableHead>
              <TableHead>{t("author")}</TableHead>
              <TableHead>{t("title")}</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {commits.map((c) => (
              <TableRow key={c.sha}>
                <TableCell className="font-mono text-xs">{shortSha(c.sha)}</TableCell>
                <TableCell>{c.commit?.author?.name || "—"}</TableCell>
                <TableCell>{(c.commit?.message || "").split("\n")[0]}</TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      ) : null}
    </PageFrame>
  )
}
