import { type FormEvent } from "react"
import { toast } from "sonner"

import { PageFrame } from "@/components/page-frame"
import { MenuButton } from "@/components/menu-button"
import { orgLabel } from "@/components/org-switcher"
import { Button } from "@/components/ui/button"
import { Field, FieldGroup, FieldLabel } from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { useLoad } from "@/hooks/use-load"
import { useT } from "@/i18n/i18n"
import { api } from "@/lib/api"

function CreateOrgMenu({ onCreated }: { onCreated: () => Promise<void> }) {
  const t = useT()
  async function submit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault()
    const fd = new FormData(e.currentTarget)
    const name = String(fd.get("name") || "").trim()
    const fullName = String(fd.get("full_name") || "").trim()
    if (!name) return
    try {
      await api.createOrg(name, fullName)
      toast.success(t("orgCreated"))
      e.currentTarget.reset()
      await onCreated()
    } catch (err) {
      toast.error(err instanceof Error ? err.message : t("createOrgFailed"))
    }
  }
  return (
    <MenuButton label={t("createOrg")} className="w-72">
      <form onSubmit={(e) => void submit(e)}>
        <FieldGroup>
          <Field>
            <FieldLabel htmlFor="org-name">{t("orgName")}</FieldLabel>
            <Input id="org-name" name="name" required autoComplete="off" />
          </Field>
          <Field>
            <FieldLabel htmlFor="org-full-name">{t("orgFullName")}</FieldLabel>
            <Input id="org-full-name" name="full_name" autoComplete="off" />
          </Field>
          <Button type="submit">{t("create")}</Button>
        </FieldGroup>
      </form>
    </MenuButton>
  )
}

export function AdminOrgsPage() {
  const t = useT()
  const load = useLoad(() => api.orgs(), [])
  const items = load.data?.items ?? []

  return (
    <PageFrame
      loading={load.loading && !load.data}
      error={load.error}
      empty={!load.error && Boolean(load.data) && items.length === 0}
      emptyText={t("orgEmpty")}
      skeleton="table"
      header={
        <div className="flex flex-wrap items-start justify-end gap-3">
          <CreateOrgMenu onCreated={load.reload} />
        </div>
      }
    >
      {items.length === 0 ? null : (
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>{t("orgName")}</TableHead>
              <TableHead>{t("orgFullName")}</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {items.map((row) => (
              <TableRow key={row.name}>
                <TableCell className="font-medium">{row.name}</TableCell>
                <TableCell>{orgLabel(row.name, row.full_name)}</TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      )}
    </PageFrame>
  )
}
