import { Link, useOutletContext, useParams } from "react-router-dom"

import { CopyField } from "@/components/copy-field"
import { PageFrame } from "@/components/page-frame"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { useLoad } from "@/hooks/use-load"
import { useT } from "@/i18n/i18n"
import { api, type Me } from "@/lib/api"

export function PackageDetailPage() {
  const t = useT()
  const me = useOutletContext<Me>()
  const { kind = "", "*": name = "" } = useParams()
  const { data, error, loading } = useLoad(() => api.packageGroup(kind, name), [kind, name])
  const root = me.root_url.replace(/\/$/, "")
  const org = me.org
  const install =
    kind === "npm"
      ? t("npmHint", { name, url: `${root}/api/packages/${org}/npm/` })
      : t("pipHint", { name, url: `${root}/api/packages/${org}/pypi/` })

  return (
    <PageFrame
      loading={loading && !data}
      error={error}
      empty={!!data && data.versions.length === 0}
      emptyText={t("noPackages")}
      skeleton="table"
      header={
        <>
          <p className="text-sm">
            <Link className="underline-offset-4 hover:underline" to="/packages">
              {t("packages")}
            </Link>
            <span className="text-muted-foreground">
              {" "}
              / {kind} / {name}
            </span>
          </p>
          <CopyField label={t("install")} value={install} />
        </>
      }
    >
      {data?.versions.length ? (
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>{t("version")}</TableHead>
              <TableHead>id</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {data.versions.map((v) => (
              <TableRow key={v.id}>
                <TableCell>{v.version}</TableCell>
                <TableCell className="font-mono text-xs">{v.id}</TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      ) : null}
    </PageFrame>
  )
}
