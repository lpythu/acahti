import { useMemo } from "react"
import { Link } from "react-router-dom"

import { PageFrame } from "@/components/page-frame"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { useLoad } from "@/hooks/use-load"
import { useT } from "@/i18n/i18n"
import { api } from "@/lib/api"

type Row = { type: string; name: string; latest: string; count: number }

export function PackagesPage() {
  const t = useT()
  const { data, error, loading } = useLoad(async () => (await api.packages()).packages || [], [])
  const rows = useMemo(() => {
    const m = new Map<string, Row>()
    for (const p of data || []) {
      const k = `${p.type}/${p.name}`
      const cur = m.get(k)
      if (!cur) {
        m.set(k, { type: p.type, name: p.name, latest: p.version, count: 1 })
      } else {
        cur.count += 1
        if (!cur.latest) cur.latest = p.version
      }
    }
    return [...m.values()]
  }, [data])

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
              <TableHead>{t("name")}</TableHead>
              <TableHead>{t("kind")}</TableHead>
              <TableHead>{t("latest")}</TableHead>
              <TableHead>{t("versions")}</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {rows.map((r) => (
              <TableRow key={`${r.type}-${r.name}`}>
                <TableCell>
                  <Link className="hover:underline" to={`/packages/${r.type}/${r.name}`}>
                    {r.name}
                  </Link>
                </TableCell>
                <TableCell>{r.type}</TableCell>
                <TableCell>{r.latest}</TableCell>
                <TableCell>{r.count}</TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      ) : null}
    </PageFrame>
  )
}
