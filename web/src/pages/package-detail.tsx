import { Link, useOutletContext, useParams } from "react-router-dom"

import { CopyField } from "@/components/copy-field"
import { PagedList } from "@/components/paged-list"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { usePage } from "@/hooks/use-page"
import { useT } from "@/i18n/i18n"
import { api, type Me } from "@/lib/api"

export function PackageDetailPage() {
  const t = useT()
  const me = useOutletContext<Me>()
  const { kind = "", "*": name = "" } = useParams()
  const list = usePage(
    (q) => api.packageVersions(kind, name, q),
    [kind, name],
  )
  const root = me.root_url.replace(/\/$/, "")
  const org = me.org
  const host = root.replace(/^https?:\/\//, "")
  const user = me.user
  const install =
    kind === "npm"
      ? t("npmHint", { name, url: `${root}/api/packages/${org}/npm/`, host, org, user })
      : t("pipHint", { name, url: `${root}/api/packages/${org}/pypi/`, host, org, user })

  return (
    <PagedList
      list={list}
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
          <CopyField label={t("install")} value={install} multiline />
        </>
      }
    >
      {(items) => (
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>{t("version")}</TableHead>
              <TableHead>id</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {items.map((v) => (
              <TableRow key={v.id}>
                <TableCell>{v.version}</TableCell>
                <TableCell className="font-mono text-xs">{v.id}</TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      )}
    </PagedList>
  )
}
