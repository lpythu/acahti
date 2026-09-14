import { useMemo } from "react"
import { Link, useSearchParams } from "react-router-dom"
import { BoxIcon } from "lucide-react"

import { PageFrame } from "@/components/page-frame"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { useLoad } from "@/hooks/use-load"
import { useT } from "@/i18n/i18n"
import { api, type Package, type PackageRow } from "@/lib/api"
import { formatSize, formatStamp } from "@/lib/format"
import { packageHref } from "@/lib/nav"

function asRows(pkgs: PackageRow[]): PackageRow[] {
  if (pkgs.some((p) => p.latest)) return pkgs
  const m = new Map<string, PackageRow>()
  for (const raw of pkgs as unknown as Package[]) {
    const k = `${raw.type}/${raw.name}`
    const cur = m.get(k)
    if (!cur) {
      m.set(k, {
        type: raw.type,
        name: raw.name,
        latest: raw.version,
        updated_at: raw.created_at || "",
        size: 0,
        versions: 1,
        downloads: 0,
      })
    } else {
      cur.versions += 1
      if (!cur.latest) cur.latest = raw.version
    }
  }
  return [...m.values()]
}

export function PackagesPage() {
  const t = useT()
  const [sp] = useSearchParams()
  const kind = sp.get("kind") || ""
  const { data, error, loading } = useLoad(async () => (await api.packages()).packages || [], [])
  const rows = useMemo(() => asRows(data || []).filter((r) => !kind || r.type === kind), [data, kind])

  return (
    <PageFrame
      loading={loading && !data}
      error={error}
      empty={!!data && rows.length === 0}
      emptyText={t("noPackages")}
      header={<p className="text-sm text-muted-foreground">{t("packagesDesc")}</p>}
      skeleton="table"
    >
      {rows.length ? (
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>{t("packageName")}</TableHead>
              <TableHead>{t("latestVersion")}</TableHead>
              <TableHead>{t("lastUpdated")}</TableHead>
              <TableHead>{t("packageSize")}</TableHead>
              <TableHead>{t("versionCount")}</TableHead>
              <TableHead>{t("downloads")}</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {rows.map((r) => (
              <TableRow key={`${r.type}-${r.name}`}>
                <TableCell>
                  <Link className="inline-flex items-center gap-2 hover:underline" to={packageHref(r.type, r.name)}>
                    <BoxIcon className="size-4 shrink-0 text-muted-foreground" />
                    <span>{r.name}</span>
                  </Link>
                </TableCell>
                <TableCell>{r.latest || "—"}</TableCell>
                <TableCell>{formatStamp(r.updated_at)}</TableCell>
                <TableCell>{formatSize(r.size)}</TableCell>
                <TableCell>{r.versions || "—"}</TableCell>
                <TableCell>{r.downloads ? r.downloads : "—"}</TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      ) : null}
    </PageFrame>
  )
}
