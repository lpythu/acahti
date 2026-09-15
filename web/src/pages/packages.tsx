import { Link, useSearchParams } from "react-router-dom"
import { BoxIcon } from "lucide-react"

import { PagedList } from "@/components/paged-list"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { usePage } from "@/hooks/use-page"
import { useT } from "@/i18n/i18n"
import { api } from "@/lib/api"
import { formatStamp } from "@/lib/format"
import { packageHref } from "@/lib/nav"

export function PackagesPage() {
  const t = useT()
  const [sp] = useSearchParams()
  const kind = sp.get("kind") || ""
  const list = usePage((q) => api.packages(q, kind), [kind])

  return (
    <PagedList
      list={list}
      emptyText={t("noPackages")}
      skeleton="table"
    >
      {(items) => (
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>{t("packageName")}</TableHead>
              <TableHead>{t("latestVersion")}</TableHead>
              <TableHead>{t("lastUpdated")}</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {items.map((r) => (
              <TableRow key={`${r.type}-${r.name}`}>
                <TableCell>
                  <Link className="inline-flex items-center gap-2 hover:underline" to={packageHref(r.type, r.name)}>
                    <BoxIcon className="size-4 shrink-0 text-muted-foreground" />
                    <span>{r.name}</span>
                  </Link>
                </TableCell>
                <TableCell>{r.latest || "—"}</TableCell>
                <TableCell>{formatStamp(r.updated_at)}</TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      )}
    </PagedList>
  )
}
